package scenario

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Scenario struct {
	ID        uuid.UUID  `json:"id"`
	HomeID    uuid.UUID  `json:"home_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Name      string     `json:"name"`
	Icon      *string    `json:"icon"`
	RoomID    *uuid.UUID `json:"room_id"`
	SortOrder int        `json:"sort_order"`
	Enabled   bool       `json:"enabled"`
	LastRunAt *time.Time `json:"last_run_at"`
	RunCount  int        `json:"run_count"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Steps     []Step     `json:"steps,omitempty"`
}

type Step struct {
	ID         uuid.UUID       `json:"id"`
	ScenarioID uuid.UUID       `json:"scenario_id"`
	SortOrder  int             `json:"sort_order"`
	DeviceID   uuid.UUID       `json:"device_id"`
	Action     string          `json:"action"`
	Params     json.RawMessage `json:"params"`
	DelayMS    int             `json:"delay_ms"`
	CreatedAt  time.Time       `json:"created_at"`
}

type StepInput struct {
	DeviceID uuid.UUID              `json:"device_id"`
	Action   string                 `json:"action"`
	Params   map[string]any        `json:"params"`
	DelayMS  int                    `json:"delay_ms"`
}

type Patch struct {
	Name      *string
	Icon      *string
	RoomID    **uuid.UUID
	SortOrder *int
	Enabled   *bool
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context, homeID uuid.UUID) ([]Scenario, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, home_id, user_id, name, icon, room_id, sort_order, enabled, last_run_at, run_count, created_at, updated_at
FROM scenarios WHERE home_id = $1 ORDER BY sort_order ASC, name ASC`, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Scenario, 0)
	for rows.Next() {
		var s Scenario
		if err := scanScenario(rows, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, userID, id uuid.UUID) (*Scenario, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, home_id, user_id, name, icon, room_id, sort_order, enabled, last_run_at, run_count, created_at, updated_at
FROM scenarios WHERE id = $1 AND user_id = $2`, id, userID)
	var s Scenario
	if err := scanScenario(row, &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	steps, err := r.stepsOf(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Steps = steps
	return &s, nil
}

func (r *Repo) Create(ctx context.Context, s Scenario, steps []StepInput) (*Scenario, error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
INSERT INTO scenarios (id, home_id, user_id, name, icon, room_id, sort_order, enabled)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		s.ID, s.HomeID, s.UserID, s.Name, s.Icon, s.RoomID, s.SortOrder, s.Enabled); err != nil {
		return nil, err
	}
	if err := writeSteps(ctx, tx, s.ID, steps); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, s.UserID, s.ID)
}

func (r *Repo) Patch(ctx context.Context, userID, id uuid.UUID, p Patch) (*Scenario, error) {
	cur, err := r.GetByID(ctx, userID, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if p.Name != nil {
		cur.Name = *p.Name
	}
	if p.Icon != nil {
		cur.Icon = p.Icon
	}
	if p.RoomID != nil {
		cur.RoomID = *p.RoomID
	}
	if p.SortOrder != nil {
		cur.SortOrder = *p.SortOrder
	}
	if p.Enabled != nil {
		cur.Enabled = *p.Enabled
	}
	if _, err := r.db.Exec(ctx, `
UPDATE scenarios SET name=$3, icon=$4, room_id=$5, sort_order=$6, enabled=$7, updated_at=now()
WHERE id=$1 AND user_id=$2`,
		id, userID, cur.Name, cur.Icon, cur.RoomID, cur.SortOrder, cur.Enabled); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, userID, id)
}

// ReplaceSteps rewrites all steps of a scenario.
func (r *Repo) ReplaceSteps(ctx context.Context, userID, id uuid.UUID, steps []StepInput) (*Scenario, error) {
	if _, err := r.GetByID(ctx, userID, id); err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM scenario_steps WHERE scenario_id = $1`, id); err != nil {
		return nil, err
	}
	if err := writeSteps(ctx, tx, id, steps); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, userID, id)
}

func (r *Repo) Delete(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM scenarios WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// MarkRun records a run outcome.
func (r *Repo) MarkRun(ctx context.Context, userID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
UPDATE scenarios SET run_count = run_count + 1, last_run_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// StepsOf returns the steps of a scenario, ordered by sort_order.
func (r *Repo) StepsOf(ctx context.Context, scenarioID uuid.UUID) ([]Step, error) {
	return r.stepsOf(ctx, scenarioID)
}

func (r *Repo) stepsOf(ctx context.Context, scenarioID uuid.UUID) ([]Step, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, scenario_id, sort_order, device_id, action, params, delay_ms, created_at
FROM scenario_steps WHERE scenario_id = $1 ORDER BY sort_order ASC`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Step, 0)
	for rows.Next() {
		var st Step
		var params []byte
		if err := rows.Scan(&st.ID, &st.ScenarioID, &st.SortOrder, &st.DeviceID, &st.Action, &params, &st.DelayMS, &st.CreatedAt); err != nil {
			return nil, err
		}
		if params == nil {
			params = []byte(`{}`)
		}
		st.Params = params
		out = append(out, st)
	}
	return out, rows.Err()
}

func writeSteps(ctx context.Context, tx pgx.Tx, scenarioID uuid.UUID, steps []StepInput) error {
	for i, st := range steps {
		var params []byte
		if st.Params == nil {
			params = []byte(`{}`)
		} else {
			b, err := json.Marshal(st.Params)
			if err != nil {
				return err
			}
			params = b
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO scenario_steps (id, scenario_id, sort_order, device_id, action, params, delay_ms)
VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			uuid.New(), scenarioID, i, st.DeviceID, st.Action, params, st.DelayMS); err != nil {
			return err
		}
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanScenario(row scannable, s *Scenario) error {
	return row.Scan(
		&s.ID, &s.HomeID, &s.UserID, &s.Name, &s.Icon, &s.RoomID,
		&s.SortOrder, &s.Enabled, &s.LastRunAt, &s.RunCount, &s.CreatedAt, &s.UpdatedAt,
	)
}