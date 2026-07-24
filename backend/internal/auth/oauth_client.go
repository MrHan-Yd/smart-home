package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/demo/smart-home/backend/internal/config"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
}

type UserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type OAuthClient struct {
	cfg    config.Config
	client *http.Client
}

func NewOAuthClient(cfg config.Config) *OAuthClient {
	return &OAuthClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func PKCEPair() (verifier, challenge string, err error) {
	verifier, err = RandomToken(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func (c *OAuthClient) AuthorizeURL(state, challenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.cfg.OAuthClientID)
	q.Set("redirect_uri", c.cfg.OAuthRedirectURI)
	q.Set("scope", c.cfg.OAuthScopes)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return c.cfg.AuthBase + "/oauth/authorize?" + q.Encode()
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code, verifier string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.OAuthRedirectURI)
	form.Set("client_id", c.cfg.OAuthClientID)
	form.Set("client_secret", c.cfg.OAuthClientSecret)
	form.Set("code_verifier", verifier)
	return c.token(ctx, form)
}

func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.cfg.OAuthClientID)
	form.Set("client_secret", c.cfg.OAuthClientSecret)
	return c.token(ctx, form)
}

func (c *OAuthClient) Revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", c.cfg.OAuthClientID)
	form.Set("client_secret", c.cfg.OAuthClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthBase+"/oauth/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("revoke status %d: %s", res.StatusCode, string(b))
	}
	return nil
}

func (c *OAuthClient) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.AuthBase+"/oauth/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("userinfo status %d: %s", res.StatusCode, string(b))
	}
	var u UserInfo
	if err := json.NewDecoder(res.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *OAuthClient) token(ctx context.Context, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthBase+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token status %d: %s", res.StatusCode, string(b))
	}
	var tr TokenResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}
