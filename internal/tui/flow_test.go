package tui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func testToken(t *testing.T, exp time.Time) string {
	t.Helper()

	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	payload, err := json.Marshal(map[string]any{"exp": exp.Unix()})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
}

func testModel(t *testing.T, handler http.HandlerFunc, sess *auth.Session) (model, *auth.Store) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := shellhub.New(srv.URL)
	if err != nil {
		t.Fatalf("shellhub.New: %v", err)
	}

	store := auth.NewStoreAt(t.TempDir() + "/session.json")

	return newModel(context.Background(), Options{Client: c, Store: store, Session: sess}), store
}

func runMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	if cmd == nil {
		t.Fatal("runMsg: nil cmd")
	}

	msg := unwrap(cmd())
	if msg == nil {
		t.Fatal("runMsg: no meaningful message in batch")
	}

	return msg
}

func unwrap(msg tea.Msg) tea.Msg {
	switch m := msg.(type) {
	case nil:
		return nil
	case tea.BatchMsg:
		for _, sub := range m {
			if got := unwrap(sub()); got != nil {
				if _, tick := got.(spinner.TickMsg); !tick {
					return got
				}
			}
		}

		return nil
	case spinner.TickMsg:
		return nil
	default:
		return msg
	}
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func updateModel(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)

	nm, ok := updated.(model)
	if !ok {
		t.Fatalf("Update returned %T, want model", updated)
	}

	return nm, cmd
}

func TestModelLoginFlow(t *testing.T) {
	token := testToken(t, time.Now().Add(2*time.Hour))

	m, store := testModel(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/login" {
			t.Errorf("path = %q, want /api/login", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")

		body, err := json.Marshal(map[string]any{
			"token": token, "user": "alice", "name": "Alice", "id": "u-1",
			"tenant": "", "email": "alice@example.com", "role": "", "mfa": false,
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)

			return
		}

		_, _ = w.Write(body)
	}, nil)

	if m.screen != screenLogin {
		t.Fatalf("screen = %v, want login", m.screen)
	}

	m, _ = updateModel(t, m, keyMsg("alice"))
	if m.login.identifier.Value() != "alice" {
		t.Fatalf("identifier = %q, want alice", m.login.identifier.Value())
	}

	m, _ = updateModel(t, m, keyMsg("tab"))
	if m.login.focused != 1 {
		t.Fatalf("focused = %d, want password", m.login.focused)
	}

	m, _ = updateModel(t, m, keyMsg("s3cret"))
	if m.login.password.Value() != "s3cret" {
		t.Fatalf("password = %q, want s3cret", m.login.password.Value())
	}

	m, cmd := updateModel(t, m, keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter: expected login command")
	}

	if !m.login.loading {
		t.Fatal("login should be loading while authenticating")
	}

	msg := runMsg(t, cmd)

	success, ok := msg.(loginSuccessMsg)
	if !ok {
		t.Fatalf("login cmd returned %T, want loginSuccessMsg", msg)
	}

	if success.session.Token != token || success.session.Tenant != "" {
		t.Errorf("session = %+v", success.session)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}

	if got.Token != token {
		t.Errorf("stored token = %q, want %q", got.Token, token)
	}

	m, _ = updateModel(t, m, success)
	if m.screen != screenNamespaces {
		t.Fatalf("screen = %v, want namespaces", m.screen)
	}
}

