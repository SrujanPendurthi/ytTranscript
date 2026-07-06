package transcript

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/queue"
)

// TypeTranscriptionProcess is the asynq task type key for all transcription jobs.
const TypeTranscriptionProcess = "transcription:process"

// TranscriptionPayload is the JSON body stored in Redis for each queued task.
// Only the job ID is stored; the worker fetches the full record from Postgres.
type TranscriptionPayload struct {
	JobID uuid.UUID `json:"job_id"`
}

// NewTranscriptionTask builds an asynq Task with 3 retries on the transcription queue.
func NewTranscriptionTask(jobID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(TranscriptionPayload{JobID: jobID})
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return asynq.NewTask(
		TypeTranscriptionProcess,
		payload,
		asynq.MaxRetry(3),
		asynq.Queue(queue.QueueTranscription),
	), nil
}
