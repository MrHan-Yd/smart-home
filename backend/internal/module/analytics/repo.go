package analytics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sample is one telemetry point written from HA state_changed events.
type Sample struct {
	DeviceID   uuid.UUID
	EntityID   string
	TS         time.Time
	State      string
	NumValue   *float64
	Attributes map[string]any
}

type DailyStat struct {
	DeviceID       uuid.UUID `json:"device_id"`
	StatDate       string   `json:"stat_date"`
	OnCount        int      `json:"on_count"`
	OnDurationSec  int64    `json:"on_duration_sec"`
	EnergyKWh      *float64 `json:"energy_kwh"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// WriteSample persists a slimmed telemetry sample.
func (r *Repo) WriteSample(ctx context.Context, s Sample) error {
	var num *float64
	if s.NumValue != nil {
		v := *s.NumValue
		num = &v
	}
	var attrs []byte
	if s.Attributes != nil {
		b, _ := json.Marshal(s.Attributes)
		attrs = b
	}
	const q = `INSERT INTO telemetry_samples (device_id, entity_id, ts, state, num_value, attributes)
VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := r.db.Exec(ctx, q, s.DeviceID, s.EntityID, s.TS, s.State, num, attrs)
	return err
}

// AggregateDay computes on_count / on_duration_sec / energy for a device on a given
// date from telemetry_samples and upserts into daily_device_stats. Lazily called by
// analytics endpoints for not-yet-aggregated days.
//
// on_count: number of transitions into an "on"-like state.
// on_duration_sec: total seconds spent in on-like state (sum of gaps while on).
// energy_kwh: sum of num_value for power/energy-class sensors (kWh when unit is kWh).
func (r *Repo) AggregateDay(ctx context.Context, userID, deviceID uuid.UUID, entityID, date string) (*DailyStat, error) {
	start, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	dayStart := start
	dayEnd := start.Add(24 * time.Hour)

	// load ordered samples
	const q = `SELECT ts, state, num_value FROM telemetry_samples
WHERE device_id = $1 AND ts >= $2 AND ts < $3 ORDER BY ts ASC`
	rows, err := r.db.Query(ctx, q, deviceID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	type row struct {
		TS    time.Time
		State string
		Num   *float64
	}
	var samples []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.TS, &x.State, &x.Num); err != nil {
			rows.Close()
			return nil, err
		}
		samples = append(samples, x)
	}
	rows.Close()

	var onCount int
	var onDuration float64
	var energy float64
	hasEnergy := false
	for i, s := range samples {
		on := isOn(s.State)
		if on && (i == 0 || !isOn(samples[i-1].State)) {
			onCount++
		}
		if on && i+1 < len(samples) {
			d := samples[i+1].TS.Sub(s.TS).Seconds()
			if d > 0 {
				onDuration += d
			}
		}
		if s.Num != nil {
			energy += *s.Num
			hasEnergy = true
		}
	}

	stat := &DailyStat{
		DeviceID:      deviceID,
		StatDate:      date,
		OnCount:       onCount,
		OnDurationSec: int64(onDuration),
	}
	if hasEnergy {
		e := energy
		stat.EnergyKWh = &e
	}

	const u = `
INSERT INTO daily_device_stats (id, user_id, device_id, stat_date, on_count, on_duration_sec, energy_kwh)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (device_id, stat_date) DO UPDATE SET
  on_count = EXCLUDED.on_count,
  on_duration_sec = EXCLUDED.on_duration_sec,
  energy_kwh = EXCLUDED.energy_kwh,
  updated_at = now()`
	_, err = r.db.Exec(ctx, u, uuid.New(), userID, deviceID, date, stat.OnCount, stat.OnDurationSec, stat.EnergyKWh)
	return stat, err
}

// StatsRange returns daily stats for a user over [start,end] inclusive (date strings).
func (r *Repo) StatsRange(ctx context.Context, userID uuid.UUID, start, end string) ([]DailyStat, error) {
	const q = `SELECT device_id, to_char(stat_date,'YYYY-MM-DD'), on_count, on_duration_sec, energy_kwh
FROM daily_device_stats WHERE user_id = $1 AND stat_date >= $2 AND stat_date <= $3
ORDER BY stat_date ASC, device_id`
	rows, err := r.db.Query(ctx, q, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DailyStat, 0)
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.DeviceID, &s.StatDate, &s.OnCount, &s.OnDurationSec, &s.EnergyKWh); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeviceAvgNumeric returns per-date average of num_value for a device's entity
// (used for temperature/humidity environment trends).
func (r *Repo) DeviceDailyAvg(ctx context.Context, deviceID uuid.UUID, start, end time.Time) ([]map[string]any, error) {
	const q = `SELECT to_char(date_trunc('day', ts),'YYYY-MM-DD') AS d, avg(num_value) AS v
FROM telemetry_samples
WHERE device_id = $1 AND ts >= $2 AND ts < $3 AND num_value IS NOT NULL
GROUP BY d ORDER BY d ASC`
	rows, err := r.db.Query(ctx, q, deviceID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var d string
		var v *float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"date": d, "value": v})
	}
	return out, rows.Err()
}

// Heatmap returns on-transition counts per (entity_id, hour) over a range.
// Transitions are computed in Go from ordered samples (HA-based, no SQL function needed).
func (r *Repo) Heatmap(ctx context.Context, userID uuid.UUID, deviceIDs []uuid.UUID, start, end time.Time) ([]map[string]any, error) {
	if len(deviceIDs) == 0 {
		return []map[string]any{}, nil
	}
	const q = `SELECT entity_id, ts, state FROM telemetry_samples
WHERE device_id = ANY($1) AND ts >= $2 AND ts < $3 ORDER BY entity_id, ts ASC`
	rows, err := r.db.Query(ctx, q, deviceIDs, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type key struct {
		eid  string
		hour int
	}
	counts := map[key]int{}
	var curEID string
	var prev string
	for rows.Next() {
		var eid, state string
		var ts time.Time
		if err := rows.Scan(&eid, &ts, &state); err != nil {
			return nil, err
		}
		if eid != curEID {
			curEID = eid
			prev = ""
		}
		if isOn(state) && !isOn(prev) && prev != "" {
			counts[key{eid, ts.In(time.UTC).Hour()}]++
		}
		prev = state
	}
	out := make([]map[string]any, 0, len(counts))
	for k, c := range counts {
		out = append(out, map[string]any{"entity_id": k.eid, "hour": k.hour, "count": c})
	}
	return out, rows.Err()
}

func isOn(state string) bool {
	switch state {
	case "on", "open", "playing", "home", "active":
		return true
	}
	return false
}