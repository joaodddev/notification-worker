package dispatcher

import (
	"context"
	"log"
	"time"

	"github.com/joaodddev/notification-worker/internal/job"
)

// Dispatcher faz polling no banco a cada `interval` e envia jobs para o pool.
type Dispatcher struct {
	repo     *job.Repository
	jobs     chan<- *job.Job // recebe o channel do Pool
	interval time.Duration
	batch    int // quantos jobs buscar por tick
}

func New(repo *job.Repository, jobs chan<- *job.Job, interval time.Duration, batch int) *Dispatcher {
	return &Dispatcher{
		repo:     repo,
		jobs:     jobs,
		interval: interval,
		batch:    batch,
	}
}

// Run inicia o loop de polling. Bloqueia até o ctx ser cancelado.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	log.Printf("[dispatcher] started (interval: %s, batch: %d)", d.interval, d.batch)

	for {
		select {
		case <-ctx.Done():
			log.Println("[dispatcher] stopped")
			return

		case <-ticker.C:
			d.dispatch(ctx)
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context) {
	pendingJobs, err := d.repo.FetchPending(ctx, d.batch)
	if err != nil {
		log.Printf("[dispatcher] error fetching jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 {
		return
	}

	log.Printf("[dispatcher] dispatching %d job(s)", len(pendingJobs))

	for _, j := range pendingJobs {
		// Envia pro channel sem bloquear — se o buffer estiver cheio, loga e segue.
		select {
		case d.jobs <- j:
		default:
			log.Printf("[dispatcher] channel full, skipping job %s", j.ID)
		}
	}
}
