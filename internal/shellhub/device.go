package shellhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Tag struct {
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *Tag) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		t.Name = name

		return nil
	}

	type tag Tag

	var decoded tag
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*t = Tag(decoded)

	return nil
}

type DeviceInfo struct {
	ID         string `json:"id"`
	PrettyName string `json:"pretty_name"`
	Version    string `json:"version"`
	Arch       string `json:"arch"`
	Platform   string `json:"platform"`
}

type DeviceIdentity struct {
	MAC string `json:"mac"`
}

type Device struct {
	UID       string         `json:"uid"`
	Name      string         `json:"name"`
	Online    bool           `json:"online"`
	Status    string         `json:"status"`
	Namespace string         `json:"namespace"`
	TenantID  string         `json:"tenant_id"`
	LastSeen  time.Time      `json:"last_seen"`
	CreatedAt time.Time      `json:"created_at"`
	Info      DeviceInfo     `json:"info"`
	Identity  DeviceIdentity `json:"identity"`
	Tags      []Tag          `json:"tags"`
}

func (d *Device) TagNames() []string {
	names := make([]string, 0, len(d.Tags))

	for _, tag := range d.Tags {
		if tag.Name == "" {
			continue
		}

		names = append(names, tag.Name)
	}

	return names
}

type DeviceListOptions struct {
	Status  string
	Page    int
	PerPage int
	SortBy  string
	OrderBy string
}

func (c *Client) ListDevices(ctx context.Context, opts DeviceListOptions) ([]Device, int, error) {
	status := opts.Status
	if status == "" {
		status = "accepted"
	}

	query := url.Values{}
	query.Set("status", status)
	query.Set("page", strconv.Itoa(normalizePage(opts.Page)))
	query.Set("per_page", strconv.Itoa(normalizePerPage(opts.PerPage)))

	if opts.SortBy != "" {
		query.Set("sort_by", opts.SortBy)
	}

	if opts.OrderBy != "" {
		query.Set("order_by", opts.OrderBy)
	}

	res, err := c.do(ctx, http.MethodGet, "/api/devices", query, nil)
	if err != nil {
		return nil, 0, err
	}

	var devices []Device

	if err := res.decode(&devices); err != nil {
		return nil, 0, err
	}

	return devices, res.totalCount(), nil
}
