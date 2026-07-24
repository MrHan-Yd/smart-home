package device

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Device struct {
	ID        uuid.UUID       `json:"id"`
	HomeID    uuid.UUID       `json:"home_id"`
	UserID    uuid.UUID       `json:"user_id"`
	EntityID  string          `json:"entity_id"`
	Domain    string          `json:"domain"`
	Name      *string         `json:"name"`
	RoomID    *uuid.UUID      `json:"room_id"`
	Favorite  bool            `json:"favorite"`
	Hidden    bool            `json:"hidden"`
	SortOrder int             `json:"sort_order"`
	Icon      *string         `json:"icon"`
	Meta      json.RawMessage `json:"meta"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ListFilter struct {
	HomeID        uuid.UUID
	UserID        uuid.UUID
	Q             string
	Domain        string
	RoomID        *uuid.UUID
	Favorite      *bool
	IncludeHidden bool
}

type Patch struct {
	Name      *string
	RoomID    **uuid.UUID // nil=no change; ptr to nil=clear
	Favorite  *bool
	Hidden    *bool
	SortOrder *int
	Icon      *string
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context, f ListFilter) ([]Device, error) {
	q := `
SELECT id, home_id, user_id, entity_id, domain, name, room_id, favorite, hidden,
       sort_order, icon, meta, created_at, updated_at
FROM devices
WHERE home_id = $1 AND user_id = $2`
	args := []any{f.HomeID, f.UserID}
	n := 3
	if !f.IncludeHidden {
		q += ` AND hidden = false`
	}
	if f.Domain != "" {
		q += ` AND domain = $` + itoa(n)
		args = append(args, f.Domain)
		n++
	}
	if f.RoomID != nil {
		q += ` AND room_id = $` + itoa(n)
		args = append(args, *f.RoomID)
		n++
	}
	if f.Favorite != nil {
		q += ` AND favorite = $` + itoa(n)
		args = append(args, *f.Favorite)
		n++
	}
	if f.Q != "" {
		q += ` AND (entity_id ILIKE $` + itoa(n) + ` OR COALESCE(name,'') ILIKE $` + itoa(n) + `)`
		args = append(args, "%"+f.Q+"%")
		n++
	}
	q += ` ORDER BY favorite DESC, sort_order ASC, COALESCE(name, entity_id) ASC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Device, 0)
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) EntityIDsByHome(ctx context.Context, homeID uuid.UUID) (map[string]uuid.UUID, error) {
	const q = `SELECT id, entity_id FROM devices WHERE home_id = $1`
	rows, err := r.db.Query(ctx, q, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var eid string
		if err := rows.Scan(&id, &eid); err != nil {
			return nil, err
		}
		m[eid] = id
	}
	return m, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, userID, id uuid.UUID) (*Device, error) {
	const q = `
SELECT id, home_id, user_id, entity_id, domain, name, room_id, favorite, hidden,
       sort_order, icon, meta, created_at, updated_at
FROM devices WHERE id = $1 AND user_id = $2`
	row := r.db.QueryRow(ctx, q, id, userID)
	d, err := scanDevice(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) Create(ctx context.Context, d Device) (*Device, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.Meta == nil {
		d.Meta = json.RawMessage(`{}`)
	}
	const q = `
INSERT INTO devices (id, home_id, user_id, entity_id, domain, name, room_id, favorite, hidden, sort_order, icon, meta)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id, home_id, user_id, entity_id, domain, name, room_id, favorite, hidden,
          sort_order, icon, meta, created_at, updated_at`
	row := r.db.QueryRow(ctx, q,
		d.ID, d.HomeID, d.UserID, d.EntityID, d.Domain, d.Name, d.RoomID,
		d.Favorite, d.Hidden, d.SortOrder, d.Icon, d.Meta,
	)
	out, err := scanDevice(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repo) Delete(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM devices WHERE id = $1 AND user_id = $2`
	tag, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repo) Patch(ctx context.Context, userID, id uuid.UUID, p Patch) (*Device, error) {
	cur, err := r.GetByID(ctx, userID, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if p.Name != nil {
		cur.Name = p.Name
	}
	if p.RoomID != nil {
		cur.RoomID = *p.RoomID
	}
	if p.Favorite != nil {
		cur.Favorite = *p.Favorite
	}
	if p.Hidden != nil {
		cur.Hidden = *p.Hidden
	}
	if p.SortOrder != nil {
		cur.SortOrder = *p.SortOrder
	}
	if p.Icon != nil {
		cur.Icon = p.Icon
	}
	const q = `
UPDATE devices SET name=$3, room_id=$4, favorite=$5, hidden=$6, sort_order=$7, icon=$8, updated_at=now()
WHERE id=$1 AND user_id=$2
RETURNING id, home_id, user_id, entity_id, domain, name, room_id, favorite, hidden,
          sort_order, icon, meta, created_at, updated_at`
	row := r.db.QueryRow(ctx, q, id, userID, cur.Name, cur.RoomID, cur.Favorite, cur.Hidden, cur.SortOrder, cur.Icon)
	out, err := scanDevice(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDevice(row scannable) (Device, error) {
	var d Device
	var meta []byte
	err := row.Scan(
		&d.ID, &d.HomeID, &d.UserID, &d.EntityID, &d.Domain, &d.Name, &d.RoomID,
		&d.Favorite, &d.Hidden, &d.SortOrder, &d.Icon, &meta, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return d, err
	}
	if meta == nil {
		meta = []byte(`{}`)
	}
	d.Meta = meta
	return d, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
