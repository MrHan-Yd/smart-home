package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/auth"
	"github.com/demo/smart-home/backend/internal/config"
	"github.com/demo/smart-home/backend/internal/middleware"
	"github.com/demo/smart-home/backend/internal/module/actionlog"
	"github.com/demo/smart-home/backend/internal/module/analytics"
	"github.com/demo/smart-home/backend/internal/module/device"
	"github.com/demo/smart-home/backend/internal/module/hainstance"
	"github.com/demo/smart-home/backend/internal/module/home"
	"github.com/demo/smart-home/backend/internal/module/room"
	"github.com/demo/smart-home/backend/internal/module/scenario"
	"github.com/demo/smart-home/backend/internal/module/user"
	"github.com/demo/smart-home/backend/internal/pkg/crypto"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg      config.Config
	log      *slog.Logger
	db       *pgxpool.Pool
	rdb      *redis.Client
	hass     *hass.Client
	hub      *hass.Hub
	sessions *auth.Store
	oauth    *auth.OAuthClient
	jwks     *auth.JWKS
	authMW   *middleware.Auth
	users    *user.Repo
	homes    *home.Repo
	rooms    *room.Repo
	devices  *device.Repo
	alog     *actionlog.Repo
	hainst   *hainstance.Repo
	anal     *analytics.Repo
	scen     *scenario.Repo
	haLoaded bool
	telemetrySub bool
	mux      *http.ServeMux
}

