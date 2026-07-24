package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

func (s *Server) handleListHAInstances(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	list, err := s.hainst.List(r.Context(), *hid)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询失败")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, x := range list {
		out = append(out, map[string]any{
			"id":          x.ID,
			"name":        x.Name,
			"base_url_host": trimURLHost(x.BaseURL),
			"is_active":   x.IsActive,
			"last_ok_at":  x.LastOkAt,
			"last_error":  x.LastError,
			"has_token":   x.TokenEncrypted != "",
		})
	}
	response.OK(w, map[string]any{"items": out})
}

func (s *Server) handleCreateHAInstance(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	var body struct {
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		strings.TrimSpace(body.BaseURL) == "" || strings.TrimSpace(body.Token) == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "base_url 与 token 必填")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "default"
	}
	x, err := s.hainst.Create(r.Context(), *hid, name, body.BaseURL, body.Token)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "创建失败: "+err.Error())
		return
	}
	s.reloadActiveHA(r.Context(), *hid)
	response.JSON(w, http.StatusCreated, 0, "ok", map[string]any{
		"id": x.ID, "name": x.Name, "base_url_host": trimURLHost(x.BaseURL),
	})
}

func (s *Server) handleUpdateHAInstance(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	var body struct {
		BaseURL *string `json:"base_url"`
		Token   *string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	x, err := s.hainst.Update(r.Context(), *hid, id, body.BaseURL, body.Token)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "更新失败: "+err.Error())
		return
	}
	if x == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "实例不存在")
		return
	}
	s.reloadActiveHA(r.Context(), *hid)
	response.OK(w, map[string]any{"id": x.ID})
}

func (s *Server) handleDeleteHAInstance(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	ok, err := s.hainst.Delete(r.Context(), *hid, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "删除失败")
		return
	}
	if !ok {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "实例不存在")
		return
	}
	s.reloadActiveHA(r.Context(), *hid)
	response.OK(w, map[string]any{"ok": true})
}

func (s *Server) handleProbeHAInstance(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	x, err := s.hainst.Active(r.Context(), *hid)
	if err != nil || x == nil || x.ID != id {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "实例不存在")
		return
	}
	token, err := s.hainst.PlainToken(x)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeInternal, "解密 token 失败")
		return
	}
	probe := hass.NewClient(x.BaseURL, token, s.cfg.HassTimeout)
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	perr := probe.Ping(ctx)
	ok := perr == nil
	msg := ""
	if perr != nil {
		msg = perr.Error()
	}
	_ = s.hainst.MarkResult(r.Context(), *hid, id, ok, msg)
	if ok {
		response.OK(w, map[string]any{"ok": true})
		return
	}
	response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "探测失败: "+msg)
}

// handleActivateHAInstance switches the active instance for the user's home.
func (s *Server) handleActivateHAInstance(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	if err := s.hainst.Activate(r.Context(), *hid, id); err != nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "实例不存在")
		return
	}
	s.reloadActiveHA(r.Context(), *hid)
	// restart hub tracking against the new instance so WS reconnects
	s.refreshHubTracking(r.Context(), *hid)
	response.OK(w, map[string]any{"ok": true})
}

// reloadActiveHA reconfigures the shared HA client from the active DB instance,
// falling back to env config when none exists. Errors are logged, not fatal.
func (s *Server) reloadActiveHA(ctx context.Context, homeID uuid.UUID) {
	if s.hainst == nil {
		return
	}
	x, err := s.hainst.Active(ctx, homeID)
	if err != nil {
		s.log.Warn("reload ha: load active", "err", err)
		return
	}
	if x == nil {
		// no DB instance → use env config
		s.hass.Reconfigure(s.cfg.HassBaseURL, s.cfg.HassToken)
		return
	}
	token, err := s.hainst.PlainToken(x)
	if err != nil {
		s.log.Warn("reload ha: decrypt token", "err", err)
		return
	}
	s.hass.Reconfigure(x.BaseURL, token)
}