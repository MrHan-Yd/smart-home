package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/module/analytics"
	"github.com/demo/smart-home/backend/internal/module/device"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

// handleAnalyticsSummary aggregates missing days then sums KPIs over the range.
func (s *Server) handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	days := parseRange(r, 7)
	start := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	end := time.Now().UTC().Truncate(24 * time.Hour)

	devs, err := s.devices.List(r.Context(), device.ListFilter{UserID: p.UserID, IncludeHidden: true})
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询设备失败")
		return
	}
	// lazy aggregate each day for each device
	ids := make([]uuid.UUID, 0, len(devs))
	for _, d := range devs {
		ids = append(ids, d.ID)
		for t := start; !t.After(end); t = t.AddDate(0, 0, 1) {
			_, _ = s.anal.AggregateDay(r.Context(), p.UserID, d.ID, d.EntityID, t.Format("2006-01-02"))
		}
	}
	stats, err := s.anal.StatsRange(r.Context(), p.UserID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "统计失败")
		return
	}

	var activationCount int
	var runtimeSec int64
	var energySum *float64
	energyHas := false
	var eAcc float64
	for _, st := range stats {
		activationCount += st.OnCount
		runtimeSec += st.OnDurationSec
		if st.EnergyKWh != nil {
			eAcc += *st.EnergyKWh
			energyHas = true
		}
	}
	if energyHas {
		energySum = &eAcc
	}

	// environment: average temperature/humidity from sensors
	avgTemp, avgHum := s.envAvg(r.Context(), devs, start, end)

	// live on/online counts from HA states
	onCount, onlineCount := 0, 0
	if s.hass != nil && s.hass.Configured() {
		if states, err := s.hass.GetStates(r.Context()); err == nil {
			idSet := map[string]bool{}
			for _, d := range devs {
				idSet[d.EntityID] = true
			}
			for _, st := range states {
				if !idSet[st.EntityID] {
					continue
				}
				if hass.Available(st) {
					onlineCount++
					if st.State == "on" || st.State == "open" || st.State == "playing" {
						onCount++
					}
				}
			}
		}
	}

	response.OK(w, map[string]any{
		"activation_count": activationCount,
		"runtime_hours":    float64(runtimeSec) / 3600.0,
		"energy_kwh":       energySum,
		"avg_temperature":  avgTemp,
		"avg_humidity":     avgHum,
		"on_count":         onCount,
		"online_count":     onlineCount,
	})
}

func (s *Server) handleAnalyticsRuntime(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	days := parseRange(r, 7)
	start := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	end := time.Now().UTC().Truncate(24 * time.Hour)
	s.lazyAggregate(r, p.UserID, start, end)
	stats, err := s.anal.StatsRange(r.Context(), p.UserID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "统计失败")
		return
	}
	out := make([]map[string]any, 0, len(stats))
	for _, st := range stats {
		out = append(out, map[string]any{
			"date":           st.StatDate,
			"device_id":      st.DeviceID,
			"hours":          float64(st.OnDurationSec) / 3600.0,
			"on_count":       st.OnCount,
			"energy_kwh":     st.EnergyKWh,
		})
	}
	response.OK(w, map[string]any{"items": out})
}

func (s *Server) handleAnalyticsRanking(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	days := parseRange(r, 7)
	start := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	end := time.Now().UTC().Truncate(24 * time.Hour)
	s.lazyAggregate(r, p.UserID, start, end)
	stats, err := s.anal.StatsRange(r.Context(), p.UserID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "统计失败")
		return
	}
	// sum per device
	type agg struct {
		hours float64
		count int
	}
	m := map[uuid.UUID]*agg{}
	for _, st := range stats {
		a, ok := m[st.DeviceID]
		if !ok {
			a = &agg{}
			m[st.DeviceID] = a
		}
		a.hours += float64(st.OnDurationSec) / 3600.0
		a.count += st.OnCount
	}
	devs, _ := s.devices.List(r.Context(), device.ListFilter{UserID: p.UserID, IncludeHidden: true})
	nameByID := map[uuid.UUID]string{}
	for _, d := range devs {
		n := d.EntityID
		if d.Name != nil && *d.Name != "" {
			n = *d.Name
		}
		nameByID[d.ID] = n
	}
	ranked := make([]map[string]any, 0, len(m))
	for id, a := range m {
		ranked = append(ranked, map[string]any{
			"device_id":   id,
			"name":        nameByID[id],
			"hours":       a.hours,
			"on_count":    a.count,
		})
	}
	// sort by hours desc
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j]["hours"].(float64) > ranked[i]["hours"].(float64) {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > len(ranked) {
		limit = len(ranked)
	}
	response.OK(w, map[string]any{"items": ranked[:limit]})
}

