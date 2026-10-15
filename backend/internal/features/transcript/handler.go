package transcript

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/srujanpendurthi/transcriptflow/internal/features/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/features/billing"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/queue"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/utils"
	"gorm.io/gorm"
)

// idempotencyTTL bounds how long a client may safely retry a submission and get
// the same job back instead of creating a duplicate (and a duplicate charge).
const idempotencyTTL = 24 * time.Hour

type Handler struct {
	db    *gorm.DB
	queue *queue.Client
	rdb   *redis.Client
}

func NewHandler(db *gorm.DB, q *queue.Client, rdb *redis.Client) *Handler {
	return &Handler{db: db, queue: q, rdb: rdb}
}

type transcribeRequest struct {
	YoutubeURL string `json:"youtube_url"`
}

// Submit handles POST /api/v1/transcribe.
// Validates the URL, persists a pending job, and enqueues it for async processing.
// Returns 202 Accepted immediately — the client must poll GET /api/v1/jobs/:id.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req transcribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.YoutubeURL == "" {
		utils.WriteError(w, http.StatusBadRequest, "youtube_url required")
		return
	}
	if _, err := ExtractVideoID(req.YoutubeURL); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid youtube_url: "+err.Error())
		return
	}

	ctx := r.Context()

	// Idempotency: a repeated submission with the same key returns the original
	// job instead of creating (and billing) a duplicate.
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey != "" {
		if existing, ok := h.lookupIdempotent(ctx, userID, idemKey); ok {
			utils.WriteJSON(w, http.StatusOK, existing)
			return
		}
	}

	// Quota: reject when the plan's included minutes are exhausted and it has no
	// metered overage.
	if err := billing.CheckQuota(ctx, h.db, userID); err != nil {
		var qe *billing.QuotaError
		if errors.As(err, &qe) {
			utils.WriteError(w, qe.Status, qe.Message)
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "quota check failed")
		return
	}

	job := models.TranscriptionJob{
		UserID:     userID,
		YoutubeURL: req.YoutubeURL,
		Status:     models.JobStatusPending,
	}
	if apiKeyID, ok := auth.APIKeyIDFrom(r.Context()); ok {
		job.APIKeyID = &apiKeyID
	}

	if err := h.db.Create(&job).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	task, err := NewTranscriptionTask(job.ID)
	if err == nil {
		err = h.queue.Enqueue(task)
	}
	if err != nil {
		// Roll back the DB row so the client doesn't see a permanently-stuck pending job.
		h.db.Delete(&job)
		utils.WriteError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	if idemKey != "" {
		h.rdb.Set(ctx, idemRedisKey(userID, idemKey), job.ID.String(), idempotencyTTL)
	}

	utils.WriteJSON(w, http.StatusAccepted, job)
}

func idemRedisKey(userID uuid.UUID, key string) string {
	return fmt.Sprintf("idem:%s:%s", userID, key)
}

// lookupIdempotent returns the job previously created for (userID, key), if the
// mapping still exists in Redis and the job row is still present.
func (h *Handler) lookupIdempotent(ctx context.Context, userID uuid.UUID, key string) (models.TranscriptionJob, bool) {
	id, err := h.rdb.Get(ctx, idemRedisKey(userID, key)).Result()
	if err != nil {
		return models.TranscriptionJob{}, false
	}
	var job models.TranscriptionJob
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&job).Error; err != nil {
		return models.TranscriptionJob{}, false
	}
	return job, true
}

// ListJobs handles GET /api/v1/jobs.
// Returns the authenticated user's 20 most recent jobs.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	q := h.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(parseLimit(r.URL.Query().Get("limit")))

	// Cursor pagination: pass the created_at of the last row to fetch older jobs.
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, cursor); err == nil {
			q = q.Where("created_at < ?", t)
		}
	}

	var jobs []models.TranscriptionJob
	if err := q.Find(&jobs).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, jobs)
}

// parseLimit clamps the page size to [1, 100], defaulting to 20.
func parseLimit(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 20
	}
	if n > 100 {
		return 100
	}
	return n
}

// GetJob handles GET /api/v1/jobs/{id}.
// Enforces ownership: users can only fetch their own jobs.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var job models.TranscriptionJob
	if err := h.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.WriteError(w, http.StatusNotFound, "job not found")
		} else {
			utils.WriteError(w, http.StatusInternalServerError, "db error")
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, job)
}
