package hainstance

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Instance struct {
	ID            uuid.UUID  `json:"id"`
	HomeID        uuid.UUID  `json:"home_id"`
	Name          string     `json:"name"`
	BaseURL       string     `json:"base_url"`
	TokenEncrypted string    `json:"-"`
	IsActive      bool       `json:"is_active"`
	LastOkAt      *time.Time `json:"last_ok_at"`
	LastError     *string    `json:"last_error"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Repo struct {
	db  *pgxpool.Pool
	key []byte
}

func NewRepo(db *pgxpool.Pool, encKey []byte) *Repo {
	return &Repo{db: db, key: encKey}
}

func (r *Repo) List(ctx context.Context, homeID uuid.UUID) ([]Instance, error) {
	const q = `
SELECT id, home_id, name, base_url, token_encrypted, is_active, last_ok_at, last_error, created_at, updated_at
FROM ha_instances WHERE home_id = $1 ORDER BY is_active DESC, name ASC`
	rows, err := r.db.Query(ctx, q, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Instance, 0)
	for rows.Next() {
		var x Instance
		if err := scanInstance(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repo) Active(ctx context.Context, homeID uuid.UUID) (*Instance, error) {
	const q = `
SELECT id, home_id, name, base_url, token_encrypted, is_active, last_ok_at, last_error, created_at, updated_at
FROM ha_instances WHERE home_id = $1 AND is_active = true LIMIT 1`
	row := r.db.QueryRow(ctx, q, homeID)
	var x Instance
	if err := scanInstance(row, &x); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &x, nil
}

// PlainToken decrypts the stored token; returns "" if key unset.
func (r *Repo) PlainToken(in *Instance) (string, error) {
	if in == nil || in.TokenEncrypted == "" {
		return "", nil
	}
	if len(r.key) == 0 {
		return "", errors.New("encryption key unset")
	}
	return crypto.Unbox(r.key, in.TokenEncrypted)
}

func (r *Repo) Create(ctx context.Context, homeID uuid.UUID, name, baseURL, token string) (*Instance, error) {
	if len(r.key) == 0 {
		return nil, errors.New("encryption key unset")
	}
	enc, err := crypto.Box(r.key, token)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	const q = `
INSERT INTO ha_instances (id, home_id, name, base_url, token_encrypted)
VALUES ($1,$2,$3,$4,$5)
RETURNING id, home_id, name, base_url, token_encrypted, is_active, last_ok_at, last_error, created_at, updated_at`
	row := r.db.QueryRow(ctx, q, id, homeID, strings.TrimSpace(name), strings.TrimRight(baseURL, "/"), enc)
	var x Instance
	if err := scanInstance(row, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *Repo) Update(ctx context.Context, homeID, id uuid.UUID, baseURL *string, token *string) (*Instance, error) {
	cur, err := r.getByID(ctx, homeID, id)
	if err != nil || cur == nil {
		return cur, err
	}
	newBase := cur.BaseURL
	if baseURL != nil {
		newBase = strings.TrimRight(*baseURL, "/")
	}
	if token != nil && *token != "" {
		if len(r.key) == 0 {
			return nil, errors.New("encryption key unset")
		}
		enc, err := crypto.Box(r.key, *token)
		if err != nil {
			return nil, err
		}
		cur.TokenEncrypted = enc
	}
	const q = `
UPDATE ha_instances SET base_url=$3, token_encrypted=$4, updated_at=now()
WHERE id=$1 AND home_id=$2
RETURNING id, home_id, name, base_url, token_encrypted, is_active, last_ok_at, last_error, created_at, updated_at`
	row := r.db.QueryRow(ctx, q, id, homeID, newBase, cur.TokenEncrypted)
	var x Instance
	if err := scanInstance(row, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *Repo) Delete(ctx context.Context, homeID, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM ha_instances WHERE id = $1 AND home_id = $2`
	tag, err := r.db.Exec(ctx, q, id, homeID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Activate marks the given instance as the sole active one for its home.
// All other instances of the same home are deactivated in the same tx; the
// idx_ha_instances_active unique partial index guarantees at most one active.
func (r *Repo) Activate(ctx context.Context, homeID, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE ha_instances SET is_active = false WHERE home_id = $1`, homeID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE ha_instances SET is_active = true, updated_at = now() WHERE id = $1 AND home_id = $2`, id, homeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("实例不存在")
	}
	return tx.Commit(ctx)
}

// MarkResult records probe outcome.
func (r *Repo) MarkResult(ctx context.Context, homeID, id uuid.UUID, ok bool, errMsg string) error {
	var lastOk *time.Time
	if ok {
		now := time.Now()
		lastOk = &now
	}
	var errPtr *string
	if !ok && errMsg != "" {
		errPtr = &errMsg
	}
	const q = `UPDATE ha_instances SET last_ok_at=$3, last_error=$4, updated_at=now() WHERE id=$1 AND home_id=$2`
	_, err := r.db.Exec(ctx, q, id, homeID, lastOk, errPtr)
	return err
}

func (r *Repo) getByID(ctx context.Context, homeID, id uuid.UUID) (*Instance, error) {
	const q = `
SELECT id, home_id, name, base_url, token_encrypted, is_active, last_ok_at, last_error, created_at, updated_at
FROM ha_instances WHERE id = $1 AND home_id = $2`
	row := r.db.QueryRow(ctx, q, id, homeID)
	var x Instance
	if err := scanInstance(row, &x); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &x, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanInstance(row scannable, x *Instance) error {
	return row.Scan(
		&x.ID, &x.HomeID, &x.Name, &x.BaseURL, &x.TokenEncrypted,
		&x.IsActive, &x.LastOkAt, &x.LastError, &x.CreatedAt, &x.UpdatedAt,
	)
}