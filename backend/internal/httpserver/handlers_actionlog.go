package httpserver

import (
	"net/http"
	"strconv"

	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
)

func (s *Server) handleListActionLogs(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	logs, err := s.alog.List(r.Context(), p.UserID, limit, offset)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询审计失败")
		return
	}
	out := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		deviceID := ""
		if l.DeviceID != nil {
			deviceID = l.DeviceID.String()
		}
		errMsg := ""
		if l.ErrorMessage != nil {
			errMsg = *l.ErrorMessage
		}
		out = append(out, map[string]any{
			"id":           l.ID,
			"device_id":    deviceID,
			"entity_id":    l.EntityID,
			"action":       l.Action,
			"success":      l.Success,
			"error_message": errMsg,
			"ha_domain":    l.HADomain,
			"ha_service":   l.HAService,
			"duration_ms":  l.DurationMS,
			"created_at":   l.CreatedAt,
		})
	}
	response.OK(w, map[string]any{"items": out})
}