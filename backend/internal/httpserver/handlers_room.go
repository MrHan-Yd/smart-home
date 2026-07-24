package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/demo/smart-home/backend/internal/module/room"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	rooms, err := s.rooms.List(r.Context(), *hid)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "查询房间失败")
		return
	}
	counts, _ := s.rooms.DeviceCount(r.Context(), *hid)
	out := make([]map[string]any, 0, len(rooms))
	for _, x := range rooms {
		out = append(out, map[string]any{
			"id":          x.ID,
			"home_id":     x.HomeID,
			"name":        x.Name,
			"sort_order":  x.SortOrder,
			"ha_area_id":  x.HAAreaID,
			"device_count": counts[x.ID],
			"created_at":  x.CreatedAt,
		})
	}
	response.OK(w, map[string]any{"items": out})
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	hid := s.userHome(w, r, p.UserID)
	if hid == nil {
		return
	}
	var body struct {
		Name      string  `json:"name"`
		SortOrder *int    `json:"sort_order"`
		HAAreaID  *string `json:"ha_area_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "name 必填")
		return
	}
	x := room.Room{HomeID: *hid, Name: strings.TrimSpace(body.Name), HAAreaID: body.HAAreaID}
	if body.SortOrder != nil {
		x.SortOrder = *body.SortOrder
	}
	created, err := s.rooms.Create(r.Context(), x)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "创建失败")
		return
	}
	response.JSON(w, http.StatusCreated, 0, "ok", map[string]any{
		"id": created.ID, "name": created.Name, "sort_order": created.SortOrder,
	})
}

func (s *Server) handlePatchRoom(w http.ResponseWriter, r *http.Request) {
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
	var body room.Patch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, http.StatusBadRequest, apperr.CodeBadRequest, "body 无效")
		return
	}
	x, err := s.rooms.Patch(r.Context(), *hid, id, body)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "更新失败")
		return
	}
	if x == nil {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "房间不存在")
		return
	}
	response.OK(w, map[string]any{"id": x.ID, "name": x.Name, "sort_order": x.SortOrder})
}

func (s *Server) handleDeleteRoom(w http.ResponseWriter, r *http.Request) {
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
	ok, err := s.rooms.Delete(r.Context(), *hid, id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "删除失败")
		return
	}
	if !ok {
		response.Fail(w, http.StatusNotFound, apperr.CodeNotFound, "房间不存在")
		return
	}
	response.OK(w, map[string]any{"ok": true})
}