package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/module/scenario"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	list, err := s.scen.List(r.Context(), *hid)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	// attach steps for editor convenience
	for i := range list {
		list[i].Steps, _ = s.scen.StepsOf(r.Context(), list[i].ID) //nolint:errcheck
	}
	out := make([]map[string]any, 0, len(list))
	for _, sc := range list {
		out = append(out, scenarioView(sc))
	}
	response.OK(w, map[string]any{"items": out})
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	sc, err := s.scen.GetByID(r.Context(), p.UserID, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	if sc == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "场景不存在")
		return
	}
	response.OK(w, scenarioView(*sc))
}

func (s *Server) handleCreateScenario(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	var body struct {
		Name   string             `json:"name"`
		Icon   *string            `json:"icon"`
		RoomID *uuid.UUID         `json:"room_id"`
		Steps  []scenario.StepInput `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "name 必填")
		return
	}
	sc := scenario.Scenario{
		HomeID: *hid,
		UserID: p.UserID,
		Name:   strings.TrimSpace(body.Name),
		Icon:   body.Icon,
		RoomID: body.RoomID,
		Enabled: true,
	}
	created, err := s.scen.Create(r.Context(), sc, body.Steps)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "创建失败: "+err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, 0, "ok", scenarioView(*created))
}

func (s *Server) handlePatchScenario(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	var body struct {
		Name      *string    `json:"name"`
		Icon      *string    `json:"icon"`
		RoomID    *uuid.UUID `json:"room_id"`
		SortOrder *int       `json:"sort_order"`
		Enabled   *bool      `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	patch := scenario.Patch{
		Name: body.Name, Icon: body.Icon, SortOrder: body.SortOrder, Enabled: body.Enabled,
	}
	if body.RoomID != nil {
		rid := body.RoomID
		patch.RoomID = &rid
	}
	sc, err := s.scen.Patch(r.Context(), p.UserID, id, patch)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "更新失败")
		return
	}
	if sc == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "场景不存在")
		return
	}
	response.OK(w, scenarioView(*sc))
}

func (s *Server) handleReplaceScenarioSteps(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	var body struct {
		Steps []scenario.StepInput `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "body 无效")
		return
	}
	sc, err := s.scen.ReplaceSteps(r.Context(), p.UserID, id, body.Steps)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "更新步骤失败")
		return
	}
	if sc == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "场景不存在")
		return
	}
	response.OK(w, scenarioView(*sc))
}

func (s *Server) handleDeleteScenario(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	ok, err := s.scen.Delete(r.Context(), p.UserID, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "删除失败")
		return
	}
	if !ok {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "场景不存在")
		return
	}
	response.OK(w, map[string]any{"ok": true})
}

// handleRunScenario executes steps serially, tolerating individual failures.
func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	sc, err := s.scen.GetByID(r.Context(), p.UserID, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	if sc == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "场景不存在")
		return
	}
	if s.hass == nil || !s.hass.Configured() {
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "Home Assistant 未配置")
		return
	}

	type stepResult struct {
		DeviceID uuid.UUID `json:"device_id"`
		Action   string    `json:"action"`
		Success  bool      `json:"success"`
		Error    string    `json:"error,omitempty"`
	}
	results := make([]stepResult, 0, len(sc.Steps))

	for _, st := range sc.Steps {
		if st.DelayMS > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(time.Duration(st.DelayMS) * time.Millisecond):
			}
		}
		d, err := s.devices.GetByID(r.Context(), p.UserID, st.DeviceID)
		if err != nil || d == nil {
			results = append(results, stepResult{st.DeviceID, st.Action, false, "设备不存在"})
			continue
		}
		var params map[string]any
		_ = json.Unmarshal(st.Params, &params)
		svcDomain, service, data, rerr := hass.ResolveAction(d.Domain, st.Action, params, d.EntityID)
		if rerr != nil {
			results = append(results, stepResult{st.DeviceID, st.Action, false, rerr.Error()})
			continue
		}
		if !hass.ActionAllowed(d.Domain, st.Action, "full") {
			results = append(results, stepResult{st.DeviceID, st.Action, false, "不支持的动作"})
			continue
		}
		start := time.Now()
		cerr := s.hass.CallService(r.Context(), svcDomain, service, data)
		dur := int(time.Since(start).Milliseconds())
		if cerr != nil {
			msg := cerr.Error()
			s.audit(r.Context(), p, d, params, svcDomain, service, st.Action, dur, false, &msg)
			results = append(results, stepResult{st.DeviceID, st.Action, false, msg})
			continue
		}
		s.audit(r.Context(), p, d, params, svcDomain, service, st.Action, dur, true, nil)
		results = append(results, stepResult{st.DeviceID, st.Action, true, ""})
	}

	_ = s.scen.MarkRun(r.Context(), p.UserID, id)
	response.OK(w, map[string]any{"results": results})
}

func scenarioView(sc scenario.Scenario) map[string]any {
	steps := make([]map[string]any, 0, len(sc.Steps))
	for _, st := range sc.Steps {
		var params any
		_ = json.Unmarshal(st.Params, &params)
		steps = append(steps, map[string]any{
			"id":           st.ID,
			"scenario_id":  st.ScenarioID,
			"sort_order":   st.SortOrder,
			"device_id":    st.DeviceID,
			"action":       st.Action,
			"params":       params,
			"delay_ms":     st.DelayMS,
		})
	}
	return map[string]any{
		"id":          sc.ID,
		"home_id":     sc.HomeID,
		"user_id":     sc.UserID,
		"name":        sc.Name,
		"icon":        sc.Icon,
		"room_id":     sc.RoomID,
		"sort_order":  sc.SortOrder,
		"enabled":     sc.Enabled,
		"last_run_at": sc.LastRunAt,
		"run_count":   sc.RunCount,
		"created_at":  sc.CreatedAt,
		"updated_at":  sc.UpdatedAt,
		"steps":       steps,
	}
}