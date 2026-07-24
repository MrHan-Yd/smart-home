package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKS struct {
	url    string
	client *http.Client
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	expAt  time.Time
	ttl    time.Duration
}

func NewJWKS(jwksURL string) *JWKS {
	return &JWKS{
		url:    jwksURL,
		client: &http.Client{Timeout: 10 * time.Second},
		keys:   map[string]*rsa.PublicKey{},
		ttl:    10 * time.Minute,
	}
}

func (j *JWKS) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	if time.Now().Before(j.expAt) {
		if k, ok := j.keys[kid]; ok {
			j.mu.RUnlock()
			return k, nil
		}
	}
	j.mu.RUnlock()

	if err := j.refresh(ctx); err != nil {
		return nil, err
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	k, ok := j.keys[kid]
	if !ok {
		return nil, fmt.Errorf("kid not found: %s", kid)
	}
	return k, nil
}

func (j *JWKS) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.url, nil)
	if err != nil {
		return err
	}
	res, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", res.StatusCode)
	}
	var doc jwksDoc
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicFromJWK(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	j.mu.Lock()
	j.keys = keys
	j.expAt = time.Now().Add(j.ttl)
	j.mu.Unlock()
	return nil
}

func rsaPublicFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nb)
	var eInt int
	for _, b := range eb {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: n, E: eInt}, nil
}

type AccessClaims struct {
	Scope    string `json:"scope"`
	TokenUse string `json:"token_use"`
	jwt.RegisteredClaims
}

func (j *JWKS) ParseAccess(ctx context.Context, tokenStr string, issuers []string, audience string) (*AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(60*time.Second),
	)
	tok, err := parser.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}
		return j.getKey(ctx, kid)
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	if len(issuers) > 0 {
		okIss := false
		for _, iss := range issuers {
			iss = strings.TrimRight(iss, "/")
			if iss != "" && claims.Issuer == iss {
				okIss = true
				break
			}
		}
		if !okIss {
			return nil, fmt.Errorf("invalid iss: got %q want one of %v", claims.Issuer, issuers)
		}
	}
	if audience != "" && len(claims.Audience) > 0 {
		okAud := false
		for _, a := range claims.Audience {
			if a == audience {
				okAud = true
				break
			}
		}
		if !okAud {
			return nil, fmt.Errorf("invalid aud: got %v want %q", claims.Audience, audience)
		}
	}
	return claims, nil
}
