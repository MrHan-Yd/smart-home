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
	"github.com/demo/smart-home/backend/internal/module/device"
	"github.com/demo/smart-home/backend/internal/module/home"
	"github.com/demo/smart-home/backend/internal/module/user"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg      config.Config
	log      *slog.Logger
	db       *pgxpool.Pool
	rdb      *redis.Client
	hass     *hass.Client
	sessions *auth.Store
	oauth    *auth.OAuthClient
	jwks     *auth.JWKS
	authMW   *middleware.Auth
	users    *user.Repo
	homes    *home.Repo
	devices  *device.Repo
	mux      *http.ServeMux
}

func New(cfg config.Config, log *slog.Logger, db *pgxpool.Pool, rdb *redis.Client, ha *hass.Client) *Server {
	sessions := auth.NewStore(rdb, cfg.SessionTTL)
	oauth := auth.NewOAuthClient(cfg)
	jwks := auth.NewJWKS(cfg.AuthBase + "/jwks.json")
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
		devices:  device.NewRepo(db),
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
	s.mux.HandleFunc("POST /oauth/logout", s.handleOAuthLogout)

	s.mux.HandleFunc("GET /api/v1/ha/status", s.handleHAStatus)
	s.mux.HandleFunc("GET /api/v1/meta", s.handleMeta)

	s.mux.Handle("GET /api/v1/me", s.authMW.Require(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("GET /api/v1/discover/entities", s.authMW.Require(http.HandlerFunc(s.handleDiscoverEntities)))
	s.mux.Handle("GET /api/v1/devices", s.authMW.Require(http.HandlerFunc(s.handleListDevices)))
	s.mux.Handle("POST /api/v1/devices", s.authMW.Require(http.HandlerFunc(s.handleCreateDevice)))
	s.mux.Handle("POST /api/v1/devices/batch", s.authMW.Require(http.HandlerFunc(s.handleBatchCreateDevices)))
	s.mux.Handle("GET /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handleGetDevice)))
	s.mux.Handle("PATCH /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handlePatchDevice)))
	s.mux.Handle("DELETE /api/v1/devices/{id}", s.authMW.Require(http.HandlerFunc(s.handleDeleteDevice)))
	s.mux.Handle("POST /api/v1/devices/{id}/actions", s.authMW.Require(http.HandlerFunc(s.handleDeviceAction)))
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

func (s *Server) handleHAStatus(w http.ResponseWriter, r *http.Request) {
	configured := s.hass != nil && s.hass.Configured()
	out := map[string]any{
		"configured":    configured,
		"online":        false,
		"base_url_host": "",
		"latency_ms":    nil,
		"message":       "not configured",
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
