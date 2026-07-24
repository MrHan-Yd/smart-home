package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/demo/smart-home/backend/internal/auth"
	"github.com/demo/smart-home/backend/internal/config"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
	"github.com/google/uuid"
)

type ctxKey int

const userCtxKey ctxKey = 1

var (
	ErrRefreshFailed = errors.New("refresh failed")
	ErrNoRefresh     = errors.New("no refresh token")
)

type Principal struct {
	UserID uuid.UUID
	Sub    string
	Email  string
	Name   string
	SID    string
}

func UserFromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(userCtxKey).(*Principal)
	return p
}

type Auth struct {
	cfg   config.Config
	store *auth.Store
	oauth *auth.OAuthClient
	jwks  *auth.JWKS
	log   *slog.Logger
}

func NewAuth(cfg config.Config, store *auth.Store, oauth *auth.OAuthClient, jwks *auth.JWKS, log *slog.Logger) *Auth {
	if log == nil {
		log = slog.Default()
	}
	return &Auth{cfg: cfg, store: store, oauth: oauth, jwks: jwks, log: log}
}

func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(a.cfg.SessionCookieName)
		if err != nil || c.Value == "" {
			response.Fail(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "未登录")
			return
		}
		sid := c.Value
		sess, err := a.store.GetSession(r.Context(), sid)
		if err != nil || sess == nil {
			ClearSessionCookie(w, a.cfg)
			response.Fail(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "未登录")
			return
		}

		if a.cfg.SessionIdle > 0 {
			last := sess.LastSeenAt
			if last.IsZero() {
				last = sess.CreatedAt
			}
			if !last.IsZero() && time.Since(last) > a.cfg.SessionIdle {
				a.log.Info("session idle timeout", "sid_prefix", trimSID(sid), "idle", time.Since(last).String())
				a.kick(r.Context(), w, sid, sess)
				response.Fail(w, http.StatusUnauthorized, apperr.CodeLoginExpired, "会话已超时，请重新登录")
				return
			}
		}

		skew := a.cfg.AccessRefreshSkew
		if skew <= 0 {
			skew = 60 * time.Second
		}
		if time.Until(sess.AccessExpiresAt) < skew {
			if err := a.refreshSession(r.Context(), sid); err != nil {
				a.log.Warn("refresh failed, kick session", "err", err, "sid_prefix", trimSID(sid))
				a.kick(r.Context(), w, sid, sess)
				response.Fail(w, http.StatusUnauthorized, apperr.CodeLoginExpired, "登录已过期，请重新登录")
				return
			}
			sess, err = a.store.GetSession(r.Context(), sid)
			if err != nil || sess == nil {
				ClearSessionCookie(w, a.cfg)
				response.Fail(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "未登录")
				return
			}
		}

		if err := a.store.TouchSession(r.Context(), sid, sess); err != nil {
			a.log.Warn("touch session", "err", err)
		}

		p := &Principal{
			UserID: sess.UserID,
			Sub:    sess.Sub,
			Email:  sess.Email,
			Name:   sess.Name,
			SID:    sid,
		}
		ctx := context.WithValue(r.Context(), userCtxKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) kick(ctx context.Context, w http.ResponseWriter, sid string, sess *auth.Session) {
	if sess != nil {
		if sess.AccessToken != "" {
			_ = a.oauth.Revoke(ctx, sess.AccessToken)
		}
		if sess.RefreshToken != "" {
			_ = a.oauth.Revoke(ctx, sess.RefreshToken)
		}
	}
	_ = a.store.DeleteSession(ctx, sid)
	ClearSessionCookie(w, a.cfg)
}

func (a *Auth) refreshSession(ctx context.Context, sid string) error {
	return a.store.WithRefreshLock(ctx, sid, func() error {
		cur, err := a.store.GetSession(ctx, sid)
		if err != nil {
			return err
		}
		if cur == nil {
			return ErrRefreshFailed
		}
		skew := a.cfg.AccessRefreshSkew
		if skew <= 0 {
			skew = 60 * time.Second
		}
		if time.Until(cur.AccessExpiresAt) >= skew {
			return nil
		}
		if cur.RefreshToken == "" {
			return ErrNoRefresh
		}
		tr, err := a.oauth.Refresh(ctx, cur.RefreshToken)
		if err != nil {
			return errors.Join(ErrRefreshFailed, err)
		}
		issuers := []string{a.cfg.OAuthIssuer, a.cfg.AuthBase}
		if _, err := a.jwks.ParseAccess(ctx, tr.AccessToken, issuers, a.cfg.OAuthClientID); err != nil {
			if a.cfg.AppEnv == "prod" || a.cfg.StrictJWT {
				return errors.Join(ErrRefreshFailed, err)
			}
			a.log.Warn("refresh jwt verify soft-fail in non-prod", "err", err)
		}
		cur.AccessToken = tr.AccessToken
		if tr.RefreshToken != "" {
			cur.RefreshToken = tr.RefreshToken
		}
		exp := tr.ExpiresIn
		if exp <= 0 {
			exp = 900
		}
		cur.AccessExpiresAt = time.Now().Add(time.Duration(exp) * time.Second)
		cur.LastSeenAt = time.Now()
		return a.store.SaveSession(ctx, sid, *cur)
	})
}

func trimSID(sid string) string {
	if len(sid) <= 8 {
		return sid
	}
	return sid[:8]
}

func sameSite(s string) http.SameSite {
	switch s {
	case "strict", "Strict":
		return http.SameSiteStrictMode
	case "none", "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func SetSessionCookie(w http.ResponseWriter, cfg config.Config, sid string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    sid,
		Path:     "/",
		MaxAge:   int(cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite(cfg.CookieSameSite),
		Domain:   cfg.CookieDomain,
	})
}

func ClearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	expired := time.Unix(0, 0).UTC()
	variants := []http.Cookie{
		{
			Name: cfg.SessionCookieName, Value: "", Path: "/",
			MaxAge: -1, Expires: expired, HttpOnly: true,
			Secure: cfg.CookieSecure, SameSite: sameSite(cfg.CookieSameSite),
		},
		{
			Name: cfg.SessionCookieName, Value: "", Path: "/",
			MaxAge: -1, Expires: expired, HttpOnly: true,
			Secure: false, SameSite: http.SameSiteLaxMode,
		},
	}
	if cfg.CookieDomain != "" {
		variants = append(variants, http.Cookie{
			Name: cfg.SessionCookieName, Value: "", Path: "/",
			MaxAge: -1, Expires: expired, HttpOnly: true,
			Secure: cfg.CookieSecure, SameSite: sameSite(cfg.CookieSameSite),
			Domain: cfg.CookieDomain,
		})
	}
	for i := range variants {
		http.SetCookie(w, &variants[i])
	}
}
