package shellhub

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New(%q): %v", srv.URL, err)
	}

	return c
}

func writeBody(w http.ResponseWriter, body string) {
	_, _ = io.WriteString(w, body)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewInvalidURL(t *testing.T) {
	if _, err := New("http://exa mple.com"); err == nil {
		t.Fatal("New with invalid URL: nil error")
	}
}

func TestServer(t *testing.T) {
	c, err := New("http://example.com:8080/")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := c.Server(); got != "http://example.com:8080/" {
		t.Errorf("Server() = %q", got)
	}
}

func TestSetTokenAndToken(t *testing.T) {
	c, err := New("http://example.com")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := c.Token(); got != "" {
		t.Errorf("Token() = %q, want empty", got)
	}

	c.SetToken("abc")

	if got := c.Token(); got != "abc" {
		t.Errorf("Token() = %q, want abc", got)
	}
}

func TestWithHTTPClient(t *testing.T) {
	var mu sync.Mutex

	called := false

	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		called = true
		mu.Unlock()

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"version":"v0.0.0"}`)),
			Request:    req,
		}, nil
	})

	c, err := New("http://example.com", WithHTTPClient(&http.Client{Transport: rt}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.Info(context.Background()); err != nil {
		t.Fatalf("Info: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Fatal("custom transport was not used")
	}
}

func TestInfoAnonymous(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Errorf("path = %q, want /info", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty", got)
		}

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `{"version":"v0.29.0","endpoints":{"api":"https://api.example.com","ssh":"ssh.example.com:22"},"setup":true,"authentication":{"local":true}}`)
	})

	c.SetToken("secret-token")

	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Version != "v0.29.0" {
		t.Errorf("Version = %q", info.Version)
	}

	if info.Endpoints.API != "https://api.example.com" || info.Endpoints.SSH != "ssh.example.com:22" {
		t.Errorf("Endpoints = %+v", info.Endpoints)
	}

	if !info.Setup || !info.Authentication.Local {
		t.Errorf("Setup = %v, Authentication = %+v", info.Setup, info.Authentication)
	}
}

func TestListNamespaces(t *testing.T) {
	var (
		mu       sync.Mutex
		gotQuery url.Values
		gotAuth  string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/namespaces" {
			t.Errorf("path = %q, want /api/namespaces", r.URL.Path)
		}

		mu.Lock()
		gotQuery = r.URL.Query()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("X-Total-Count", "7")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[{"name":"alpha","owner":"alice","tenant_id":"t-1","devices_accepted_count":2,"devices_pending_count":1,"devices_rejected_count":0,"max_devices":10,"created_at":"2024-01-02T03:04:05Z"},{"name":"beta","tenant_id":"t-2"}]`)
	})

	c.SetToken("tok")

	namespaces, total, err := c.ListNamespaces(context.Background(), 2, 25)
	if err != nil {
		t.Fatalf("ListNamespaces: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if total != 7 {
		t.Errorf("total = %d, want 7", total)
	}

	if len(namespaces) != 2 {
		t.Fatalf("len(namespaces) = %d, want 2", len(namespaces))
	}

	if namespaces[0].Name != "alpha" || namespaces[0].TenantID != "t-1" || namespaces[0].MaxDevices != 10 {
		t.Errorf("namespaces[0] = %+v", namespaces[0])
	}

	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
	}

	if gotQuery.Get("page") != "2" || gotQuery.Get("per_page") != "25" {
		t.Errorf("query = %v", gotQuery)
	}
}

