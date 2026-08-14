package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Session struct {
	Server    string    `json:"server"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Tenant    string    `json:"tenant"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

const expiryMargin = 30 * time.Second

func (s *Session) Valid(server string) bool {
	if s.Token == "" {
		return false
	}

	if s.Server != server {
		return false
	}

	return time.Now().Add(expiryMargin).Before(s.ExpiresAt)
}

func TokenExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("auth: jwt must have 3 segments, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("auth: jwt payload is not valid base64url: %w", err)
		}
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("auth: jwt payload is not valid JSON: %w", err)
	}

	raw, ok := claims["exp"]
	if !ok {
		return time.Time{}, errors.New("auth: jwt payload has no exp claim")
	}

	var exp int64
	if err := json.Unmarshal(raw, &exp); err != nil {
		return time.Time{}, errors.New("auth: jwt exp claim is not a number")
	}

	return time.Unix(exp, 0).UTC(), nil
}
