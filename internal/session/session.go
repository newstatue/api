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

const TTL = 7 * 24 * time.Hour

type Manager struct {
	Valkey *redis.Client
}

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

	err = m.Valkey.Set(ctx, sessionKey(sessionID), value, TTL).Err()
	if err != nil {
		return "", err
	}

	return
}

func (m *Manager) Get(ctx context.Context, sessionID string) (data Session, err error) {
	value, err := m.Valkey.Get(ctx, sessionKey(sessionID)).Bytes()
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(value, &data)
	return
}

func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	return m.Valkey.Del(ctx, sessionKey(sessionID)).Err()
}

func (m *Manager) Refresh(ctx context.Context, sessionID string) (refreshed bool, err error) {
	return m.Valkey.Expire(ctx, sessionKey(sessionID), TTL).Result()
}

func (m *Manager) GetOrCreate(ctx context.Context, sessionID string, data Session) (string, error) {
	if sessionID != "" {
		oldSession, err := m.Get(ctx, sessionID)

		if err != nil && !errors.Is(err, redis.Nil) {
			return "", err
		}

		if err == nil && oldSession.UserID == data.UserID {
			refreshed, err := m.Refresh(ctx, sessionID)
			if err != nil {
				return "", err
			}

			if refreshed {
				return sessionID, nil
			}
		}
	}

	return m.Create(ctx, data)
}

func sessionKey(sessionID string) string {
	hash := sha256.Sum256([]byte(sessionID))
	return "session:" + hex.EncodeToString(hash[:])
}
