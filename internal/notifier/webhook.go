package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joaodddev/notification-worker/internal/job"
)

// Webhook é responsável por disparar notificações via HTTP POST.
type Webhook struct {
	client *http.Client
}

func NewWebhook() *Webhook {
	return &Webhook{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send decodifica o payload do job e faz o POST no URL de destino.
func (w *Webhook) Send(ctx context.Context, j *job.Job) error {
	var payload job.WebhookPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return fmt.Errorf("decode webhook payload: %w", err)
	}

	if payload.URL == "" {
		return fmt.Errorf("payload missing url")
	}

	bodyBytes, err := json.Marshal(payload.Body)
	if err != nil {
		return fmt.Errorf("marshal webhook body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, payload.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
