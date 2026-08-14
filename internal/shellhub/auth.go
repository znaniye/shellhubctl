package shellhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type UserAuth struct {
	Token  string `json:"token"`
	User   string `json:"user"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Tenant string `json:"tenant"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	MFA    bool   `json:"mfa"`
}

type LoginError struct {
	Status       int
	MFAToken     string
	LockoutUntil time.Time
}

func (e *LoginError) Error() string {
	msg := http.StatusText(e.Status)

	switch e.Status {
	case http.StatusUnauthorized:
		if e.MFAToken != "" {
			msg = "mfa required"
		} else {
			msg = "invalid credentials"
		}
	case http.StatusForbidden:
		msg = "email not confirmed"
	case http.StatusLocked:
		msg = "account pending approval"
	case http.StatusTooManyRequests:
		msg = "account locked"
	case http.StatusInternalServerError:
		msg = "local login disabled"
	}

	return fmt.Sprintf("shellhub: login: %s (status %d)", msg, e.Status)
}

func (c *Client) Login(ctx context.Context, identifier, password string) (*UserAuth, error) {
	payload, err := json.Marshal(map[string]string{
		"username": identifier,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	res, err := c.doRaw(ctx, http.MethodPost, "/api/login", nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if res.status != http.StatusOK {
		return nil, res.loginError()
	}

	var auth UserAuth

	if err := res.decode(&auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func (c *Client) SwitchNamespace(ctx context.Context, tenantID string) (*UserAuth, error) {
	path := "/api/auth/token/" + url.PathEscape(tenantID)

	res, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var auth UserAuth

	if err := res.decode(&auth); err != nil {
		return nil, err
	}

	c.SetToken(auth.Token)

	return &auth, nil
}

func (r *response) loginError() *LoginError {
	e := &LoginError{Status: r.status}
	e.MFAToken = r.header.Get("X-MFA-Token")

	raw := r.header.Get("X-Account-Lockout")
	if raw == "" || raw == "0" {
		return e
	}

	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return e
	}

	e.LockoutUntil = time.Unix(sec, 0).UTC()

	return e
}
