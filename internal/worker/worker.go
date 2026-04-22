package worker

import (
	"context"
	"log"

	"github.com/joaodddev/notification-worker/internal/job"
	"github.com/joaodddev/notification-worker/internal/notifier"
)

// Worker consome jobs de um channel e os processa.
type Worker struct {
	id      int
	jobs    <-chan *job.Job
	repo    *job.Repository
	webhook *notifier.Webhook
}

func newWorker(id int, jobs <-chan *job.Job, repo *job.Repository, webhook *notifier.Webhook) *Worker {
	return &Worker{
		id:      id,
		jobs:    jobs,
		repo:    repo,
		webhook: webhook,
	}
}

// start inicia o loop do worker. Roda até o channel ser fechado.
func (w *Worker) start(ctx context.Context) {
	log.Printf("[worker %d] started", w.id)

	for j := range w.jobs {
		w.process(ctx, j)
	}

	log.Printf("[worker %d] stopped", w.id)
}

func (w *Worker) process(ctx context.Context, j *job.Job) {
	log.Printf("[worker %d] processing job %s (type: %s)", w.id, j.ID, j.Type)

	err := w.webhook.Send(ctx, j)
	if err != nil {
		log.Printf("[worker %d] job %s failed: %v", w.id, j.ID, err)
		_ = w.repo.UpdateStatus(ctx, j.ID, job.StatusFailed)
		return
	}

	_ = w.repo.UpdateStatus(ctx, j.ID, job.StatusDone)
	log.Printf("[worker %d] job %s done", w.id, j.ID)
}