func TestListDevicesDefaultStatus(t *testing.T) {
	var (
		mu       sync.Mutex
		gotQuery url.Values
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/devices" {
			t.Errorf("path = %q, want /api/devices", r.URL.Path)
		}

		mu.Lock()
		gotQuery = r.URL.Query()
		mu.Unlock()

		w.Header().Set("X-Total-Count", "3")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[{"uid":"d-1","name":"dev1","online":true,"status":"accepted","info":{"id":"i-1","pretty_name":"Dev One","version":"v1","arch":"arm64","platform":"linux"},"identity":{"mac":"aa:bb:cc:dd:ee:ff"},"tags":[{"tenant_id":"t-1","name":"a","created_at":"2024-01-02T03:04:05Z","updated_at":"2024-01-02T03:04:05Z"},{"tenant_id":"t-1","name":"b"}]}]`)
	})

	c.SetToken("tok")

	devices, total, err := c.ListDevices(context.Background(), DeviceListOptions{})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if got := gotQuery.Get("status"); got != "accepted" {
		t.Errorf("status = %q, want accepted", got)
	}

	if got := gotQuery.Get("page"); got != "1" {
		t.Errorf("page = %q, want 1", got)
	}

	if got := gotQuery.Get("per_page"); got != "100" {
		t.Errorf("per_page = %q, want 100", got)
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}

	if devices[0].UID != "d-1" || !devices[0].Online || devices[0].Info.PrettyName != "Dev One" || devices[0].Identity.MAC != "aa:bb:cc:dd:ee:ff" || len(devices[0].Tags) != 2 {
		t.Errorf("devices[0] = %+v", devices[0])
	}

	if devices[0].Tags[0].Name != "a" || devices[0].Tags[0].TenantID != "t-1" {
		t.Errorf("devices[0].Tags[0] = %+v", devices[0].Tags[0])
	}

	if !devices[0].Tags[0].CreatedAt.Equal(time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("devices[0].Tags[0].CreatedAt = %v", devices[0].Tags[0].CreatedAt)
	}

	if !devices[0].Tags[1].CreatedAt.IsZero() || !devices[0].Tags[1].UpdatedAt.IsZero() {
		t.Errorf("devices[0].Tags[1] = %+v", devices[0].Tags[1])
	}

	if names := devices[0].TagNames(); len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("TagNames() = %v", names)
	}
}

func TestListDevicesTagShapes(t *testing.T) {
	body := `[{"uid":"d-1","tags":[{"tenant_id":"t-1","name":"prod"}]},{"uid":"d-2","tags":["legacy","old"]},{"uid":"d-3","tags":null},{"uid":"d-4"}]`

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, body)
	})

	devices, _, err := c.ListDevices(context.Background(), DeviceListOptions{})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	if len(devices) != 4 {
		t.Fatalf("len(devices) = %d, want 4", len(devices))
	}

	if len(devices[0].Tags) != 1 || devices[0].Tags[0].Name != "prod" || devices[0].Tags[0].TenantID != "t-1" {
		t.Errorf("devices[0].Tags = %+v", devices[0].Tags)
	}

	if names := devices[1].TagNames(); len(names) != 2 || names[0] != "legacy" || names[1] != "old" {
		t.Errorf("devices[1].TagNames() = %v", names)
	}

	for _, device := range devices[2:] {
		if device.Tags != nil {
			t.Errorf("%s Tags = %+v, want nil", device.UID, device.Tags)
		}

		if names := device.TagNames(); len(names) != 0 {
			t.Errorf("%s TagNames() = %v, want empty", device.UID, names)
		}
	}
}

func TestTagUnmarshalInvalid(t *testing.T) {
	var tag Tag

	if err := json.Unmarshal([]byte(`42`), &tag); err == nil {
		t.Fatal("Unmarshal: expected error")
	}
}

func TestListDevicesSortAndClamp(t *testing.T) {
	var (
		mu       sync.Mutex
		gotQuery url.Values
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotQuery = r.URL.Query()
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[]`)
	})

	_, _, err := c.ListDevices(context.Background(), DeviceListOptions{
		Status:  "pending",
		Page:    -1,
		PerPage: 0,
		SortBy:  "last_seen",
		OrderBy: "desc",
	})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if got := gotQuery.Get("status"); got != "pending" {
		t.Errorf("status = %q, want pending", got)
	}

	if got := gotQuery.Get("page"); got != "1" {
		t.Errorf("page = %q, want 1", got)
	}

	if got := gotQuery.Get("per_page"); got != "100" {
		t.Errorf("per_page = %q, want 100", got)
	}

	if got := gotQuery.Get("sort_by"); got != "last_seen" {
		t.Errorf("sort_by = %q, want last_seen", got)
	}

	if got := gotQuery.Get("order_by"); got != "desc" {
		t.Errorf("order_by = %q, want desc", got)
	}
}

