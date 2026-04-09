package persistent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/pusherr"
	"github.com/sday-kenta/backend/pkg/postgres"
)

type PushDeviceRepo struct {
	*postgres.Postgres
}

func NewPushDeviceRepo(pg *postgres.Postgres) *PushDeviceRepo {
	return &PushDeviceRepo{pg}
}

func (r *PushDeviceRepo) Upsert(ctx context.Context, device *entity.PushDevice) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PushDeviceRepo - Upsert - Begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(
		ctx,
		"DELETE FROM push_devices WHERE fcm_token = $1 AND device_id <> $2",
		device.FCMToken,
		device.DeviceID,
	); err != nil {
		return fmt.Errorf("PushDeviceRepo - Upsert - delete duplicated token: %w", err)
	}

	if err = tx.QueryRow(
		ctx,
		`
INSERT INTO push_devices (user_id, device_id, platform, fcm_token, app_version)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (device_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    fcm_token = EXCLUDED.fcm_token,
    app_version = EXCLUDED.app_version,
    updated_at = NOW(),
    last_seen_at = NOW()
RETURNING id, created_at, updated_at, last_seen_at`,
		device.UserID,
		strings.TrimSpace(device.DeviceID),
		strings.ToLower(strings.TrimSpace(device.Platform)),
		strings.TrimSpace(device.FCMToken),
		strings.TrimSpace(device.AppVersion),
	).Scan(&device.ID, &device.CreatedAt, &device.UpdatedAt, &device.LastSeenAt); err != nil {
		return fmt.Errorf("PushDeviceRepo - Upsert - QueryRow: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("PushDeviceRepo - Upsert - Commit: %w", err)
	}

	return nil
}

func (r *PushDeviceRepo) ListByUserID(ctx context.Context, userID int64) ([]entity.PushDevice, error) {
	rows, err := r.Pool.Query(
		ctx,
		`
SELECT id, user_id, device_id, platform, fcm_token, app_version, created_at, updated_at, last_seen_at
FROM push_devices
WHERE user_id = $1
ORDER BY updated_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("PushDeviceRepo - ListByUserID - Query: %w", err)
	}
	defer rows.Close()

	devices := make([]entity.PushDevice, 0)
	for rows.Next() {
		var device entity.PushDevice
		if err = rows.Scan(
			&device.ID,
			&device.UserID,
			&device.DeviceID,
			&device.Platform,
			&device.FCMToken,
			&device.AppVersion,
			&device.CreatedAt,
			&device.UpdatedAt,
			&device.LastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("PushDeviceRepo - ListByUserID - rows.Scan: %w", err)
		}
		devices = append(devices, device)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("PushDeviceRepo - ListByUserID - rows.Err: %w", err)
	}

	return devices, nil
}

func (r *PushDeviceRepo) DeleteByUserAndDeviceID(ctx context.Context, userID int64, deviceID string) error {
	var deletedID int64
	if err := r.Pool.QueryRow(
		ctx,
		"DELETE FROM push_devices WHERE user_id = $1 AND device_id = $2 RETURNING id",
		userID,
		strings.TrimSpace(deviceID),
	).Scan(&deletedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pusherr.ErrDeviceNotFound
		}
		return fmt.Errorf("PushDeviceRepo - DeleteByUserAndDeviceID - QueryRow: %w", err)
	}

	return nil
}

func (r *PushDeviceRepo) DeleteByToken(ctx context.Context, token string) error {
	if _, err := r.Pool.Exec(ctx, "DELETE FROM push_devices WHERE fcm_token = $1", strings.TrimSpace(token)); err != nil {
		return fmt.Errorf("PushDeviceRepo - DeleteByToken - Exec: %w", err)
	}

	return nil
}
