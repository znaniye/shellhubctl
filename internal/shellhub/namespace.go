package shellhub

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Namespace struct {
	Name                 string    `json:"name"`
	Owner                string    `json:"owner"`
	TenantID             string    `json:"tenant_id"`
	DevicesAcceptedCount int       `json:"devices_accepted_count"`
	DevicesPendingCount  int       `json:"devices_pending_count"`
	DevicesRejectedCount int       `json:"devices_rejected_count"`
	MaxDevices           int       `json:"max_devices"`
	CreatedAt            time.Time `json:"created_at"`
}

func (c *Client) ListNamespaces(ctx context.Context, page, perPage int) ([]Namespace, int, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(normalizePage(page)))
	query.Set("per_page", strconv.Itoa(normalizePerPage(perPage)))

	res, err := c.do(ctx, http.MethodGet, "/api/namespaces", query, nil)
	if err != nil {
		return nil, 0, err
	}

	var namespaces []Namespace

	if err := res.decode(&namespaces); err != nil {
		return nil, 0, err
	}

	return namespaces, res.totalCount(), nil
}
