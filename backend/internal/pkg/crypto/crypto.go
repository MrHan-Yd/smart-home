// Package crypto provides AES-GCM symmetric encryption for secrets stored at rest
// (e.g. HA Long-Lived Tokens in ha_instances). Tokens are never returned to the client.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Box encrypts plaintext with AES-GCM using the derived key. Output is hex(nonce|ciphertext).
func Box(key []byte, plaintext string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(out), nil
}

// Unbox reverses Box. Returns error on tamper / wrong key.
func Unbox(key []byte, encoded string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}
	raw, err := hex.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("crypto: hex decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns+1 {
		return "", errors.New("crypto: ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(pt), nil
}

// DeriveKey returns a 32-byte AES key from a passphrase via HMAC-SHA256 with a fixed salt.
// Used as fallback when HA_TOKEN_ENC_KEY is unset, so tokens still aren't stored in plaintext.
func DeriveKey(passphrase string) []byte {
	mac := hmac.New(sha256.New, []byte("smart-home/ha-token/v1"))
	mac.Write([]byte(passphrase))
	return mac.Sum(nil)
}

// ParseHexKey accepts a 64-char hex string (32 bytes) or returns nil.
func ParseHexKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid hex key: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("crypto: hex key must be 32 bytes, got %d", len(raw))
	}
	return raw, nil
}