package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// periodStart returns the start of the current billing period (calendar month, UTC).
// Phase 2 can align this to the Stripe subscription period via User.CurrentPeriodEnd.
func periodStart() time.Time {
	n := time.Now().UTC()
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// RecordUsage writes an audit row and increments the period counter for a
// completed job. Idempotency across retries is handled upstream (the worker only
// records once per job); billing minutes come from the measured duration.
func RecordUsage(ctx context.Context, db *gorm.DB, job *models.TranscriptionJob) error {
	if job.DurationSeconds == nil {
		return nil
	}
	minutes := MinutesFromSeconds(*job.DurationSeconds)
	if minutes <= 0 {
		return nil
	}

	source := ""
	if job.Source != nil {
		source = *job.Source
	}
	rec := models.UsageRecord{
		UserID:   job.UserID,
		APIKeyID: job.APIKeyID,
		JobID:    job.ID,
		Minutes:  minutes,
		Source:   source,
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		return incrementCounter(tx, job.UserID, minutes)
	})
}

// incrementCounter atomically upserts the (user, period) rollup.
func incrementCounter(tx *gorm.DB, userID uuid.UUID, minutes int) error {
	counter := models.UsageCounter{
		UserID:      userID,
		PeriodStart: periodStart(),
		MinutesUsed: minutes,
		JobsCount:   1,
		UpdatedAt:   time.Now(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "period_start"}},
		DoUpdates: clause.Assignments(map[string]any{
			"minutes_used": gorm.Expr("usage_counters.minutes_used + ?", minutes),
			"jobs_count":   gorm.Expr("usage_counters.jobs_count + 1"),
			"updated_at":   time.Now(),
		}),
	}).Create(&counter).Error
}

// QuotaError signals a rejected submission with the HTTP status to return.
type QuotaError struct {
	Status  int
	Message string
}

func (e *QuotaError) Error() string { return e.Message }

// CheckQuota returns a *QuotaError when the user has consumed their included
// minutes and their plan does not allow metered overage.
func CheckQuota(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
	var user models.User
	if err := db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	ent := EntitlementFor(user.Plan)

	used, _, err := periodUsage(ctx, db, userID)
	if err != nil {
		return err
	}
	if used >= ent.IncludedMinutes && !ent.AllowOverage {
		return &QuotaError{
			Status: http.StatusPaymentRequired,
			Message: fmt.Sprintf(
				"monthly limit of %d minutes reached on the %s plan — upgrade to keep transcribing",
				ent.IncludedMinutes, user.Plan,
			),
		}
	}
	return nil
}

// Summary is the payload for GET /v1/usage.
type Summary struct {
	Plan            string    `json:"plan"`
	IncludedMinutes int       `json:"included_minutes"`
	MinutesUsed     int       `json:"minutes_used"`
	OverageMinutes  int       `json:"overage_minutes"`
	JobsCount       int       `json:"jobs_count"`
	AllowOverage    bool      `json:"allow_overage"`
	PeriodStart     time.Time `json:"period_start"`
}

// UsageSummary reports the current period's usage for a user.
func UsageSummary(ctx context.Context, db *gorm.DB, userID uuid.UUID) (*Summary, error) {
	var user models.User
	if err := db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	ent := EntitlementFor(user.Plan)

	used, jobs, err := periodUsage(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	overage := used - ent.IncludedMinutes
	if overage < 0 {
		overage = 0
	}
	return &Summary{
		Plan:            string(user.Plan),
		IncludedMinutes: ent.IncludedMinutes,
		MinutesUsed:     used,
		OverageMinutes:  overage,
		JobsCount:       jobs,
		AllowOverage:    ent.AllowOverage,
		PeriodStart:     periodStart(),
	}, nil
}

// periodUsage reads the (minutes, jobs) rollup for the current period, treating
// a missing row as zero usage.
func periodUsage(ctx context.Context, db *gorm.DB, userID uuid.UUID) (int, int, error) {
	var c models.UsageCounter
	err := db.WithContext(ctx).
		Where("user_id = ? AND period_start = ?", userID, periodStart()).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return c.MinutesUsed, c.JobsCount, nil
}
