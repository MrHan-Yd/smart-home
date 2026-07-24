package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/config"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg  config.Config
	log  *slog.Logger
	db   *pgxpool.Pool
	rdb  *redis.Client
	hass *hass.Client
	mux  *http.ServeMux
}

func New(cfg config.Config, log *slog.Logger, db *pgxpool.Pool, rdb *redis.Client, ha *hass.Client) *Server {
	s := &Server{cfg: cfg, log: log, db: db, rdb: rdb, hass: ha, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.cors(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)
	s.mux.HandleFunc("GET /api/v1/ha/status", s.handleHAStatus)
	s.mux.HandleFunc("GET /api/v1/meta", s.handleMeta)
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
	// 不暴露完整 URL 中的敏感路径；仅 host 方便排查
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
		"service":         "smart-home-service",
		"auth_portal_url": s.cfg.AuthPortalURL,
		"app_base_url":    s.cfg.AppBaseURL,
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
