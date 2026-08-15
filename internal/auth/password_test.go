package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func loginServer(t *testing.T, handler http.HandlerFunc) *shellhub.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := shellhub.New(srv.URL)
	if err != nil {
		t.Fatalf("shellhub.New(%q): %v", srv.URL, err)
	}

	return c
}

func TestLoginWithPasswordSuccess(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	token := buildToken(t, map[string]any{"exp": exp.Unix()})

	c := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/login" {
			t.Errorf("path = %q, want /api/login", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")

		body, err := json.Marshal(map[string]any{
			"token": token, "user": "alice", "name": "Alice", "id": "u-1",
			"tenant": "ns-1", "email": "alice@example.com", "role": "owner", "mfa": false,
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)

			return
		}

		_, _ = w.Write(body)
	})

	sess, err := LoginWithPassword(context.Background(), c, "alice", "s3cret")
	if err != nil {
		t.Fatalf("LoginWithPassword: %v", err)
	}

	if sess.Token != token {
		t.Errorf("Token = %q, want %q", sess.Token, token)
	}

	if sess.UserID != "u-1" || sess.Username != "alice" || sess.Email != "alice@example.com" || sess.Tenant != "ns-1" || sess.Role != "owner" {
		t.Errorf("Session = %+v", sess)
	}

	if sess.Server != c.Server() {
		t.Errorf("Server = %q, want %q", sess.Server, c.Server())
	}

	if !sess.ExpiresAt.Equal(exp.UTC()) {
		t.Errorf("ExpiresAt = %v, want %v", sess.ExpiresAt, exp.UTC())
	}

	if got := c.Token(); got != token {
		t.Errorf("client token = %q, want %q", got, token)
	}
}

func TestLoginWithPasswordInvalidToken(t *testing.T) {
	c := loginServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := json.Marshal(map[string]any{
			"token": "a.b.c", "user": "alice", "name": "Alice", "id": "u-1",
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)

			return
		}

		_, _ = w.Write(body)
	})

	_, err := LoginWithPassword(context.Background(), c, "alice", "s3cret")
	if err == nil {
		t.Fatal("LoginWithPassword: expected error for unparseable token")
	}

	var lf *LoginFailure
	if errors.As(err, &lf) {
		t.Fatalf("LoginWithPassword: got %q, want a token expiry error", lf.Msg)
	}
}

func TestLoginWithPasswordFailureStatuses(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, "invalid username or password"},
		{http.StatusForbidden, "email not confirmed"},
		{http.StatusLocked, "account waiting for administrator approval"},
		{http.StatusInternalServerError, "server failure; check whether local login is enabled"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status-%d", tt.status), func(t *testing.T) {
			c := loginServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})

			_, err := LoginWithPassword(context.Background(), c, "alice", "pw")
			if err == nil {
				t.Fatal("LoginWithPassword: expected error")
			}

			var lf *LoginFailure
			if !errors.As(err, &lf) {
				t.Fatalf("errors.As: got %T, want *LoginFailure", err)
			}

			if lf.Msg != tt.want {
				t.Errorf("Msg = %q, want %q", lf.Msg, tt.want)
			}
		})
	}
}

func TestLoginFailureMessages(t *testing.T) {
	until := time.Now().Add(30 * time.Minute).Truncate(time.Minute)

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "too many requests with lockout time",
			err:  &shellhub.LoginError{Status: http.StatusTooManyRequests, LockoutUntil: until},
			want: "too many attempts; try again at " + until.Format("15:04"),
		},
		{
			name: "lockout on any status",
			err:  &shellhub.LoginError{Status: http.StatusUnauthorized, LockoutUntil: until},
			want: "too many attempts; try again at " + until.Format("15:04"),
		},
		{
			name: "too many requests without lockout time",
			err:  &shellhub.LoginError{Status: http.StatusTooManyRequests},
			want: "too many attempts; try again later",
		},
		{
			name: "mfa required",
			err:  &shellhub.LoginError{Status: http.StatusUnauthorized, MFAToken: "mfa-token"},
			want: "this account requires MFA, which this ShellHub instance does not support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loginFailure(tt.err)

			var lf *LoginFailure
			if !errors.As(err, &lf) {
				t.Fatalf("loginFailure(%v): got %T, want *LoginFailure", tt.err, err)
			}

			if lf.Msg != tt.want {
				t.Errorf("Msg = %q, want %q", lf.Msg, tt.want)
			}
		})
	}
}

func TestLoginFailurePassesThroughOtherErrors(t *testing.T) {
	sentinel := errors.New("boom")

	if got := loginFailure(sentinel); !errors.Is(got, sentinel) {
		t.Errorf("loginFailure(sentinel) = %v, want the original error", got)
	}
}
