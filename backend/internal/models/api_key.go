package models

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	Base
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index"           json:"user_id"`
	User       User       `gorm:"foreignKey:UserID"                  json:"-"`
	// KeyHash stores SHA-256(raw_key); never expose the raw key after creation.
	KeyHash    string     `gorm:"uniqueIndex;not null"               json:"-"`
	// Prefix holds the first 12 chars of the raw key for display (e.g. "tf_live_xk3j").
	Prefix     string     `gorm:"type:varchar(12);not null;index"    json:"prefix"`
	Name       string     `gorm:"not null"                           json:"name"`
	LastUsedAt *time.Time `                                           json:"last_used_at"`
}