func TestListDevicesEmptyBodyError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	_, _, err := c.ListDevices(context.Background(), DeviceListOptions{})
	if err == nil {
		t.Fatal("ListDevices: expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As: got %T, want *APIError", err)
	}

	if apiErr.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", apiErr.Status, http.StatusForbidden)
	}

	if apiErr.Method != http.MethodGet || apiErr.Path != "/api/devices" {
		t.Errorf("APIError = %+v", apiErr)
	}

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("errors.Is(err, ErrForbidden) = false")
	}

	if errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = true")
	}
}

func TestAPIErrorSentinels(t *testing.T) {
	tests := []struct {
		status   int
		sentinel error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusLocked, ErrLocked},
		{http.StatusTooManyRequests, ErrTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			err := &APIError{Status: tt.status, Method: http.MethodGet, Path: "/api/x"}

			if !errors.Is(err, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false", err, tt.sentinel)
			}
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	var (
		mu      sync.Mutex
		gotBody map[string]string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}

		if r.URL.Path != "/api/login" {
			t.Errorf("path = %q, want /api/login", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}

		mu.Lock()
		gotBody = body
		mu.Unlock()

		w.Header().Set("X-MFA-Token", "")
		w.Header().Set("X-Account-Lockout", "0")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `{"token":"tk-1","user":"alice","name":"Alice","id":"u-1","tenant":"ns-1","email":"alice@example.com","role":"owner","mfa":false}`)
	})

	auth, err := c.Login(context.Background(), "alice@example.com", "s3cret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotBody["username"] != "alice@example.com" || gotBody["password"] != "s3cret" {
		t.Errorf("body = %v", gotBody)
	}

	if auth.Token != "tk-1" || auth.User != "alice" || auth.Name != "Alice" || auth.ID != "u-1" || auth.Tenant != "ns-1" || auth.Email != "alice@example.com" || auth.Role != "owner" || auth.MFA {
		t.Errorf("auth = %+v", auth)
	}
}

func TestLoginMFARequired(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-MFA-Token", "mfa-token-1")
		w.Header().Set("X-Account-Lockout", "0")
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := c.Login(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("Login: expected error")
	}

	var le *LoginError
	if !errors.As(err, &le) {
		t.Fatalf("errors.As: got %T, want *LoginError", err)
	}

	if le.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", le.Status, http.StatusUnauthorized)
	}

	if le.MFAToken != "mfa-token-1" {
		t.Errorf("MFAToken = %q, want mfa-token-1", le.MFAToken)
	}

	if !le.LockoutUntil.IsZero() {
		t.Errorf("LockoutUntil = %v, want zero", le.LockoutUntil)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-MFA-Token", "")
		w.Header().Set("X-Account-Lockout", "0")
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := c.Login(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("Login: expected error")
	}

	var le *LoginError
	if !errors.As(err, &le) {
		t.Fatalf("errors.As: got %T, want *LoginError", err)
	}

	if le.MFAToken != "" {
		t.Errorf("MFAToken = %q, want empty", le.MFAToken)
	}

	if !le.LockoutUntil.IsZero() {
		t.Errorf("LockoutUntil = %v, want zero", le.LockoutUntil)
	}
}

func TestLoginLockout(t *testing.T) {
	until := time.Now().Add(30 * time.Minute).Truncate(time.Second)

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Account-Lockout", strconv.FormatInt(until.Unix(), 10))
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := c.Login(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("Login: expected error")
	}

	var le *LoginError
	if !errors.As(err, &le) {
		t.Fatalf("errors.As: got %T, want *LoginError", err)
	}

	if le.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", le.Status, http.StatusTooManyRequests)
	}

	if !le.LockoutUntil.Equal(until) {
		t.Errorf("LockoutUntil = %v, want %v", le.LockoutUntil, until)
	}
}

