package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/auth"
	"github.com/demo/smart-home/backend/internal/middleware"
	"github.com/demo/smart-home/backend/internal/module/device"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *middleware.Principal {
	p := middleware.UserFromContext(r.Context())
	if p == nil {
		response.Fail(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "未登录")
		return nil
	}
	return p
}

func (s *Server) userHome(w http.ResponseWriter, r *http.Request, userID uuid.UUID) *uuid.UUID {
	h, err := s.homes.EnsureDefault(r.Context(), userID)
	if err != nil || h == nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "加载家庭失败")
		return nil
	}
	return &h.ID
}

func (s *Server) handleDiscoverEntities(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}
	if s.hass == nil || !s.hass.Configured() {
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "Home Assistant 未配置")
		return
	}

	states, err := s.hass.GetStates(r.Context())
	if err != nil {
		s.log.Error("discover get states", "err", err)
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "Home Assistant 不可达")
		return
	}
	added, err := s.devices.EntityIDsByHome(r.Context(), *homeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询设备失败")
		return
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	domainFilter := strings.TrimSpace(r.URL.Query().Get("domain"))
	onlyNew := r.URL.Query().Get("only_new") != "false"
	onlyCtrl := r.URL.Query().Get("only_controllable") == "true"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	type item struct {
		EntityID      string   `json:"entity_id"`
		Domain        string   `json:"domain"`
		Name          string   `json:"name"`
		State         string   `json:"state"`
		Available     bool     `json:"available"`
		AlreadyAdded  bool     `json:"already_added"`
		Capabilities  []string `json:"capabilities"`
		ControlLevel  string   `json:"control_level"`
		DeviceClass   any      `json:"device_class"`
		Area          any      `json:"area"`
	}

	items := make([]item, 0)
	for _, st := range states {
		domain := hass.DomainOf(st.EntityID)
		if domain == "" || hass.IsDenylisted(domain) {
			continue
		}
		if domainFilter != "" && domain != domainFilter {
			continue
		}
		_, isAdded := added[st.EntityID]
		if onlyNew && isAdded {
			continue
		}
		caps, level := hass.InferCapabilities(domain, st.Attributes)
		if onlyCtrl && level == "read_only" {
			continue
		}
		name := hass.FriendlyName(st)
		if q != "" {
			if !strings.Contains(strings.ToLower(st.EntityID), q) && !strings.Contains(strings.ToLower(name), q) {
				continue
			}
		}
		var dc any
		if st.Attributes != nil {
			dc = st.Attributes["device_class"]
		}
		items = append(items, item{
			EntityID:     st.EntityID,
			Domain:       domain,
			Name:         name,
			State:        st.State,
			Available:    hass.Available(st),
			AlreadyAdded: isAdded,
			Capabilities: caps,
			ControlLevel: level,
			DeviceClass:  dc,
			Area:         nil,
		})
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	response.OK(w, map[string]any{
		"items":     items[start:end],
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}

	f := device.ListFilter{
		HomeID:        *homeID,
		UserID:        p.UserID,
		Q:             strings.TrimSpace(r.URL.Query().Get("q")),
		Domain:        strings.TrimSpace(r.URL.Query().Get("domain")),
		IncludeHidden: r.URL.Query().Get("include_hidden") == "true",
	}
	if rid := r.URL.Query().Get("room_id"); rid != "" {
		id, err := uuid.Parse(rid)
		if err != nil {
			response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "room_id 无效")
			return
		}
		f.RoomID = &id
	}
	if v := r.URL.Query().Get("favorite"); v == "true" || v == "false" {
		b := v == "true"
		f.Favorite = &b
	}

	devs, err := s.devices.List(r.Context(), f)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询设备失败")
		return
	}

	stateMap := map[string]hass.State{}
	if s.hass != nil && s.hass.Configured() {
		if states, err := s.hass.GetStates(r.Context()); err == nil {
			for _, st := range states {
				stateMap[st.EntityID] = st
			}
		} else {
			s.log.Warn("list devices ha states", "err", err)
		}
	}

	onlyCtrl := r.URL.Query().Get("controllable") == "true"
	views := make([]map[string]any, 0, len(devs))
	for _, d := range devs {
		st, ok := stateMap[d.EntityID]
		view := s.deviceView(d, st, ok, false)
		if onlyCtrl {
			if cl, _ := view["control_level"].(string); cl == "read_only" {
				continue
			}
		}
		views = append(views, view)
	}
	response.OK(w, map[string]any{"items": views})
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
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
	var st hass.State
	ok := false
	if s.hass != nil && s.hass.Configured() {
		if got, err := s.hass.GetState(r.Context(), d.EntityID); err == nil && got != nil {
			st = *got
			ok = true
		}
	}
	response.OK(w, s.deviceView(*d, st, ok, true))
}

