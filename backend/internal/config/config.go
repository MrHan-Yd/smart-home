package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppHost    string
	AppPort    int
	AppBaseURL string
	AppEnv     string
	LogLevel   string

	AuthBase          string
	OAuthClientID     string
	OAuthClientSecret string
	OAuthRedirectURI  string
	OAuthIssuer       string
	OAuthScopes       string
	AuthPortalURL     string

	SessionCookieName string
	SessionTTL        time.Duration
	SessionIdle       time.Duration
	AccessRefreshSkew time.Duration
	StrictJWT         bool
	CookieSecure      bool
	CookieDomain      string
	CookieSameSite    string

	DatabaseURL string
	RedisURL    string

	HassBaseURL      string
	HassToken        string
	HassTimeout      time.Duration
	ReadyzRequireHA  bool
	HATokenEncKey    string
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.AppHost, c.AppPort)
}

func Load() (Config, error) {
	port, err := getenvInt("APP_PORT", 3002)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := getenvDuration("SESSION_TTL", 168*time.Hour)
	if err != nil {
		return Config{}, err
	}
	sessionIdle, err := getenvDuration("SESSION_IDLE", 2*time.Hour)
	if err != nil {
		return Config{}, err
	}
	refreshSkew, err := getenvDuration("ACCESS_REFRESH_SKEW", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	hassTimeout, err := getenvDuration("HASS_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppHost:    getenv("APP_HOST", "0.0.0.0"),
		AppPort:    port,
		AppBaseURL: strings.TrimRight(getenv("APP_BASE_URL", "http://127.0.0.1:5175"), "/"),
		AppEnv:     getenv("APP_ENV", "dev"),
		LogLevel:   getenv("LOG_LEVEL", "info"),

		AuthBase:          strings.TrimRight(getenv("AUTH_BASE", "http://127.0.0.1:3000"), "/"),
		OAuthClientID:     getenv("OAUTH_CLIENT_ID", "app_696b1b"),
		OAuthClientSecret: getenv("OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURI:  getenv("OAUTH_REDIRECT_URI", "http://127.0.0.1:3002/oauth/callback"),
		OAuthIssuer:       strings.TrimRight(getenv("OAUTH_ISSUER", "http://127.0.0.1:3000"), "/"),
		OAuthScopes:       getenv("OAUTH_SCOPES", "openid profile email"),
		AuthPortalURL:     strings.TrimRight(getenv("AUTH_PORTAL_URL", "http://127.0.0.1:5173"), "/"),

		SessionCookieName: getenv("SESSION_COOKIE_NAME", "sh_sid"),
		SessionTTL:        sessionTTL,
		SessionIdle:       sessionIdle,
		AccessRefreshSkew: refreshSkew,
		StrictJWT:         getenvBool("STRICT_JWT", false),
		CookieSecure:      getenvBool("COOKIE_SECURE", false),
		CookieDomain:      getenv("COOKIE_DOMAIN", ""),
		CookieSameSite:    getenv("COOKIE_SAME_SITE", "Lax"),

		DatabaseURL: getenv("DATABASE_URL", ""),
		RedisURL:    getenv("REDIS_URL", ""),

		HassBaseURL:     strings.TrimRight(getenv("HASS_BASE_URL", ""), "/"),
		HassToken:       getenv("HASS_TOKEN", ""),
		HassTimeout:     hassTimeout,
		ReadyzRequireHA: getenvBool("READYZ_REQUIRE_HA", false),
		HATokenEncKey:   strings.TrimSpace(getenv("HA_TOKEN_ENC_KEY", "")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("REDIS_URL is required")
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvInt(k string, def int) (int, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", k, err)
	}
	return n, nil
}

func getenvDuration(k string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", k, err)
	}
	return d, nil
}
