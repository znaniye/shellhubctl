package shellhub

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrLocked          = errors.New("locked")
	ErrTooManyRequests = errors.New("too many requests")
)

type APIError struct {
	Status int
	Method string
	Path   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("shellhub: %s %s: %s (status %d)", e.Method, e.Path, http.StatusText(e.Status), e.Status)
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == http.StatusUnauthorized
	case ErrForbidden:
		return e.Status == http.StatusForbidden
	case ErrNotFound:
		return e.Status == http.StatusNotFound
	case ErrLocked:
		return e.Status == http.StatusLocked
	case ErrTooManyRequests:
		return e.Status == http.StatusTooManyRequests
	}

	return false
}