func (s *Server) handleAnalyticsTypeMix(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	devs, err := s.devices.List(r.Context(), device.ListFilter{UserID: p.UserID, IncludeHidden: true})
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	days := parseRange(r, 7)
	start := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	end := time.Now().UTC().Truncate(24 * time.Hour)
	s.lazyAggregate(r, p.UserID, start, end)
	stats, _ := s.anal.StatsRange(r.Context(), p.UserID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	domainByID := map[uuid.UUID]string{}
	countByDomain := map[string]int{}
	hoursByDomain := map[string]float64{}
	for _, d := range devs {
		domainByID[d.ID] = d.Domain
		countByDomain[d.Domain]++
	}
	for _, st := range stats {
		dom := domainByID[st.DeviceID]
		hoursByDomain[dom] += float64(st.OnDurationSec) / 3600.0
	}
	out := make([]map[string]any, 0, len(countByDomain))
	for dom, n := range countByDomain {
		out = append(out, map[string]any{
			"domain": dom,
			"count":  n,
			"hours":  hoursByDomain[dom],
		})
	}
	response.OK(w, map[string]any{"items": out})
}

func (s *Server) handleAnalyticsHeatmap(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	days := parseRange(r, 7)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	devs, err := s.devices.List(r.Context(), device.ListFilter{UserID: p.UserID, IncludeHidden: true})
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	ids := make([]uuid.UUID, 0, len(devs))
	for _, d := range devs {
		ids = append(ids, d.ID)
	}
	hm, err := s.anal.Heatmap(r.Context(), p.UserID, ids, start, end)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "热力统计失败")
		return
	}
	response.OK(w, map[string]any{"items": hm})
}

func (s *Server) handleAnalyticsEnvironment(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	days := parseRange(r, 7)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	devs, err := s.devices.List(r.Context(), device.ListFilter{UserID: p.UserID, IncludeHidden: true})
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	out := make([]map[string]any, 0)
	for _, d := range devs {
		if d.Domain != "sensor" {
			continue
		}
		dc := ""
		if d.EntityID == "" {
			continue
		}
		// heuristic: temperature / humidity sensors
		eid := d.EntityID
		if containsAny(eid, "temp", "humid") || dc == "temperature" || dc == "humidity" {
			series, err := s.anal.DeviceDailyAvg(r.Context(), d.ID, start, end)
			if err != nil {
				continue
			}
			out = append(out, map[string]any{
				"device_id": d.ID,
				"entity_id": eid,
				"series":    series,
			})
		}
	}
	response.OK(w, map[string]any{"items": out})
}

// lazyAggregate re-aggregates each day in [start,end] for the user's devices.
func (s *Server) lazyAggregate(r *http.Request, userID uuid.UUID, start, end time.Time) {
	devs, err := s.devices.List(r.Context(), device.ListFilter{UserID: userID, IncludeHidden: true})
	if err != nil {
		return
	}
	for t := start; !t.After(end); t = t.AddDate(0, 0, 1) {
		for _, d := range devs {
			_, _ = s.anal.AggregateDay(r.Context(), userID, d.ID, d.EntityID, t.Format("2006-01-02"))
		}
	}
}

func (s *Server) envAvg(ctx context.Context, devs []device.Device, start, end time.Time) (temp, humid *float64) {
	var tSum, hSum float64
	var tN, hN int
	for _, d := range devs {
		if d.Domain != "sensor" {
			continue
		}
		isTemp := contains(d.EntityID, "temp")
		isHum := contains(d.EntityID, "humid")
		if !isTemp && !isHum {
			continue
		}
		series, err := s.anal.DeviceDailyAvg(ctx, d.ID, start, end)
		if err != nil || len(series) == 0 {
			continue
		}
		for _, pt := range series {
			if v, ok := pt["value"].(*float64); ok && v != nil {
				if isTemp {
					tSum += *v
					tN++
				} else {
					hSum += *v
					hN++
				}
			}
		}
	}
	if tN > 0 {
		v := tSum / float64(tN)
		temp = &v
	}
	if hN > 0 {
		v := hSum / float64(hN)
		humid = &v
	}
	return temp, humid
}

func parseRange(r *http.Request, def int) int {
	v := r.URL.Query().Get("range")
	if v == "" {
		return def
	}
	if v == "7d" {
		return 7
	}
	if v == "30d" {
		return 30
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	if n > 365 {
		return 365
	}
	return n
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

var _ analytics.Sample