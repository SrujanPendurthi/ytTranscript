package transcript

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"gorm.io/gorm"
)

// TranscriptionProcessor is the asynq task handler for transcription jobs.
// It owns the state machine and delegates extraction to the 3-layer Pipeline.
type TranscriptionProcessor struct {
	db       *gorm.DB
	pipeline *Pipeline
}

func NewTranscriptionProcessor(db *gorm.DB, p *Pipeline) *TranscriptionProcessor {
	return &TranscriptionProcessor{db: db, pipeline: p}
}

// ProcessTask is the asynq handler registered for TypeTranscriptionProcess.
// State machine:  pending → processing → completed | failed
func (p *TranscriptionProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload TranscriptionPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	var job models.TranscriptionJob
	if err := p.db.WithContext(ctx).First(&job, "id = ?", payload.JobID).Error; err != nil {
		return fmt.Errorf("fetch job %s: %w", payload.JobID, err)
	}

	if job.Status == models.JobStatusCompleted {
		log.Printf("[queue] job %s already completed, skipping", job.ID)
		return nil
	}

	if err := p.db.WithContext(ctx).Model(&job).
		Update("status", models.JobStatusProcessing).Error; err != nil {
		return fmt.Errorf("set processing %s: %w", job.ID, err)
	}
	log.Printf("[queue] processing job %s url=%s", job.ID, job.YoutubeURL)

	result, err := p.pipeline.Run(ctx, job.YoutubeURL)
	if err != nil {
		errMsg := err.Error()
		p.db.WithContext(ctx).Model(&job).Updates(map[string]any{
			"status":        models.JobStatusFailed,
			"error_message": errMsg,
		})
		return fmt.Errorf("pipeline %s: %w", job.ID, err)
	}

	segs := make(models.Segments, len(result.Segments))
	for i, s := range result.Segments {
		segs[i] = models.Segment{Start: s.Start, End: s.End, Text: s.Text}
	}
	return p.db.WithContext(ctx).Model(&job).Updates(map[string]any{
		"status":           models.JobStatusCompleted,
		"transcript_text":  result.Text,
		"language":         result.Language,
		"duration_seconds": result.Duration,
		"whisper_used":     result.WhisperUsed,
		"source":           result.Source,
		"segments":         segs,
		"error_message":    nil,
	}).Error
}
