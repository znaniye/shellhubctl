package shellhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	base  *url.URL
	http  *http.Client
	token string
}

type Option func(*Client)

func New(server string, opts ...Option) (*Client, error) {
	base, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	c := &Client{
		base: base,
		http: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.http = h
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) Server() string {
	return c.base.String()
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r *response) totalCount() int {
	n, err := strconv.Atoi(r.header.Get("X-Total-Count"))
	if err != nil {
		return 0
	}

	return n
}

func (r *response) decode(out any) error {
	if len(r.body) == 0 {
		return nil
	}

	return json.Unmarshal(r.body, out)
}

func (r *response) checkStatus(method, path string) error {
	if r.status >= 200 && r.status < 300 {
		return nil
	}

	return &APIError{Status: r.status, Method: method, Path: path}
}

func (c *Client) resolve(path string) *url.URL {
	return c.base.ResolveReference(&url.URL{Path: path})
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}

	return page
}

func normalizePerPage(perPage int) int {
	if perPage <= 0 {
		return 100
	}

	return perPage
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*response, error) {
	res, err := c.doRequest(ctx, method, path, query, body, true)
	if err != nil {
		return nil, err
	}

	if err := res.checkStatus(method, path); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body io.Reader) (*response, error) {
	return c.doRequest(ctx, method, path, query, body, true)
}

func (c *Client) doAnonymous(ctx context.Context, method, path string, query url.Values, body io.Reader) (*response, error) {
	res, err := c.doRequest(ctx, method, path, query, body, false)
	if err != nil {
		return nil, err
	}

	if err := res.checkStatus(method, path); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, auth bool) (*response, error) {
	u := c.resolve(path)

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("shellhub: %s %s: %w", method, u.Path, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if auth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shellhub: %s %s: %w", method, u.Path, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shellhub: %s %s: %w", method, u.Path, err)
	}

	res := &response{status: resp.StatusCode, header: resp.Header, body: data}

	return res, nil
}
