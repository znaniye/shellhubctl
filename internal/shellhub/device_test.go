package shellhub

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"
)

func TestListAllDevicesAcrossPages(t *testing.T) {
	var (
		mu    sync.Mutex
		pages []string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")

		mu.Lock()

		pages = append(pages, page)
		mu.Unlock()

		var body string

		switch page {
		case "1":
			body = `[{"uid":"d-1"},{"uid":"d-2"}]`
		case "2":
			body = `[{"uid":"d-3"},{"uid":"d-4"}]`
		case "3":
			body = `[{"uid":"d-5"}]`
		default:
			t.Errorf("unexpected page %q", page)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("X-Total-Count", "5")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, body)
	})

	devices, err := c.ListAllDevices(context.Background(), DeviceListOptions{PerPage: 2})
	if err != nil {
		t.Fatalf("ListAllDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if want := []string{"1", "2", "3"}; !slices.Equal(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}

	if len(devices) != 5 {
		t.Fatalf("len(devices) = %d, want 5", len(devices))
	}

	for i, want := range []string{"d-1", "d-2", "d-3", "d-4", "d-5"} {
		if devices[i].UID != want {
			t.Errorf("devices[%d].UID = %q, want %q", i, devices[i].UID, want)
		}
	}
}

func TestListAllDevicesStopsAtTotalCount(t *testing.T) {
	var (
		mu    sync.Mutex
		pages []string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")

		mu.Lock()

		pages = append(pages, page)
		mu.Unlock()

		w.Header().Set("X-Total-Count", "3")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[{"uid":"d-1"},{"uid":"d-2"}]`)
	})

	devices, err := c.ListAllDevices(context.Background(), DeviceListOptions{PerPage: 2})
	if err != nil {
		t.Fatalf("ListAllDevices: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if want := []string{"1", "2"}; !slices.Equal(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}

	if len(devices) != 4 {
		t.Errorf("len(devices) = %d, want 4", len(devices))
	}
}

func TestListAllDevicesEmpty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[]`)
	})

	devices, err := c.ListAllDevices(context.Background(), DeviceListOptions{})
	if err != nil {
		t.Fatalf("ListAllDevices: %v", err)
	}

	if devices == nil {
		t.Fatal("ListAllDevices: nil slice, want non-nil empty")
	}

	if len(devices) != 0 {
		t.Errorf("len(devices) = %d, want 0", len(devices))
	}
}

func TestListAllDevicesErrorPropagated(t *testing.T) {
	var (
		mu    sync.Mutex
		pages []string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")

		mu.Lock()

		pages = append(pages, page)
		mu.Unlock()

		if page == "2" {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("X-Total-Count", "10")
		w.Header().Set("Content-Type", "application/json")
		writeBody(w, `[{"uid":"d-1"},{"uid":"d-2"}]`)
	})

	_, err := c.ListAllDevices(context.Background(), DeviceListOptions{PerPage: 2})
	if err == nil {
		t.Fatal("ListAllDevices: expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As: got %T, want *APIError", err)
	}

	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", apiErr.Status, http.StatusInternalServerError)
	}

	mu.Lock()
	defer mu.Unlock()

	if want := []string{"1", "2"}; !slices.Equal(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
}
