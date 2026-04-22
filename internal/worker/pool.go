package worker

import (
	"context"
	"sync"

	"github.com/joaodddev/notification-worker/internal/job"
	"github.com/joaodddev/notification-worker/internal/notifier"
)

// Pool gerencia um conjunto fixo de workers concorrentes.
type Pool struct {
	size    int
	Jobs    chan *job.Job // channel público para o dispatcher enfileirar
	repo    *job.Repository
	webhook *notifier.Webhook
	wg      sync.WaitGroup
}

// NewPool cria um Pool com `size` workers e buffer de `bufferSize` no channel.
func NewPool(size, bufferSize int, repo *job.Repository, webhook *notifier.Webhook) *Pool {
	return &Pool{
		size:    size,
		Jobs:    make(chan *job.Job, bufferSize),
		repo:    repo,
		webhook: webhook,
	}
}

// Start inicializa todos os workers. Eles rodam até o channel Jobs ser fechado.
func (p *Pool) Start(ctx context.Context) {
	for i := 1; i <= p.size; i++ {
		w := newWorker(i, p.Jobs, p.repo, p.webhook)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.start(ctx)
		}()
	}
}

// Stop fecha o channel de jobs e aguarda todos os workers terminarem.
func (p *Pool) Stop() {
	close(p.Jobs)
	p.wg.Wait()
}