func TestModelValidSessionFlow(t *testing.T) {
	newToken := testToken(t, time.Now().Add(2*time.Hour))

	var (
		mu          sync.Mutex
		devicesAuth string
	)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/namespaces":
			_, _ = w.Write([]byte(`[{"name":"alpha","tenant_id":"t-1","devices_accepted_count":2}]`))
		case "/api/auth/token/t-1":
			body, err := json.Marshal(map[string]any{
				"token": newToken, "user": "alice", "name": "Alice", "id": "u-1",
				"tenant": "t-1", "email": "alice@example.com", "role": "owner", "mfa": false,
			})
			if err != nil {
				t.Errorf("marshal response: %v", err)

				return
			}

			_, _ = w.Write(body)
		case "/api/devices":
			mu.Lock()
			devicesAuth = r.Header.Get("Authorization")
			mu.Unlock()

			w.Header().Set("X-Total-Count", "1")
			_, _ = w.Write([]byte(`[{"uid":"d-1","name":"dev1","online":true,"status":"accepted","info":{"platform":"linux","arch":"arm64"}}]`))
		}
	}

	oldToken := testToken(t, time.Now().Add(2*time.Hour))
	sess := &auth.Session{
		Server:    "http://placeholder",
		Token:     oldToken,
		Username:  "alice",
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	m, store := testModel(t, handler, sess)
	if m.screen != screenNamespaces {
		t.Fatalf("screen = %v, want namespaces", m.screen)
	}

	msg := runMsg(t, m.Init())

	loaded, ok := msg.(namespacesLoadedMsg)
	if !ok {
		t.Fatalf("init cmd returned %T, want namespacesLoadedMsg", msg)
	}

	if len(loaded.namespaces) != 1 || loaded.namespaces[0].Name != "alpha" {
		t.Errorf("namespaces = %+v", loaded.namespaces)
	}

	m, _ = updateModel(t, m, loaded)
	if !m.namespaces.hasList {
		t.Fatal("namespace list was not built")
	}

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter: expected switch command")
	}

	msg = runMsg(t, cmd)

	switched, ok := msg.(namespaceSelectedMsg)
	if !ok {
		t.Fatalf("switch cmd returned %T, want namespaceSelectedMsg", msg)
	}

	if switched.session.Token != newToken || switched.session.Tenant != "t-1" {
		t.Errorf("session = %+v", switched.session)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}

	if got.Token != newToken {
		t.Errorf("stored token = %q, want %q", got.Token, newToken)
	}

	m, cmd = updateModel(t, m, switched)
	if m.screen != screenDevices {
		t.Fatalf("screen = %v, want devices", m.screen)
	}

	if cmd == nil {
		t.Fatal("expected devices load command")
	}

	msg = runMsg(t, cmd)

	devLoaded, ok := msg.(devicesLoadedMsg)
	if !ok {
		t.Fatalf("devices cmd returned %T, want devicesLoadedMsg", msg)
	}

	if len(devLoaded.devices) != 1 || devLoaded.total != 1 {
		t.Errorf("devices = %+v, total = %d", devLoaded.devices, devLoaded.total)
	}

	m, _ = updateModel(t, m, devLoaded)
	if !m.devices.hasTable {
		t.Fatal("device table was not built")
	}

	mu.Lock()
	gotAuth := devicesAuth
	mu.Unlock()

	if gotAuth != "Bearer "+newToken {
		t.Errorf("devices request Authorization = %q, want Bearer %s", gotAuth, newToken)
	}

	m, cmd = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	msg = runMsg(t, cmd)

	if _, ok := msg.(backMsg); !ok {
		t.Fatalf("esc cmd returned %T, want backMsg", msg)
	}

	m, _ = updateModel(t, m, msg)
	if m.screen != screenNamespaces {
		t.Fatalf("screen = %v, want namespaces after esc", m.screen)
	}
}

func TestModelNetworkErrorDoesNotCrash(t *testing.T) {
	m, _ := testModel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}, &auth.Session{
		Server:    "http://placeholder",
		Token:     "irrelevant",
		ExpiresAt: time.Now().Add(2 * time.Hour),
	})

	msg := runMsg(t, m.Init())

	if errMsg, ok := msg.(errMsg); !ok {
		t.Fatalf("init cmd returned %T, want errMsg", msg)
	} else if errMsg.err == nil {
		t.Fatal("errMsg without error")
	}

	m, _ = updateModel(t, m, msg)

	if m.namespaces.err == "" {
		t.Fatal("namespaces screen did not record the error")
	}

	if cmd := m.namespaces.reload(); cmd == nil {
		t.Fatal("reload returned nil cmd")
	}
}
