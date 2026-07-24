package httpserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/auth"
	"github.com/demo/smart-home/backend/internal/middleware"
	"github.com/demo/smart-home/backend/internal/pkg/apperr"
	"github.com/demo/smart-home/backend/internal/pkg/response"
)
func (s *Server) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	s.log.Info("oauth login")
	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}
	if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/"
	}

	state, err := auth.RandomToken(16)
	if err != nil {
		s.writeOAuthError(w, http.StatusInternalServerError, apperr.CodeInternal, "生成 state 失败")
		return
	}
	verifier, challenge, err := auth.PKCEPair()
	if err != nil {
		s.writeOAuthError(w, http.StatusInternalServerError, apperr.CodeInternal, "生成 PKCE 失败")
		return
	}
	if err := s.sessions.SaveOAuthState(r.Context(), state, auth.OAuthState{
		Verifier: verifier,
		ReturnTo: returnTo,
	}); err != nil {
		s.writeOAuthError(w, http.StatusServiceUnavailable, apperr.CodeRedis, "保存登录状态失败")
		return
	}
	http.Redirect(w, r, s.oauth.AuthorizeURL(state, challenge), http.StatusFound)
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	s.log.Info("oauth callback", "host", r.Host, "query", r.URL.RawQuery)
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		s.writeOAuthError(w, http.StatusBadRequest, apperr.CodeBadRequest, "缺少 code 或 state")
		return
	}
	st, err := s.sessions.TakeOAuthState(r.Context(), state)
	if err != nil || st == nil {
		s.writeOAuthError(w, http.StatusUnauthorized, 40110, "state 无效或过期，请重新从门户进入")
		return
	}

	tr, err := s.oauth.ExchangeCode(r.Context(), code, st.Verifier)
	if err != nil {
		s.log.Error("exchange code", "err", err)
		s.writeOAuthError(w, http.StatusBadGateway, 40112, "向认证中心换票失败: "+err.Error())
		return
	}

	issuers := []string{s.cfg.OAuthIssuer, s.cfg.AuthBase}
	claims, err := s.jwks.ParseAccess(r.Context(), tr.AccessToken, issuers, s.cfg.OAuthClientID)
	if err != nil {
		s.log.Error("verify access", "err", err)
		if s.cfg.AppEnv == "prod" || s.cfg.StrictJWT {
			s.writeOAuthError(w, http.StatusUnauthorized, 40114, "access token 验签失败: "+err.Error())
			return
		}
		s.log.Warn("jwt verify failed, fallback userinfo in non-prod")
	}

	sub := ""
	if claims != nil {
		sub = claims.Subject
	}
	email, name := "", ""
	if ui, err := s.oauth.UserInfo(r.Context(), tr.AccessToken); err == nil && ui != nil {
		email, name = ui.Email, ui.Name
		if ui.Sub != "" {
			sub = ui.Sub
		}
	} else if err != nil {
		s.log.Warn("userinfo", "err", err)
	}
	if sub == "" {
		s.writeOAuthError(w, http.StatusUnauthorized, 40114, "无法取得用户 sub")
		return
	}

	u, err := s.users.UpsertBySub(r.Context(), sub, email, name)
	if err != nil {
		s.log.Error("upsert user", "err", err)
		s.writeOAuthError(w, http.StatusServiceUnavailable, apperr.CodeDB, "用户写入失败: "+err.Error())
		return
	}
	if _, err := s.homes.EnsureDefault(r.Context(), u.ID); err != nil {
		s.log.Error("ensure home", "err", err)
		s.writeOAuthError(w, http.StatusServiceUnavailable, apperr.CodeDB, "创建默认家庭失败: "+err.Error())
		return
	}

	sid, err := auth.RandomToken(24)
	if err != nil {
		s.writeOAuthError(w, http.StatusInternalServerError, apperr.CodeInternal, "创建会话失败")
		return
	}
	exp := tr.ExpiresIn
	if exp <= 0 {
		exp = 900
	}
	now := time.Now()
	sess := auth.Session{
		UserID:          u.ID,
		Sub:             u.Sub,
		Email:           u.Email,
		Name:            u.Name,
		AccessToken:     tr.AccessToken,
		RefreshToken:    tr.RefreshToken,
		AccessExpiresAt: now.Add(time.Duration(exp) * time.Second),
		LastSeenAt:      now,
		CreatedAt:       now,
	}
	if err := s.sessions.SaveSession(r.Context(), sid, sess); err != nil {
		s.writeOAuthError(w, http.StatusServiceUnavailable, apperr.CodeRedis, "保存会话失败")
		return
	}

	loc := st.ReturnTo
	if loc == "" || !strings.HasPrefix(loc, "/") || strings.HasPrefix(loc, "//") {
		loc = "/"
	}

	if s.cfg.AppBaseURL != "" && s.cfg.AppEnv != "prod" {
		ticket, err := auth.RandomToken(16)
		if err != nil {
			s.writeOAuthError(w, http.StatusInternalServerError, apperr.CodeInternal, "生成 ticket 失败")
			return
		}
		if err := s.sessions.SaveLoginTicket(r.Context(), ticket, auth.LoginTicket{
			SID:      sid,
			ReturnTo: loc,
		}); err != nil {
			s.writeOAuthError(w, http.StatusServiceUnavailable, apperr.CodeRedis, "保存 ticket 失败")
			return
		}
		complete, err := url.Parse(s.cfg.AppBaseURL)
		if err != nil {
			s.writeOAuthError(w, http.StatusInternalServerError, apperr.CodeInternal, "APP_BASE_URL 无效")
			return
		}
		complete.Path = "/oauth/complete"
		q2 := complete.Query()
		q2.Set("ticket", ticket)
		complete.RawQuery = q2.Encode()
		s.log.Info("oauth callback ok, redirect complete", "to", complete.String())
		http.Redirect(w, r, complete.String(), http.StatusFound)
		return
	}

	middleware.SetSessionCookie(w, s.cfg, sid)
	http.Redirect(w, r, loc, http.StatusFound)
}