func TestLoginErrorStatuses(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusLocked, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			_, err := c.Login(context.Background(), "alice", "pw")
			if err == nil {
				t.Fatal("Login: expected error")
			}

			var le *LoginError
			if !errors.As(err, &le) {
				t.Fatalf("errors.As: got %T, want *LoginError", err)
			}

			if le.Status != status {
				t.Errorf("Status = %d, want %d", le.Status, status)
			}
		})
	}
}

func TestSwitchNamespace(t *testing.T) {
	var (
		mu      sync.Mutex
		gotPath string
		gotAuth string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `{"token":"new-token","user":"alice","name":"Alice","id":"u-1","tenant":"ns-2","email":"alice@example.com","role":"owner","mfa":false}`)
	})

	c.SetToken("old-token")

	auth, err := c.SwitchNamespace(context.Background(), "ns-2")
	if err != nil {
		t.Fatalf("SwitchNamespace: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotPath != "/api/auth/token/ns-2" {
		t.Errorf("path = %q, want /api/auth/token/ns-2", gotPath)
	}

	if gotAuth != "Bearer old-token" {
		t.Errorf("Authorization = %q, want Bearer old-token", gotAuth)
	}

	if got := c.Token(); got != "new-token" {
		t.Errorf("Token() = %q, want new-token", got)
	}

	if auth.Tenant != "ns-2" || auth.Token != "new-token" {
		t.Errorf("auth = %+v", auth)
	}
}

func TestDoRequestAPIKey(t *testing.T) {
	var (
		mu        sync.Mutex
		gotAPIKey string
		gotAuth   string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAPIKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[]`)
	})

	c.SetAPIKey("key-1")

	if _, _, err := c.ListDevices(context.Background(), DeviceListOptions{}); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotAPIKey != "key-1" {
		t.Errorf("X-API-Key = %q, want key-1", gotAPIKey)
	}

	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty", gotAuth)
	}
}

func TestDoRequestTokenOnly(t *testing.T) {
	var (
		mu        sync.Mutex
		gotAPIKey string
		gotAuth   string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAPIKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[]`)
	})

	c.SetToken("tok")

	if _, _, err := c.ListDevices(context.Background(), DeviceListOptions{}); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
	}

	if gotAPIKey != "" {
		t.Errorf("X-API-Key = %q, want empty", gotAPIKey)
	}
}

func TestDoRequestAPIKeyPrecedence(t *testing.T) {
	var (
		mu        sync.Mutex
		gotAPIKey string
		gotAuth   string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAPIKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[]`)
	})

	c.SetToken("tok")
	c.SetAPIKey("key-1")

	if _, _, err := c.ListDevices(context.Background(), DeviceListOptions{}); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotAPIKey != "key-1" {
		t.Errorf("X-API-Key = %q, want key-1", gotAPIKey)
	}

	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty", gotAuth)
	}
}

func TestDoAnonymousSendsNoCredentials(t *testing.T) {
	var (
		mu        sync.Mutex
		gotAPIKey string
		gotAuth   string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Errorf("path = %q, want /info", r.URL.Path)
		}

		mu.Lock()
		gotAPIKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `{"version":"v0.29.0","endpoints":{"api":"https://api.example.com","ssh":"ssh.example.com:22"},"setup":true,"authentication":{"local":true}}`)
	})

	c.SetToken("tok")
	c.SetAPIKey("key-1")

	if _, err := c.Info(context.Background()); err != nil {
		t.Fatalf("Info: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotAPIKey != "" {
		t.Errorf("X-API-Key = %q, want empty", gotAPIKey)
	}

	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty", gotAuth)
	}
}

func TestHostDropsTheAPIPort(t *testing.T) {
	tests := []struct {
		server string
		want   string
	}{
		{server: "https://cloud.shellhub.io", want: "cloud.shellhub.io"},
		{server: "http://localhost", want: "localhost"},
		{server: "http://localhost:8080", want: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.server, func(t *testing.T) {
			c, err := New(tt.server)
			if err != nil {
				t.Fatalf("New(%q): %v", tt.server, err)
			}

			if got := c.Host(); got != tt.want {
				t.Errorf("Host() = %q, want %q", got, tt.want)
			}
		})
	}
}
