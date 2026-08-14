package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func buildToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
}

func TestTokenExpiry(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Unix()
	token := buildToken(t, map[string]any{"exp": exp})

	got, err := TokenExpiry(token)
	if err != nil {
		t.Fatalf("TokenExpiry() error = %v", err)
	}

	want := time.Unix(exp, 0).UTC()
	if !got.Equal(want) {
		t.Errorf("TokenExpiry() = %v, want %v", got, want)
	}
}

func TestTokenExpiryAcceptsPaddedPayload(t *testing.T) {
	const exp = int64(123456789)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp)))
	token := header + "." + payload + ".ZmFrZQ"

	got, err := TokenExpiry(token)
	if err != nil {
		t.Fatalf("TokenExpiry() error = %v", err)
	}

	want := time.Unix(exp, 0).UTC()
	if !got.Equal(want) {
		t.Errorf("TokenExpiry() = %v, want %v", got, want)
	}
}

func TestTokenExpiryMalformed(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "one segment",
			token: "abc",
		},
		{
			name:  "two segments",
			token: "a.b",
		},
		{
			name:  "invalid base64 payload",
			token: "eyJhbGciOiJub25lIn0.!!!.ZmFrZQ",
		},
		{
			name:  "payload is not JSON",
			token: "eyJhbGciOiJub25lIn0.bm90IGpzb24.ZmFrZQ",
		},
		{
			name:  "missing exp claim",
			token: buildToken(t, map[string]any{"sub": "user-1"}),
		},
		{
			name:  "exp is not a number",
			token: buildToken(t, map[string]any{"exp": "tomorrow"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := TokenExpiry(tt.token); err == nil {
				t.Errorf("TokenExpiry(%q) expected error, got nil", tt.token)
			}
		})
	}
}

func TestSessionValid(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		sess   *Session
		server string
		want   bool
	}{
		{
			name:   "empty token",
			sess:   &Session{Server: "https://s.example", ExpiresAt: future},
			server: "https://s.example",
			want:   false,
		},
		{
			name:   "different server",
			sess:   &Session{Server: "https://a.example", Token: "tok", ExpiresAt: future},
			server: "https://b.example",
			want:   false,
		},
		{
			name:   "expired",
			sess:   &Session{Server: "https://s.example", Token: "tok", ExpiresAt: now.Add(-time.Minute)},
			server: "https://s.example",
			want:   false,
		},
		{
			name:   "expires in two seconds",
			sess:   &Session{Server: "https://s.example", Token: "tok", ExpiresAt: now.Add(2 * time.Second)},
			server: "https://s.example",
			want:   false,
		},
		{
			name:   "valid for one hour",
			sess:   &Session{Server: "https://s.example", Token: "tok", ExpiresAt: now.Add(time.Hour)},
			server: "https://s.example",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sess.Valid(tt.server); got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
