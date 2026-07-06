package transcript

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srujanpendurthi/transcriptflow/internal/features/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/queue"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/utils"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	queue *queue.Client
}

func NewHandler(db *gorm.DB, q *queue.Client) *Handler {
	return &Handler{db: db, queue: q}
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

	utils.WriteJSON(w, http.StatusAccepted, job)
}

// ListJobs handles GET /api/v1/jobs.
// Returns the authenticated user's 20 most recent jobs.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var jobs []models.TranscriptionJob
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(20).
		Find(&jobs).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, jobs)
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
