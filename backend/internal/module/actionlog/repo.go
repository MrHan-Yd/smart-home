package actionlog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Log struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	HomeID      uuid.UUID       `json:"home_id"`
	DeviceID    *uuid.UUID      `json:"device_id"`
	EntityID    string          `json:"entity_id"`
	Action      string          `json:"action"`
	Params      json.RawMessage `json:"params"`
	Success     bool            `json:"success"`
	ErrorMessage *string        `json:"error_message"`
	HADomain    string          `json:"ha_domain"`
	HAService   string          `json:"ha_service"`
	DurationMS  int             `json:"duration_ms"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) Insert(ctx context.Context, l Log) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.Params == nil {
		l.Params = json.RawMessage(`{}`)
	}
	const q = `
INSERT INTO device_action_logs
  (id, user_id, home_id, device_id, entity_id, action, params, success, error_message, ha_domain, ha_service, duration_ms)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.Exec(ctx, q,
		l.ID, l.UserID, l.HomeID, l.DeviceID, l.EntityID, l.Action, l.Params,
		l.Success, l.ErrorMessage, l.HADomain, l.HAService, l.DurationMS)
	return err
}

func (r *Repo) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Log, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	const q = `
SELECT id, user_id, home_id, device_id, entity_id, action, params, success, error_message, ha_domain, ha_service, duration_ms, created_at
FROM device_action_logs WHERE user_id = $1
ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Log, 0)
	for rows.Next() {
		var l Log
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.HomeID, &l.DeviceID, &l.EntityID, &l.Action, &l.Params,
			&l.Success, &l.ErrorMessage, &l.HADomain, &l.HAService, &l.DurationMS, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}