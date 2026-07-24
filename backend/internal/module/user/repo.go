package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID          uuid.UUID  `json:"id"`
	Sub         string     `json:"sub"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) UpsertBySub(ctx context.Context, sub, email, name string) (*User, error) {
	const q = `
INSERT INTO users (id, sub, email, name, last_login_at, updated_at)
VALUES ($1, $2, $3, $4, now(), now())
ON CONFLICT (sub) DO UPDATE SET
  email = COALESCE(EXCLUDED.email, users.email),
  name = COALESCE(NULLIF(EXCLUDED.name, ''), users.name),
  last_login_at = now(),
  updated_at = now()
RETURNING id, sub, COALESCE(email,''), COALESCE(name,''), last_login_at`
	var u User
	err := r.db.QueryRow(ctx, q, uuid.New(), sub, nullStr(email), nullStr(name)).Scan(
		&u.ID, &u.Sub, &u.Email, &u.Name, &u.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `SELECT id, sub, COALESCE(email,''), COALESCE(name,''), last_login_at FROM users WHERE id=$1`
	var u User
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Sub, &u.Email, &u.Name, &u.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
