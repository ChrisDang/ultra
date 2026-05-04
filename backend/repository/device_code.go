package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/christopherdang/vibecloud/backend/model"
)

type DeviceCodeRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceCodeRepository(pool *pgxpool.Pool) *DeviceCodeRepository {
	return &DeviceCodeRepository{pool: pool}
}

func (r *DeviceCodeRepository) Create(ctx context.Context, dc *model.DeviceCode) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_codes (id, code, user_id, claimed, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		dc.ID, dc.Code, dc.UserID, dc.Claimed, dc.ExpiresAt, dc.CreatedAt,
	)
	return err
}

func (r *DeviceCodeRepository) FindByCode(ctx context.Context, code string) (*model.DeviceCode, error) {
	var dc model.DeviceCode
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, user_id, claimed, expires_at, created_at
		 FROM device_codes WHERE code = $1 AND NOT claimed AND expires_at > now()`, code,
	).Scan(&dc.ID, &dc.Code, &dc.UserID, &dc.Claimed, &dc.ExpiresAt, &dc.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &dc, err
}

func (r *DeviceCodeRepository) MarkClaimed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE device_codes SET claimed = true WHERE id = $1`, id,
	)
	return err
}

func (r *DeviceCodeRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM device_codes WHERE expires_at < now()`,
	)
	return err
}
