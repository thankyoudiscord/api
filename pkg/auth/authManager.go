package auth

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const SESSION_ID_COOKIE = "session_id"
const SESSION_TTL = time.Hour * 24 * 7

var mgr AuthManager

type AuthManager struct {
	RedisClient *redis.Client
}

type Session struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	UserID       string `json:"user_id"`
}

func (m AuthManager) CreateSession(s Session) (string, error) {
	sessionId := uuid.New().String()

	var sess bytes.Buffer
	enc := gob.NewEncoder(&sess)

	err := enc.Encode(s)
	if err != nil {
		fmt.Printf("failed to encode session data for id=%v: %v\n", sessionId, err)
		return "", err
	}

	status := m.RedisClient.SetEX(context.Background(), sessionId, sess.Bytes(), SESSION_TTL)
	if status.Err() != nil {
		return "", status.Err()
	}

	return sessionId, nil
}

func (m AuthManager) DeleteSession(id string) error {
	res := m.RedisClient.Del(context.Background(), id)

	return res.Err()
}

func (m AuthManager) GetSession(id string) (*Session, error) {
	res := m.RedisClient.Get(context.Background(), id)
	if res.Err() != nil {
		if res.Err() == redis.Nil {
			return nil, nil
		}

		return nil, res.Err()
	}

	b, err := res.Bytes()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var sess Session
	err = dec.Decode(&sess)
	if err != nil {
		fmt.Printf("failed to decode session id=%v: %v\n", id, err)
		return nil, err
	}

	return &sess, nil
}

var initOnce sync.Once

func InitAuthManager(r *redis.Client) {
	initOnce.Do(func() {
		mgr = AuthManager{
			RedisClient: r,
		}
	})
}

func GetManager() AuthManager {
	return mgr
}
