package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository encapsula todas as queries relacionadas a jobs.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Enqueue insere um novo job com status "pending".
func (r *Repository) Enqueue(ctx context.Context, jobType string, payload []byte) (*Job, error) {
	query := `
		INSERT INTO jobs (type, payload)
		VALUES ($1, $2)
		RETURNING id, type, payload, status, attempts, created_at, updated_at
	`

	j := &Job{}
	err := r.db.QueryRow(ctx, query, jobType, payload).Scan(
		&j.ID, &j.Type, &j.Payload, &j.Status,
		&j.Attempts, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	return j, nil
}

// FetchPending busca até `limit` jobs com status "pending" e os marca como "running".
// Usa FOR UPDATE SKIP LOCKED para ser seguro com múltiplos workers concorrentes.
func (r *Repository) FetchPending(ctx context.Context, limit int) ([]*Job, error) {
	query := `
		UPDATE jobs
		SET status = 'running', attempts = attempts + 1, updated_at = now()
		WHERE id IN (
			SELECT id FROM jobs
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, type, payload, status, attempts, created_at, updated_at
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		j := &Job{}
		if err := rows.Scan(
			&j.ID, &j.Type, &j.Payload, &j.Status,
			&j.Attempts, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// UpdateStatus atualiza o status de um job pelo ID.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	query := `UPDATE jobs SET status = $1, updated_at = now() WHERE id = $2`

	_, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	return nil
}
