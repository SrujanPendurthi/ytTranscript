package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageLog is append-only: no UpdatedAt, no soft-delete.
type UsageLog struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"               json:"id"`
	CreatedAt   time.Time  `                                           json:"created_at"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index"           json:"user_id"`
	User        User       `gorm:"foreignKey:UserID"                  json:"-"`
	APIKeyID    *uuid.UUID `gorm:"type:uuid;index"                    json:"api_key_id"`
	APIKey      *APIKey    `gorm:"foreignKey:APIKeyID"                json:"-"`
	Endpoint    string     `gorm:"not null"                           json:"endpoint"`
	Method      string     `gorm:"type:varchar(10);not null"          json:"method"`
	StatusCode  int        `gorm:"not null"                           json:"status_code"`
	CreditsUsed int        `gorm:"not null;default:0"                 json:"credits_used"`
	IPAddress   string     `                                           json:"ip_address,omitempty"`
}

func (u *UsageLog) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
