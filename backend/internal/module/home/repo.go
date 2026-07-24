package home

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Home struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetDefault(ctx context.Context, userID uuid.UUID) (*Home, error) {
	const q = `
SELECT id, user_id, name, is_default
FROM homes
WHERE user_id = $1
ORDER BY is_default DESC, created_at ASC
LIMIT 1`
	var h Home
	err := r.db.QueryRow(ctx, q, userID).Scan(&h.ID, &h.UserID, &h.Name, &h.IsDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *Repo) EnsureDefault(ctx context.Context, userID uuid.UUID) (*Home, error) {
	h, err := r.GetDefault(ctx, userID)
	if err != nil {
		return nil, err
	}
	if h != nil {
		return h, nil
	}
	const q = `
INSERT INTO homes (id, user_id, name, is_default)
VALUES ($1, $2, '我的家', true)
RETURNING id, user_id, name, is_default`
	var out Home
	err = r.db.QueryRow(ctx, q, uuid.New(), userID).Scan(&out.ID, &out.UserID, &out.Name, &out.IsDefault)
	if err != nil {
		// race: another request created it
		h2, err2 := r.GetDefault(ctx, userID)
		if err2 != nil {
			return nil, err
		}
		if h2 != nil {
			return h2, nil
		}
		return nil, err
	}
	return &out, nil
}
