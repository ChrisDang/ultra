package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeployLogRepository struct {
	pool *pgxpool.Pool
}

func NewDeployLogRepository(pool *pgxpool.Pool) *DeployLogRepository {
	return &DeployLogRepository{pool: pool}
}

func (r *DeployLogRepository) Create(ctx context.Context, userID, projectName string, providers []string, environment, status string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO deploy_logs (user_id, project_name, providers, environment, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, projectName, providers, environment, status,
	)
	return err
}

func (r *DeployLogRepository) CountSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM deploy_logs WHERE user_id = $1 AND created_at >= $2`,
		userID, since,
	).Scan(&count)
	return count, err
}
