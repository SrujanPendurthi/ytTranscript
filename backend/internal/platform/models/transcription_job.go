package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// JobStatus represents the lifecycle of a transcription request.
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Segments []Segment

func (s Segments) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *Segments) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch v := value.(type) {
	case string:
		return json.Unmarshal([]byte(v), s)
	case []byte:
		return json.Unmarshal(v, s)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

type TranscriptionJob struct {
	Base
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"                          json:"user_id"`
	User            User       `gorm:"foreignKey:UserID"                                 json:"-"`
	APIKeyID        *uuid.UUID `gorm:"type:uuid;index"                                   json:"api_key_id"`
	APIKey          *APIKey    `gorm:"foreignKey:APIKeyID"                               json:"-"`
	YoutubeURL      string     `gorm:"not null"                                          json:"youtube_url"`
	Status          JobStatus  `gorm:"type:varchar(20);not null;default:'pending';index"  json:"status"`
	TranscriptText  *string    `gorm:"type:text"                                         json:"transcript_text,omitempty"`
	Language        *string    `gorm:"type:varchar(10)"                                  json:"language,omitempty"`
	DurationSeconds *float64   `                                                         json:"duration_seconds,omitempty"`
	WhisperUsed     bool       `gorm:"not null;default:false"                            json:"whisper_used"`
	Source          *string    `gorm:"type:varchar(20)"                                  json:"source,omitempty"` // cache | captions | whisper
	Segments        Segments   `gorm:"type:text"                                         json:"segments,omitempty"`
	ErrorMessage    *string    `                                                         json:"error_message,omitempty"`
}
