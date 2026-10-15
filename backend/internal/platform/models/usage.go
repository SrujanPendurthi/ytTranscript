package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageRecord is an append-only ledger row written when a job completes.
// It is the audit trail behind metered billing (one row per billable job).
type UsageRecord struct {
	Base
	UserID   uuid.UUID  `gorm:"type:uuid;not null;index"        json:"user_id"`
	APIKeyID *uuid.UUID `gorm:"type:uuid;index"                 json:"api_key_id,omitempty"`
	JobID    uuid.UUID  `gorm:"type:uuid;index"                 json:"job_id"`
	Minutes  int        `gorm:"not null"                        json:"minutes"`
	Source   string     `gorm:"type:varchar(20)"                json:"source"`
}

// UsageCounter is a per-user, per-billing-period rollup used for fast quota
// checks at submit time. One row per (UserID, PeriodStart); incremented
// transactionally as jobs complete.
type UsageCounter struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"                                 json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_usage_user_period" json:"user_id"`
	PeriodStart time.Time `gorm:"not null;uniqueIndex:idx_usage_user_period"           json:"period_start"`
	MinutesUsed int       `gorm:"not null;default:0"                                   json:"minutes_used"`
	JobsCount   int       `gorm:"not null;default:0"                                   json:"jobs_count"`
	UpdatedAt   time.Time `                                                           json:"updated_at"`
}

func (c *UsageCounter) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
