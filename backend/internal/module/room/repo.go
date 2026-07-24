package room

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Room struct {
	ID        uuid.UUID `json:"id"`
	HomeID    uuid.UUID `json:"home_id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	HAAreaID  *string   `json:"ha_area_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Patch struct {
	Name      *string
	SortOrder *int
	HAAreaID  *string
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context, homeID uuid.UUID) ([]Room, error) {
	const q = `
SELECT id, home_id, name, sort_order, ha_area_id, created_at, updated_at
FROM rooms WHERE home_id = $1 ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.Query(ctx, q, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Room, 0)
	for rows.Next() {
		var x Room
		if err := scanRoom(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// NameByID returns id→name for a home, used to fill room_name without N+1.
func (r *Repo) NameByID(ctx context.Context, homeID uuid.UUID) (map[uuid.UUID]string, error) {
	const q = `SELECT id, name FROM rooms WHERE home_id = $1`
	rows, err := r.db.Query(ctx, q, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		m[id] = name
	}
	return m, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, homeID, id uuid.UUID) (*Room, error) {
	const q = `
SELECT id, home_id, name, sort_order, ha_area_id, created_at, updated_at
FROM rooms WHERE id = $1 AND home_id = $2`
	row := r.db.QueryRow(ctx, q, id, homeID)
	var x Room
	if err := scanRoom(row, &x); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &x, nil
}

func (r *Repo) Create(ctx context.Context, x Room) (*Room, error) {
	if x.ID == uuid.Nil {
		x.ID = uuid.New()
	}
	const q = `
INSERT INTO rooms (id, home_id, name, sort_order, ha_area_id)
VALUES ($1,$2,$3,$4,$5)
RETURNING id, home_id, name, sort_order, ha_area_id, created_at, updated_at`
	row := r.db.QueryRow(ctx, q, x.ID, x.HomeID, x.Name, x.SortOrder, x.HAAreaID)
	var out Room
	if err := scanRoom(row, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repo) Patch(ctx context.Context, homeID, id uuid.UUID, p Patch) (*Room, error) {
	cur, err := r.GetByID(ctx, homeID, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if p.Name != nil {
		cur.Name = *p.Name
	}
	if p.SortOrder != nil {
		cur.SortOrder = *p.SortOrder
	}
	if p.HAAreaID != nil {
		cur.HAAreaID = p.HAAreaID
	}
	const q = `
UPDATE rooms SET name=$3, sort_order=$4, ha_area_id=$5, updated_at=now()
WHERE id=$1 AND home_id=$2
RETURNING id, home_id, name, sort_order, ha_area_id, created_at, updated_at`
	row := r.db.QueryRow(ctx, q, id, homeID, cur.Name, cur.SortOrder, cur.HAAreaID)
	var out Room
	if err := scanRoom(row, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repo) Delete(ctx context.Context, homeID, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM rooms WHERE id = $1 AND home_id = $2`
	tag, err := r.db.Exec(ctx, q, id, homeID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeviceCount returns room_id→count for a home, for the rooms view summary.
func (r *Repo) DeviceCount(ctx context.Context, homeID uuid.UUID) (map[uuid.UUID]int, error) {
	const q = `SELECT room_id, count(*) FROM devices WHERE home_id = $1 AND room_id IS NOT NULL GROUP BY room_id`
	rows, err := r.db.Query(ctx, q, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[uuid.UUID]int{}
	for rows.Next() {
		var rid uuid.UUID
		var n int
		if err := rows.Scan(&rid, &n); err != nil {
			return nil, err
		}
		m[rid] = n
	}
	return m, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRoom(row scannable, x *Room) error {
	return row.Scan(&x.ID, &x.HomeID, &x.Name, &x.SortOrder, &x.HAAreaID, &x.CreatedAt, &x.UpdatedAt)
}