func (s *Server) handleOAuthComplete(w http.ResponseWriter, r *http.Request) {
	s.log.Info("oauth complete", "host", r.Host)
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		s.writeOAuthError(w, http.StatusBadRequest, apperr.CodeBadRequest, "缺少 ticket")
		return
	}
	t, err := s.sessions.TakeLoginTicket(r.Context(), ticket)
	if err != nil || t == nil || t.SID == "" {
		s.writeOAuthError(w, http.StatusUnauthorized, 40110, "ticket 无效或过期，请重新登录")
		return
	}
	sess, err := s.sessions.GetSession(r.Context(), t.SID)
	if err != nil || sess == nil {
		s.writeOAuthError(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "会话不存在")
		return
	}
	middleware.SetSessionCookie(w, s.cfg, t.SID)
	loc := t.ReturnTo
	if loc == "" || !strings.HasPrefix(loc, "/") || strings.HasPrefix(loc, "//") {
		loc = "/"
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p := middleware.UserFromContext(r.Context())
	if p == nil {
		response.Fail(w, http.StatusUnauthorized, apperr.CodeUnauthorized, "未登录")
		return
	}
	h, err := s.homes.EnsureDefault(r.Context(), p.UserID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, apperr.CodeDB, "加载家庭失败")
		return
	}
	s.ensureHALoaded(h.ID)
	out := map[string]any{
		"id":    p.UserID,
		"sub":   p.Sub,
		"email": p.Email,
		"name":  p.Name,
	}
	if h != nil {
		out["home"] = map[string]any{
			"id":   h.ID,
			"name": h.Name,
		}
	}
	response.OK(w, out)
}

func (s *Server) writeOAuthError(w http.ResponseWriter, httpStatus, code int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(httpStatus)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>登录失败</title></head>
<body style="font-family:sans-serif;max-width:40rem;margin:3rem auto;padding:0 1rem">
<h1>登录失败</h1>
<p>%s</p>
<p style="color:#666">code=%d</p>
<p><a href="%s/oauth/login?return_to=/">重新登录</a></p>
</body></html>`, htmlEscape(message), code, htmlEscape(s.cfg.AppBaseURL))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