func New(cfg config.Config, log *slog.Logger, db *pgxpool.Pool, rdb *redis.Client, ha *hass.Client) *Server {
	sessions := auth.NewStore(rdb, cfg.SessionTTL)
	oauth := auth.NewOAuthClient(cfg)
	jwks := auth.NewJWKS(cfg.AuthBase + "/jwks.json")
	encKey := resolveHAEncKey(cfg, log)
	s := &Server{
		cfg:      cfg,
		log:      log,
		db:       db,
		rdb:      rdb,
		hass:     ha,
		sessions: sessions,
		oauth:    oauth,
		jwks:     jwks,
		authMW:   middleware.NewAuth(cfg, sessions, oauth, jwks, log),
		users:    user.NewRepo(db),
		homes:    home.NewRepo(db),
		rooms:    room.NewRepo(db),
		devices:  device.NewRepo(db),
		alog:     actionlog.NewRepo(db),
		hainst:   hainstance.NewRepo(db, encKey),
		anal:     analytics.NewRepo(db),
		scen:     scenario.NewRepo(db),
		hub:      hass.NewHub(ha, log),
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.cors(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)

	s.mux.HandleFunc("GET /oauth/login", s.handleOAuthLogin)
	s.mux.HandleFunc("GET /oauth/callback", s.handleOAuthCallback)
	s.mux.HandleFunc("GET /oauth/complete", s.handleOAuthComplete)

	s.mux.HandleFunc("GET /api/v1/ha/status", s.handleHAStatus)
	s.mux.HandleFunc("GET /api/v1/meta", s.handleMeta)

	s.mux.Handle("GET /api/v1/rooms", s.authMW.Require(http.HandlerFunc(s.handleListRooms)))
	s.mux.Handle("POST /api/v1/rooms", s.authMW.Require(http.HandlerFunc(s.handleCreateRoom)))
	s.mux.Handle("PATCH /api/v1/rooms/{id}", s.authMW.Require(http.HandlerFunc(s.handlePatchRoom)))
	s.mux.Handle("DELETE /api/v1/rooms/{id}", s.authMW.Require(http.HandlerFunc(s.handleDeleteRoom)))

	s.mux.Handle("GET /api/v1/me", s.authMW.Require(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("GET /api/v1/discover/entities", s.authMW.Require(http.HandlerFunc(s.handleDiscoverEntities)))
	s.mux.Handle("GET /api/v1/devices", s.authMW.Require(http.HandlerFunc(s.handleListDevices)))
	s.mux.Handle("POST /api/v1/devices", s.authMW.Require(http.HandlerFunc(s.handleCreateDevice)))
	s.mux.Handle("POST /api/v1/devices/batch", s.authMW.Require(http.HandlerFunc(s.handleBatchCreateDevices)))
	s.mux.Handle("POST /api/v1/devices/composite", s.authMW.Require(http.HandlerFunc(s.handleCreateCompositeDevice)))
	s.mux.Handle("GET /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handleGetDevice)))
	s.mux.Handle("PATCH /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handlePatchDevice)))
	s.mux.Handle("DELETE /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handleDeleteDevice)))
	s.mux.Handle("POST /api/v1/devices/{id}/actions", s.authMW.Require(http.HandlerFunc(s.handleDeviceAction)))
	s.mux.Handle("GET /api/v1/devices/{id}/history", s.authMW.Require(http.HandlerFunc(s.handleDeviceHistory)))
	s.mux.Handle("GET /api/v1/history", s.authMW.Require(http.HandlerFunc(s.handleHistory)))
	s.mux.Handle("GET /api/v1/action-logs", s.authMW.Require(http.HandlerFunc(s.handleListActionLogs)))

	s.mux.Handle("GET /api/v1/analytics/summary", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsSummary)))
	s.mux.Handle("GET /api/v1/analytics/runtime", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsRuntime)))
	s.mux.Handle("GET /api/v1/analytics/ranking", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsRanking)))
	s.mux.Handle("GET /api/v1/analytics/type-mix", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsTypeMix)))
	s.mux.Handle("GET /api/v1/analytics/heatmap", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsHeatmap)))
	s.mux.Handle("GET /api/v1/analytics/environment", s.authMW.Require(http.HandlerFunc(s.handleAnalyticsEnvironment)))

	s.mux.Handle("GET /api/v1/ha/instances", s.authMW.Require(http.HandlerFunc(s.handleListHAInstances)))
	s.mux.Handle("POST /api/v1/ha/instances", s.authMW.Require(http.HandlerFunc(s.handleCreateHAInstance)))
	s.mux.Handle("PATCH /api/v1/ha/instances/{id}", s.authMW.Require(http.HandlerFunc(s.handleUpdateHAInstance)))
	s.mux.Handle("DELETE /api/v1/ha/instances/{id}", s.authMW.Require(http.HandlerFunc(s.handleDeleteHAInstance)))
	s.mux.Handle("POST /api/v1/ha/instances/{id}/probe", s.authMW.Require(http.HandlerFunc(s.handleProbeHAInstance)))
	s.mux.Handle("POST /api/v1/ha/instances/{id}/activate", s.authMW.Require(http.HandlerFunc(s.handleActivateHAInstance)))

	s.mux.Handle("GET /api/v1/scenarios", s.authMW.Require(http.HandlerFunc(s.handleListScenarios)))
	s.mux.Handle("POST /api/v1/scenarios", s.authMW.Require(http.HandlerFunc(s.handleCreateScenario)))
	s.mux.Handle("GET /api/v1/scenarios/{id}", s.authMW.Require(http.HandlerFunc(s.handleGetScenario)))
	s.mux.Handle("PATCH /api/v1/scenarios/{id}", s.authMW.Require(http.HandlerFunc(s.handlePatchScenario)))
	s.mux.Handle("PUT /api/v1/scenarios/{id}/steps", s.authMW.Require(http.HandlerFunc(s.handleReplaceScenarioSteps)))
	s.mux.Handle("DELETE /api/v1/scenarios/{id}", s.authMW.Require(http.HandlerFunc(s.handleDeleteScenario)))
	s.mux.Handle("POST /api/v1/scenarios/{id}/run", s.authMW.Require(http.HandlerFunc(s.handleRunScenario)))

	s.mux.Handle("GET /api/v1/ws", s.authMW.Require(http.HandlerFunc(s.handleWS)))
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		response.Fail(w, http.StatusServiceUnavailable, 50000, "postgres not ready")
		return
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		response.Fail(w, http.StatusServiceUnavailable, 50000, "redis not ready")
		return
	}
	if s.cfg.ReadyzRequireHA {
		if s.hass == nil || !s.hass.Configured() {
			response.Fail(w, http.StatusServiceUnavailable, 50200, "hass not configured")
			return
		}
		if err := s.hass.Ping(ctx); err != nil {
			response.Fail(w, http.StatusServiceUnavailable, 50200, "hass not ready")
			return
		}
	}
	response.OK(w, map[string]any{
		"status": "ready",
		"ha":     s.hass != nil && s.hass.Configured(),
	})
}

// userHomeSafe returns the caller's home id without failing on missing auth;
// used by public endpoints (e.g. HAStatus) that still want to surface active
// instance when the user happens to be logged in.
func (s *Server) userHomeSafe(r *http.Request) *uuid.UUID {
	p := middleware.UserFromContext(r.Context())
	if p == nil {
		return nil
	}
	h, err := s.homes.EnsureDefault(r.Context(), p.UserID)
	if err != nil || h == nil {
		return nil
	}
	return &h.ID
}

