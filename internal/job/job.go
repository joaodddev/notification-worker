package job

import (
	"time"

	"github.com/google/uuid"
)

// Status representa o estado atual de um job.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// Job representa uma tarefa persistida no banco de dados.
type Job struct {
	ID        uuid.UUID
	Type      string
	Payload   []byte // JSONB raw
	Status    Status
	Attempts  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WebhookPayload é o payload esperado para jobs do tipo "webhook".
type WebhookPayload struct {
	URL  string         `json:"url"`
	Body map[string]any `json:"body"`
}
