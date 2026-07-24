package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	p := s.requireUser(w, r)
	if p == nil {
		return
	}
	if s.hub == nil {
		response.Fail(w, http.StatusServiceUnavailable, 50300, "实时推送未启用（HA 未配置）")
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// same-origin in dev: backend :3002 ↔ frontend :5175; allow origins configured by app base
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.log.Warn("ws accept", "err", err)
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// device lookup helper for emitting DeviceView
	drepo := s.devices
	userID := p.UserID

	emit := func(entityID string, st hass.State) {
		d, err := drepo.GetByEntity(r.Context(), userID, entityID)
		if err == nil && d != nil {
			view := s.deviceView(*d, st, true, true, nil)
			msg, _ := json.Marshal(map[string]any{"type": "state_changed", "device": view})
			_ = c.Write(ctx, websocket.MessageText, msg)
		}
		// if this entity is a member of a composite device, also republish the composite view
		if compID, _ := drepo.DeviceIDByMemberEntity(r.Context(), userID, entityID); compID != uuid.Nil {
			if cd, _ := drepo.GetByID(r.Context(), userID, compID); cd != nil {
				cview := s.deviceView(*cd, hass.State{}, false, true, nil)
				cmsg, _ := json.Marshal(map[string]any{"type": "state_changed", "device": cview})
				_ = c.Write(ctx, websocket.MessageText, cmsg)
			}
		}
	}

	unsub := s.hub.Subscribe(emit)
	defer unsub()

	// ping/pong reader: any message resets deadline; "ping" → pong
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			_, msg, err := c.Read(ctx)
			if err != nil {
				cancel()
				return
			}
			var p struct{ Type string `json:"type"` }
			_ = json.Unmarshal(msg, &p)
			if p.Type == "ping" {
				pong, _ := json.Marshal(map[string]any{"type": "pong", "ts": time.Now().Unix()})
				_ = c.Write(ctx, websocket.MessageText, pong)
			}
		}
	}()

	<-ctx.Done()
	_ = c.Close(websocket.StatusNormalClosure, "")
}