func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}
	var body struct {
		EntityID string     `json:"entity_id"`
		Name     *string    `json:"name"`
		RoomID   *uuid.UUID `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.EntityID) == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "entity_id 必填")
		return
	}
	body.EntityID = strings.TrimSpace(body.EntityID)
	domain := hass.DomainOf(body.EntityID)
	if domain == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "entity_id 格式无效")
		return
	}

	if s.hass != nil && s.hass.Configured() {
		if _, err := s.hass.GetState(r.Context(), body.EntityID); err != nil {
			if errors.Is(err, hass.ErrEntityNotFound) {
				response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "HA 中不存在该实体")
				return
			}
			response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "校验 HA 实体失败")
			return
		}
	}

	d := device.Device{
		HomeID:   *homeID,
		UserID:   p.UserID,
		EntityID: body.EntityID,
		Domain:   domain,
		Name:     body.Name,
		RoomID:   body.RoomID,
	}
	created, err := s.devices.Create(r.Context(), d)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, apperr.CodeConflict, "设备已添加")
			return
		}
		s.log.Error("create device", "err", err)
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "创建失败")
		return
	}
	var st hass.State
	ok := false
	if s.hass != nil && s.hass.Configured() {
		if got, err := s.hass.GetState(r.Context(), created.EntityID); err == nil && got != nil {
			st = *got
			ok = true
		}
	}
	response.JSON(w, http.StatusCreated, 0, "ok", s.deviceView(*created, st, ok, true))
}

func (s *Server) handleBatchCreateDevices(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}
	var body struct {
		EntityIDs []string `json:"entity_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.EntityIDs) == 0 {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "entity_ids 必填")
		return
	}

	created := make([]map[string]any, 0)
	skipped := make([]map[string]string, 0)
	for _, eid := range body.EntityIDs {
		eid = strings.TrimSpace(eid)
		domain := hass.DomainOf(eid)
		if domain == "" {
			skipped = append(skipped, map[string]string{"entity_id": eid, "reason": "invalid"})
			continue
		}
		d, err := s.devices.Create(r.Context(), device.Device{
			HomeID: *homeID, UserID: p.UserID, EntityID: eid, Domain: domain,
		})
		if err != nil {
			if isUniqueViolation(err) {
				skipped = append(skipped, map[string]string{"entity_id": eid, "reason": "already_added"})
				continue
			}
			skipped = append(skipped, map[string]string{"entity_id": eid, "reason": "error"})
			continue
		}
		created = append(created, map[string]any{
			"id": d.ID, "entity_id": d.EntityID, "domain": d.Domain,
		})
	}
	response.OK(w, map[string]any{"created": created, "skipped": skipped})
}

