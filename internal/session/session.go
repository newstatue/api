package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Manager struct {
	Valkey *redis.Client
	Secure bool
	TTL    time.Duration
}

var ErrNotFound = errors.New("会话不存在或已过期")

type Session struct {
	UserID    string    `json:"user_id"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func (m *Manager) Create(ctx context.Context, data Session) (sessionID string, err error) {
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return "", err
	}

	sessionID = base64.RawURLEncoding.EncodeToString(token)

	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now().UTC()
	}

	value, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	err = m.Valkey.Set(ctx, sessionKey(sessionID), value, m.TTL).Err()
	if err != nil {
		return "", err
	}

	return
}

// Get 读取 Session，并续期
func (m *Manager) Get(ctx context.Context, sessionID string) (session *Session, err error) {
	if !validID(sessionID) {
		return nil, ErrNotFound
	}

	value, err := m.Valkey.GetEx(ctx, sessionKey(sessionID), m.TTL).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(value, &session)
	return
}

func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	return m.Valkey.Del(ctx, sessionKey(sessionID)).Err()
}

func validID(id string) bool {
	if len(id) != base64.RawURLEncoding.EncodedLen(32) {
		return false
	}

	token, err := base64.RawURLEncoding.DecodeString(id)
	return err == nil && len(token) == 32 && base64.RawURLEncoding.EncodeToString(token) == id
}

func sessionKey(sessionID string) string {
	hash := sha256.Sum256([]byte(sessionID))
	return "session:" + hex.EncodeToString(hash[:])
}