func (s *Server) handleHAStatus(w http.ResponseWriter, r *http.Request) {
	configured := s.hass != nil && s.hass.Configured()
	out := map[string]any{
		"configured":         configured,
		"online":             false,
		"base_url_host":      "",
		"latency_ms":         nil,
		"message":            "not configured",
		"active_instance_id": nil,
	}
	// surface which DB instance is active (if any)
	if hid := s.userHomeSafe(r); hid != nil {
		if x, _ := s.hainst.Active(r.Context(), *hid); x != nil {
			out["active_instance_id"] = x.ID
		}
	}
	if !configured {
		response.OK(w, out)
		return
	}
	out["base_url_host"] = trimURLHost(s.cfg.HassBaseURL)
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.HassTimeout)
	defer cancel()
	err := s.hass.Ping(ctx)
	out["latency_ms"] = time.Since(start).Milliseconds()
	if err != nil {
		out["online"] = false
		out["message"] = err.Error()
		response.OK(w, out)
		return
	}
	out["online"] = true
	out["message"] = "ok"
	response.OK(w, out)
}

func (s *Server) handleMeta(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, map[string]any{
		"service":          "smart-home-service",
		"auth_portal_url":  s.cfg.AuthPortalURL,
		"app_base_url":     s.cfg.AppBaseURL,
		"session_ttl_sec":  int(s.cfg.SessionTTL.Seconds()),
		"session_idle_sec": int(s.cfg.SessionIdle.Seconds()),
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = s.cfg.AppBaseURL
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func trimURLHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		raw = strings.TrimPrefix(raw, "https://")
		raw = strings.TrimPrefix(raw, "http://")
		if i := strings.IndexByte(raw, '/'); i >= 0 {
			return raw[:i]
		}
		return raw
	}
	return u.Host
}

// StartHub launches the HA websocket hub (if configured). Called once from main.
func (s *Server) StartHub(ctx context.Context) {
	if s.hub != nil && s.hass.Configured() {
		s.hub.Start(ctx)
	}
}

// ensureHALoaded reconfigures the HA client from the active DB instance once
// per process (self-use single home). Subsequent changes call reloadActiveHA directly.
func (s *Server) ensureHALoaded(homeID uuid.UUID) {
	if s.haLoaded || s.hainst == nil {
		return
	}
	s.haLoaded = true
	s.reloadActiveHA(context.Background(), homeID)
	s.refreshHubTracking(context.Background(), homeID)
}

// refreshHubTracking reloads the tracked entity set from the DB and keeps the
// telemetry-writing subscriber registered (once).
func (s *Server) refreshHubTracking(ctx context.Context, homeID uuid.UUID) {
	if s.hub == nil {
		return
	}
	// user_id = home.owner; for self-use single home, look it up via the home.
	owner, err := s.homes.Owner(ctx, homeID)
	if err != nil || owner == uuid.Nil {
		s.log.Warn("hub tracking: owner unknown", "err", err)
		return
	}
	ids, err := s.devices.EntityIDsByUserWithMembers(ctx, owner)
	if err != nil {
		s.log.Warn("hub tracking: load entity ids", "err", err)
		return
	}
	s.hub.SetTracked(ids)

	if !s.telemetrySub {
		s.telemetrySub = true
		s.hub.Subscribe(func(entityID string, st hass.State) {
			s.recordTelemetry(owner, entityID, st)
		})
	}
}

// recordTelemetry persists a slim telemetry sample asynchronously.
func (s *Server) recordTelemetry(userID uuid.UUID, entityID string, st hass.State) {
	d, err := s.devices.GetByEntity(context.Background(), userID, entityID)
	if err != nil || d == nil {
		return
	}
	var num *float64
	if f, ok := parseFloat(st.State); ok {
		v := f
		num = &v
	}
	_ = s.anal.WriteSample(context.Background(), analytics.Sample{
		DeviceID:   d.ID,
		EntityID:   entityID,
		TS:         time.Now().UTC(),
		State:      st.State,
		NumValue:   num,
		Attributes: st.Attributes,
	})
}
// OAUTH_CLIENT_SECRET so tokens are still encrypted at rest (with a warning).
func resolveHAEncKey(cfg config.Config, log *slog.Logger) []byte {
	if cfg.HATokenEncKey != "" {
		k, err := crypto.ParseHexKey(cfg.HATokenEncKey)
		if err != nil {
			log.Error("HA_TOKEN_ENC_KEY invalid, falling back to derived key", "err", err)
		} else {
			return k
		}
	}
	log.Warn("HA_TOKEN_ENC_KEY unset; deriving HA token encryption key from OAUTH_CLIENT_SECRET")
	return crypto.DeriveKey(cfg.OAuthClientSecret)
}