func (s *Server) handlePatchDevice(w http.ResponseWriter, r *http.Request) {
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
		RoomID    *uuid.UUID `json:"room_id"`
		Favorite  *bool      `json:"favorite"`
		Hidden    *bool      `json:"hidden"`
		SortOrder *int       `json:"sort_order"`
		Icon      *string    `json:"icon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "body 无效")
		return
	}
	patch := device.Patch{
		Name: body.Name, Favorite: body.Favorite, Hidden: body.Hidden,
		SortOrder: body.SortOrder, Icon: body.Icon,
	}
	if body.RoomID != nil {
		rid := body.RoomID
		patch.RoomID = &rid
	}

	d, err := s.devices.Patch(r.Context(), p.UserID, id, patch)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "更新失败")
		return
	}
	if d == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "设备不存在")
		return
	}
	var st hass.State
	ok := false
	if s.hass != nil && s.hass.Configured() {
		if got, err := s.hass.GetState(r.Context(), d.EntityID); err == nil && got != nil {
			st = *got
			ok = true
		}
	}
	response.OK(w, s.deviceView(*d, st, ok, true))
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "id 无效")
		return
	}
	ok, err := s.devices.Delete(r.Context(), p.UserID, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "删除失败")
		return
	}
	if !ok {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "设备不存在")
		return
	}
	response.OK(w, map[string]any{"ok": true})
}

func (s *Server) handleDeviceAction(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
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
	if s.hass == nil || !s.hass.Configured() {
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "Home Assistant 未配置")
		return
	}

	var body struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Action) == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "action 必填")
		return
	}

	// Idempotency: same key within TTL returns the prior response once.
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey != "" {
		rec, hit, err := s.sessions.TakeIdempotency(r.Context(), p.SID, idemKey)
		if err != nil {
			s.log.Warn("idem take", "err", err)
		} else if hit {
			if rec == nil {
				response.Fail(w, http.StatusConflict, apperr.CodeConflict, "请求处理中，请稍后重试")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Idempotent-Replay", "true")
			w.WriteHeader(rec.Status)
			_, _ = w.Write(rec.Body)
			return
		}
	}

	_, level := hass.InferCapabilities(d.Domain, nil)
	// refine level from live state if available
	if st, err := s.hass.GetState(r.Context(), d.EntityID); err == nil && st != nil {
		_, level = hass.InferCapabilities(d.Domain, st.Attributes)
		if !hass.Available(*st) {
			if idemKey != "" {
				s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
			}
			response.Fail(w, http.StatusConflict, apperr.CodeConflict, "设备不可用")
			return
		}
	}
	if !hass.ActionAllowed(d.Domain, body.Action, level) {
		if idemKey != "" {
			s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
		}
		response.Fail(w, http.StatusBadRequest, apperr.CodeUnsupported, "不支持的 action 或设备只读")
		return
	}

	svcDomain, service, data, err := hass.ResolveAction(d.Domain, body.Action, body.Params, d.EntityID)
	if err != nil {
		if idemKey != "" {
			s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
		}
		response.Fail(w, http.StatusBadRequest, apperr.CodeUnsupported, err.Error())
		return
	}
	if err := s.hass.CallService(r.Context(), svcDomain, service, data); err != nil {
		s.log.Error("call service", "err", err, "entity", d.EntityID, "service", service)
		if idemKey != "" {
			s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
		}
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "控制失败: "+err.Error())
		return
	}

	st, err := s.hass.GetState(r.Context(), d.EntityID)
	if err != nil || st == nil {
		if idemKey != "" {
			_ = s.sessions.FinishIdempotency(r.Context(), p.SID, idemKey, auth.IdempotencyRecord{
				Status: http.StatusOK,
				Body:   mustJSON(0, "ok", map[string]any{"ok": true, "entity_id": d.EntityID}),
			})
		}
		response.OK(w, map[string]any{"ok": true, "entity_id": d.EntityID})
		return
	}
	view := s.deviceView(*d, *st, true, true)
	if idemKey != "" {
		_ = s.sessions.FinishIdempotency(r.Context(), p.SID, idemKey, auth.IdempotencyRecord{
			Status: http.StatusOK,
			Body:   mustJSON(0, "ok", view),
		})
	}
	response.OK(w, view)
}

func (s *Server) deviceView(d device.Device, st hass.State, hasState, fullAttrs bool) map[string]any {
	name := d.EntityID
	if d.Name != nil && *d.Name != "" {
		name = *d.Name
	} else if hasState {
		name = hass.FriendlyName(st)
	}
	domain := d.Domain
	state := ""
	available := false
	var attrs map[string]any
	var caps []string
	level := "read_only"
	if hasState {
		state = st.State
		available = hass.Available(st)
		attrs = st.Attributes
		caps, level = hass.InferCapabilities(domain, attrs)
	} else {
		caps, level = hass.InferCapabilities(domain, nil)
	}
	primary := hass.PrimaryDisplay(domain, state, attrs)
	view := map[string]any{
		"id":              d.ID,
		"entity_id":       d.EntityID,
		"domain":          domain,
		"name":            name,
		"room_id":         d.RoomID,
		"room_name":       nil,
		"favorite":        d.Favorite,
		"hidden":          d.Hidden,
		"state":           state,
		"available":       available,
		"primary_display": primary,
		"capabilities":    caps,
		"control_level":   level,
	}
	if fullAttrs && attrs != nil {
		view["attributes"] = attrs
	} else if attrs != nil {
		// summary
		sum := map[string]any{}
		for _, k := range []string{"brightness", "color_temp_kelvin", "hs_color", "friendly_name", "unit_of_measurement", "device_class"} {
			if v, ok := attrs[k]; ok {
				sum[k] = v
			}
		}
		view["attributes"] = sum
	}
	if hasState {
		view["ha"] = map[string]any{
			"last_changed": st.LastChanged,
			"last_updated": st.LastUpdated,
		}
	}
	return view
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique")
}

// mustJSON serializes a response.Body payload for caching.
func mustJSON(code int, message string, data any) json.RawMessage {
	b, err := json.Marshal(response.Body{Code: code, Message: message, Data: data})
	if err != nil {
		return json.RawMessage(`{"code":50000,"message":"idem encode failed"}`)
	}
	return b
}
