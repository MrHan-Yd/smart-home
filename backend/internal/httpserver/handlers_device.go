package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/auth"
	"github.com/demo/smart-home/backend/internal/middleware"
	"github.com/demo/smart-home/backend/internal/module/actionlog"
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
	// also mark composite member entity_ids as already_added (so they don't show as addable)
	if home, _ := s.homes.EnsureDefault(r.Context(), p.UserID); home != nil {
		owner, _ := s.homes.Owner(r.Context(), home.ID)
		if owner != uuid.Nil {
			if memberEIDs, err := s.devices.EntityIDsByUserWithMembers(r.Context(), owner); err == nil {
				for _, eid := range memberEIDs {
					if _, ok := added[eid]; !ok {
						added[eid] = uuid.Nil
					}
				}
			}
		}
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

	roomNames, _ := s.rooms.NameByID(r.Context(), *homeID)

	onlyCtrl := r.URL.Query().Get("controllable") == "true"
	views := make([]map[string]any, 0, len(devs))
	for _, d := range devs {
		st, ok := stateMap[d.EntityID]
		view := s.deviceView(d, st, ok, false, roomNames)
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
	response.OK(w, s.deviceView(*d, st, ok, true, nil))
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
	response.JSON(w, http.StatusCreated, 0, "ok", s.deviceView(*created, st, ok, true, nil))
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

// handleCreateCompositeDevice creates one device row bound to multiple entities.
// The first entity (or primary_entity_id) becomes the device's primary entity
// and domain; meta.kind = "composite"; device_members stores all bindings.
func (s *Server) handleCreateCompositeDevice(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	homeID := s.userHome(w, r, p.UserID)
	if homeID == nil {
		return
	}
	var body struct {
		EntityIDs       []string  `json:"entity_ids"`
		PrimaryEntityID string    `json:"primary_entity_id"`
		Name            *string   `json:"name"`
		RoomID          *uuid.UUID `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.EntityIDs) < 2 {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "至少选择 2 个实体")
		return
	}
	// normalize + validate
	seen := map[string]bool{}
	ids := make([]string, 0, len(body.EntityIDs))
	for _, eid := range body.EntityIDs {
		eid = strings.TrimSpace(eid)
		if hass.DomainOf(eid) == "" || seen[eid] {
			continue
		}
		seen[eid] = true
		ids = append(ids, eid)
	}
	if len(ids) < 2 {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "至少选择 2 个有效实体")
		return
	}
	primary := strings.TrimSpace(body.PrimaryEntityID)
	if primary == "" {
		primary = ids[0]
	}
	if !seen[primary] {
		primary = ids[0]
	}
	domain := hass.DomainOf(primary)

	// verify each entity exists in HA (if configured)
	if s.hass != nil && s.hass.Configured() {
		for _, eid := range ids {
			if _, err := s.hass.GetState(r.Context(), eid); err != nil {
				if errors.Is(err, hass.ErrEntityNotFound) {
					response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "HA 中不存在实体: "+eid)
					return
				}
				response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "校验 HA 实体失败")
				return
			}
		}
	}

	meta, _ := json.Marshal(device.DeviceMeta{Kind: "composite", EntityIDs: ids})
	d := device.Device{
		HomeID:   *homeID,
		UserID:   p.UserID,
		EntityID: primary,
		Domain:   domain,
		Name:     body.Name,
		RoomID:   body.RoomID,
		Meta:     meta,
	}
	created, err := s.devices.Create(r.Context(), d)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, apperr.CodeConflict, "主实体已添加为独立设备")
			return
		}
		s.log.Error("create composite device", "err", err)
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "创建失败")
		return
	}
	if err := s.devices.SetMembers(r.Context(), created.ID, primary, ids); err != nil {
		s.log.Error("set members", "err", err)
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "写入成员失败")
		return
	}
	s.refreshHubTracking(context.Background(), *homeID)

	response.JSON(w, http.StatusCreated, 0, "ok", s.deviceView(*created, hass.State{}, false, true, nil))
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
	response.OK(w, s.deviceView(*d, st, ok, true, nil))
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

	// composite device: apply the same action to each member entity.
	if _, kind := deviceMetaKind(d.Meta); kind == "composite" {
		memberIDs, _ := s.devices.EntityIDsByDevice(r.Context(), d.ID)
		if len(memberIDs) == 0 {
			memberIDs = []string{d.EntityID}
		}
		start := time.Now()
		var firstErr error
		for _, eid := range memberIDs {
			mDomain := hass.DomainOf(eid)
			_, mLevel := hass.InferCapabilities(mDomain, nil)
			if !hass.ActionAllowed(mDomain, body.Action, mLevel) {
				continue
			}
			mSvc, mService, mData, mErr := hass.ResolveAction(mDomain, body.Action, body.Params, eid)
			if mErr != nil {
				if firstErr == nil {
					firstErr = mErr
				}
				continue
			}
			if e := s.hass.CallService(r.Context(), mSvc, mService, mData); e != nil {
				if firstErr == nil {
					firstErr = e
				}
				s.log.Warn("composite call service", "err", e, "entity", eid)
			}
		}
		dur := int(time.Since(start).Milliseconds())
		if firstErr != nil {
			if idemKey != "" {
				s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
			}
			msg := firstErr.Error()
			s.audit(r.Context(), p, d, body.Params, svcDomain, service, body.Action, dur, false, &msg)
			response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "控制失败: "+msg)
			return
		}
		s.audit(r.Context(), p, d, body.Params, svcDomain, service, body.Action, dur, true, nil)
		view := s.deviceView(*d, hass.State{}, false, true, nil)
		if idemKey != "" {
			_ = s.sessions.FinishIdempotency(r.Context(), p.SID, idemKey, auth.IdempotencyRecord{
				Status: http.StatusOK,
				Body:   mustJSON(0, "ok", view),
			})
		}
		response.OK(w, view)
		return
	}

	// execute with audit timing
	start := time.Now()
	callErr := s.hass.CallService(r.Context(), svcDomain, service, data)
	dur := int(time.Since(start).Milliseconds())

	if callErr != nil {
		s.log.Error("call service", "err", callErr, "entity", d.EntityID, "service", service)
		if idemKey != "" {
			s.sessions.ReleaseIdempotency(r.Context(), p.SID, idemKey)
		}
		msg := callErr.Error()
		s.audit(r.Context(), p, d, body.Params, svcDomain, service, body.Action, dur, false, &msg)
		response.Fail(w, http.StatusBadGateway, apperr.CodeHA, "控制失败: "+msg)
		return
	}
	s.audit(r.Context(), p, d, body.Params, svcDomain, service, body.Action, dur, true, nil)

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
	view := s.deviceView(*d, *st, true, true, nil)
	if idemKey != "" {
		_ = s.sessions.FinishIdempotency(r.Context(), p.SID, idemKey, auth.IdempotencyRecord{
			Status: http.StatusOK,
			Body:   mustJSON(0, "ok", view),
		})
	}
	response.OK(w, view)
}

func (s *Server) deviceView(d device.Device, st hass.State, hasState, fullAttrs bool, roomNames map[uuid.UUID]string) map[string]any {
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
	var roomName any
	if d.RoomID != nil && roomNames != nil {
		if n, ok := roomNames[*d.RoomID]; ok {
			roomName = n
		}
	}
	view := map[string]any{
		"id":              d.ID,
		"entity_id":       d.EntityID,
		"domain":          domain,
		"name":            name,
		"room_id":         d.RoomID,
		"room_name":       roomName,
		"favorite":        d.Favorite,
		"hidden":          d.Hidden,
		"state":           state,
		"available":       available,
		"primary_display": primary,
		"capabilities":    caps,
		"control_level":   level,
	}
	// composite device: expose member entity_ids + live member states
	if _, kind := deviceMetaKind(d.Meta); kind == "composite" {
		memberIDs, _ := s.devices.EntityIDsByDevice(context.Background(), d.ID)
		view["entity_ids"] = memberIDs
		view["meta"] = map[string]any{"kind": "composite"}
		members := s.compositeMembersView(memberIDs)
		view["members"] = members
		// aggregate: any member on → on
		aggState := "off"
		aggAvail := false
		for _, m := range members {
			if a, ok := m["available"].(bool); ok && a {
				aggAvail = true
			}
			if ms, ok := m["state"].(string); ok && ms == "on" {
				aggState = "on"
			}
		}
		view["state"] = aggState
		view["available"] = aggAvail
		if aggAvail {
			if aggState == "on" {
				view["primary_display"] = "运行中"
			} else {
				view["primary_display"] = "已关闭"
			}
		}
		view["control_level"] = "full"
		if !capsContains(caps, "on_off") {
			view["capabilities"] = append(caps, "on_off")
		}
	}
	if fullAttrs && attrs != nil {
		view["attributes"] = attrs
	} else if attrs != nil {
		// summary: keep card-level fields small but include control-relevant ones
		sum := map[string]any{}
		for _, k := range []string{
			"brightness", "color_temp", "color_temp_kelvin", "hs_color",
			"friendly_name", "unit_of_measurement", "device_class",
			"position", "current_temperature", "temperature", "hvac_modes", "hvac_mode",
		} {
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

// compositeMembersView looks up live states for each member entity_id.
func (s *Server) compositeMembersView(memberIDs []string) []map[string]any {
	out := make([]map[string]any, 0, len(memberIDs))
	if s.hass == nil || !s.hass.Configured() || len(memberIDs) == 0 {
		for _, eid := range memberIDs {
			out = append(out, map[string]any{"entity_id": eid, "state": "", "available": false})
		}
		return out
	}
	states, err := s.hass.GetStates(context.Background())
	if err != nil {
		for _, eid := range memberIDs {
			out = append(out, map[string]any{"entity_id": eid, "state": "", "available": false})
		}
		return out
	}
	byID := map[string]hass.State{}
	for _, st := range states {
		byID[st.EntityID] = st
	}
	for _, eid := range memberIDs {
		st, ok := byID[eid]
		if !ok {
			out = append(out, map[string]any{"entity_id": eid, "state": "", "available": false})
			continue
		}
		out = append(out, map[string]any{
			"entity_id": eid,
			"state":     st.State,
			"available": hass.Available(st),
		})
	}
	return out
}

// deviceMetaKind extracts meta.kind from the JSONB meta blob.
func deviceMetaKind(meta json.RawMessage) (m device.DeviceMeta, kind string) {
	if len(meta) == 0 {
		return device.DeviceMeta{}, ""
	}
	_ = json.Unmarshal(meta, &m)
	return m, m.Kind
}

func capsContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// audit writes a device_action_logs row; failures are non-fatal.
// params is the request params (already without entity_id), recorded as-is.
func (s *Server) audit(ctx context.Context, p *middleware.Principal, d *device.Device, params map[string]any, haDomain, haService, action string, durMS int, success bool, errMsg *string) {
	if s.alog == nil {
		return
	}
	homeID := uuid.Nil
	if h, err := s.homes.EnsureDefault(ctx, p.UserID); err == nil && h != nil {
		homeID = h.ID
	}
	if params == nil {
		params = map[string]any{}
	}
	raw, _ := json.Marshal(params)
	l := actionlog.Log{
		UserID:   p.UserID,
		HomeID:   homeID,
		DeviceID: &d.ID,
		EntityID: d.EntityID,
		Action:   action,
		Params:   raw,
		Success:  success,
		HADomain: haDomain,
		HAService: haService,
		DurationMS: durMS,
	}
	if errMsg != nil {
		l.ErrorMessage = errMsg
	}
	if err := s.alog.Insert(ctx, l); err != nil {
		s.log.Warn("audit insert", "err", err)
	}
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
