package hass

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// StateChanged is a HA state_changed event payload.
type StateChanged struct {
	EntityID string `json:"entity_id"`
	NewState State  `json:"new_state"`
	OldState *State `json:"old_state"`
}

// Subscriber receives filtered state_changed events for tracked entities.
type Subscriber func(entityID string, st State)

// Hub bridges HA's websocket to local subscribers. It maintains the set of
// tracked entity_ids (the user's added devices) and only forwards matching events.
type Hub struct {
	log    *slog.Logger
	client *Client

	mu        sync.RWMutex
	tracked   map[string]bool
	subs      map[uint64]Subscriber
	nextSubID uint64
	connURL   string

	cancel context.CancelFunc
}

// NewHub builds a (not yet started) hub.
func NewHub(c *Client, log *slog.Logger) *Hub {
	return &Hub{
		log:     log,
		client:  c,
		tracked: map[string]bool{},
		subs:    map[uint64]Subscriber{},
	}
}

// SetTracked replaces the tracked entity set (called on startup + device add/delete).
func (h *Hub) SetTracked(ids []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tracked = map[string]bool{}
	for _, id := range ids {
		h.tracked[id] = true
	}
}

func (h *Hub) isTracked(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tracked[id]
}

// Subscribe registers a local frontend WS subscriber; returns an unsubscribe func.
func (h *Hub) Subscribe(fn Subscriber) (cancel func()) {
	h.mu.Lock()
	id := h.nextSubID
	h.nextSubID++
	h.subs[id] = fn
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *Hub) dispatch(entityID string, st State) {
	if !h.isTracked(entityID) {
		return
	}
	h.mu.RLock()
	subs := make([]Subscriber, 0, len(h.subs))
	for _, fn := range h.subs {
		subs = append(subs, fn)
	}
	h.mu.RUnlock()
	for _, fn := range subs {
		fn(entityID, st)
	}
}

// Start connects to HA's websocket and loops until ctx is cancelled.
// Reconnects with exponential backoff; on each successful connect it refetches all
// states once so subscribers get a full snapshot after a gap.
func (h *Hub) Start(ctx context.Context) {
	if !h.client.Configured() {
		h.log.Info("ha hub: not configured, skipping")
		return
	}
	h.connURL = strings.Replace(h.client.baseURL, "http://", "ws://", 1)
	h.connURL = strings.Replace(h.connURL, "https://", "wss://", 1) + "/api/websocket"

	ctx, h.cancel = context.WithCancel(ctx)
	go h.run(ctx)
}

// Stop tears down the connection.
func (h *Hub) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}

func (h *Hub) run(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := h.connectAndServe(ctx)
		if ctx.Err() != nil {
			return
		}
		h.log.Warn("ha ws disconnected, reconnecting", "err", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (h *Hub) connectAndServe(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, h.connURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()

	// 1. read auth_required
	if _, msg, err := conn.Read(ctx); err != nil {
		return fmt.Errorf("read auth_required: %w", err)
	} else if !isAuthRequired(msg) {
		// some servers send auth_required; proceed either way
	}

	// 2. send auth
	auth := map[string]any{"type": "auth", "access_token": h.client.token}
	if err := writeJSON(ctx, conn, auth); err != nil {
		return fmt.Errorf("send auth: %w", err)
	}
	// 3. read auth_ok
	if _, msg, err := conn.Read(ctx); err != nil {
		return fmt.Errorf("read auth_ok: %w", err)
	} else if !isAuthOK(msg) {
		return fmt.Errorf("auth rejected: %s", trim(string(msg)))
	}

	// 4. subscribe to state_changed
	sub := map[string]any{"type": "subscribe_events", "event_type": "state_changed", "id": 1}
	if err := writeJSON(ctx, conn, sub); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// 5. full-state snapshot compensation after (re)connect
	if states, err := h.client.GetStates(ctx); err == nil {
		for _, st := range states {
			h.dispatch(st.EntityID, st)
		}
	} else {
		h.log.Warn("ha hub snapshot", "err", err)
	}

	// 6. read loop
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, msg, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		var env struct {
			Type  string `json:"type"`
			Event struct {
				EventType string `json:"event_type"`
				Data      struct {
					EntityID string `json:"entity_id"`
					NewState State  `json:"new_state"`
				} `json:"data"`
			} `json:"event"`
		}
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}
		if env.Type == "event" && env.Event.EventType == "state_changed" && env.Event.Data.EntityID != "" {
			h.dispatch(env.Event.Data.EntityID, env.Event.Data.NewState)
		}
	}
}

func writeJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

func isAuthRequired(msg []byte) bool {
	return strings.Contains(string(msg), "auth_required")
}
func isAuthOK(msg []byte) bool {
	return strings.Contains(string(msg), "auth_ok")
}
func trim(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}