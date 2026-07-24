package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

func (s *Server) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, perr := uuid.Parse(r.PathValue("id"))
	if perr != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	d, err := s.devices.GetByID(r.Context(), p.UserID, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	if d == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "设备不存在")
		return
	}
	s.serveHistory(w, r, d.EntityID)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	if entityID == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "entity_id 必填")
		return
	}
	// scope: must be a device the user owns
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}
	added, err := s.devices.EntityIDsByHome(r.Context(), *homeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	if _, ok := added[entityID]; !ok {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "实体未纳入")
		return
	}
	s.serveHistory(w, r, entityID)
}

func (s *Server) serveHistory(w http.ResponseWriter, r *http.Request, entityID string) {
	if s.hass == nil || !s.hass.Configured() {
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "Home Assistant 未配置")
		return
	}

	q := r.URL.Query()
	now := time.Now().UTC()
	end := strings.TrimSpace(q.Get("end"))
	if end == "" {
		end = now.Format(time.RFC3339)
	}
	start := strings.TrimSpace(q.Get("start"))
	if start == "" {
		start = now.Add(-24 * time.Hour).Format(time.RFC3339)
	}
	significantOnly := q.Get("significant_only") != "false"

	entries, err := s.hass.History(r.Context(), entityID, start, end, significantOnly)
	if err != nil {
		s.log.Error("history", "err", err, "entity", entityID)
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "HA 历史不可用")
		return
	}

	type point struct {
		T    string   `json:"t"`
		State string  `json:"state"`
		Num  *float64 `json:"num"`
	}
	points := make([]point, 0, len(entries))
	for _, e := range entries {
		var num *float64
		if f, ok := parseFloat(e.State); ok {
			num = &f
		}
		ts := e.LastChanged
		if ts == "" {
			ts = e.LastUpdated
		}
		points = append(points, point{T: ts, State: e.State, Num: num})
	}
	response.OK(w, map[string]any{
		"entity_id": entityID,
		"points":    points,
		"count":     len(points),
	})
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "unavailable" || s == "unknown" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}