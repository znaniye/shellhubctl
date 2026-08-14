package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

type LoginFailure struct {
	Msg string
}

func (e *LoginFailure) Error() string {
	return e.Msg
}

func LoginWithPassword(ctx context.Context, c *shellhub.Client, identifier, password string) (*Session, error) {
	ua, err := c.Login(ctx, identifier, password)
	if err != nil {
		return nil, loginFailure(err)
	}

	c.SetToken(ua.Token)

	expiresAt, err := TokenExpiry(ua.Token)
	if err != nil {
		return nil, fmt.Errorf("auth: parse token expiry: %w", err)
	}

	return &Session{
		Server:    c.Server(),
		Token:     ua.Token,
		UserID:    ua.ID,
		Username:  ua.User,
		Email:     ua.Email,
		Tenant:    ua.Tenant,
		Role:      ua.Role,
		ExpiresAt: expiresAt,
	}, nil
}

func loginFailure(err error) error {
	var le *shellhub.LoginError
	if !errors.As(err, &le) {
		return err
	}

	var msg string

	switch {
	case !le.LockoutUntil.IsZero() && le.LockoutUntil.After(time.Now()):
		msg = fmt.Sprintf("too many attempts; try again at %s", le.LockoutUntil.Format("15:04"))
	case le.Status == http.StatusTooManyRequests:
		msg = "too many attempts; try again later"
	case le.Status == http.StatusUnauthorized && le.MFAToken != "":
		msg = "this account requires MFA, which this ShellHub instance does not support"
	case le.Status == http.StatusUnauthorized:
		msg = "invalid username or password"
	case le.Status == http.StatusForbidden:
		msg = "email not confirmed"
	case le.Status == http.StatusLocked:
		msg = "account waiting for administrator approval"
	case le.Status == http.StatusInternalServerError:
		msg = "server failure; check whether local login is enabled"
	default:
		msg = le.Error()
	}

	return &LoginFailure{Msg: msg}
}
