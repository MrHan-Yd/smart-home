package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keySession  = "sh:sess:"
	keyState    = "sh:oauth:state:"
	keyRefresh  = "sh:refresh:lock:"
	keyTicket   = "sh:oauth:ticket:"
	keyIdem     = "sh:idem:"
	idemKeyTTL  = 10 * time.Minute
)

// IdempotencyRecord stores the serialized response for a repeated request.
type IdempotencyRecord struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// TakeIdempotency tries to claim an idempotency slot keyed by (sid, key).
// If the slot already exists, it returns the stored record (hit=true).
// If claimed now, the caller must FinishIdempotency to publish the response;
// concurrent callers see the in-flight sentinel and get a 409.
func (s *Store) TakeIdempotency(ctx context.Context, sid, key string) (rec *IdempotencyRecord, hit bool, err error) {
	if key == "" {
		return nil, false, nil
	}
	full := keyIdem + sid + ":" + key
	ok, err := s.rdb.SetNX(ctx, full, "0", idemKeyTTL).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		raw, gerr := s.rdb.Get(ctx, full).Bytes()
		if gerr == redis.Nil {
			// in-flight by another request
			return nil, true, nil
		}
		if gerr != nil {
			return nil, false, gerr
		}
		var r IdempotencyRecord
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, false, err
		}
		return &r, true, nil
	}
	return nil, false, nil
}

// FinishIdempotency stores the response for a previously claimed slot.
func (s *Store) FinishIdempotency(ctx context.Context, sid, key string, rec IdempotencyRecord) error {
	if key == "" {
		return nil
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, keyIdem+sid+":"+key, raw, idemKeyTTL).Err()
}

// ReleaseIdempotency drops an uncommitted slot on failure so the request can be retried.
func (s *Store) ReleaseIdempotency(ctx context.Context, sid, key string) {
	if key == "" {
		return
	}
	_ = s.rdb.Del(ctx, keyIdem+sid+":"+key).Err()
}

type Session struct {
	UserID          uuid.UUID `json:"user_id"`
	Sub             string    `json:"sub"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	AccessToken     string    `json:"access_token"`
	RefreshToken    string    `json:"refresh_token"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type OAuthState struct {
	Verifier string `json:"verifier"`
	ReturnTo string `json:"return_to"`
}

type LoginTicket struct {
	SID      string `json:"sid"`
	ReturnTo string `json:"return_to"`
}

type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewStore(rdb *redis.Client, ttl time.Duration) *Store {
	return &Store{rdb: rdb, ttl: ttl}
}

func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) SaveSession(ctx context.Context, sid string, sess Session) error {
	raw, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, keySession+sid, raw, s.ttl).Err()
}

func (s *Store) GetSession(ctx context.Context, sid string) (*Session, error) {
	raw, err := s.rdb.Get(ctx, keySession+sid).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, sid string) error {
	return s.rdb.Del(ctx, keySession+sid).Err()
}

func (s *Store) TouchSession(ctx context.Context, sid string, sess *Session) error {
	sess.LastSeenAt = time.Now()
	return s.SaveSession(ctx, sid, *sess)
}

func (s *Store) SaveOAuthState(ctx context.Context, state string, st OAuthState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, keyState+state, raw, 10*time.Minute).Err()
}

func (s *Store) TakeOAuthState(ctx context.Context, state string) (*OAuthState, error) {
	key := keyState + state
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = s.rdb.Del(ctx, key)
	var st OAuthState
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) WithRefreshLock(ctx context.Context, sid string, fn func() error) error {
	key := keyRefresh + sid
	ok, err := s.rdb.SetNX(ctx, key, "1", 15*time.Second).Result()
	if err != nil {
		return err
	}
	if !ok {
		for i := 0; i < 30; i++ {
			time.Sleep(100 * time.Millisecond)
			exists, err := s.rdb.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				return nil
			}
		}
		return fmt.Errorf("refresh lock timeout")
	}
	defer s.rdb.Del(ctx, key)
	return fn()
}

func (s *Store) SaveLoginTicket(ctx context.Context, ticket string, t LoginTicket) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, keyTicket+ticket, raw, 2*time.Minute).Err()
}

func (s *Store) TakeLoginTicket(ctx context.Context, ticket string) (*LoginTicket, error) {
	key := keyTicket + ticket
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = s.rdb.Del(ctx, key)
	var t LoginTicket
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
