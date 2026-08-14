package shellhub

import (
	"context"
	"net/http"
)

type SystemInfo struct {
	Version   string `json:"version"`
	Endpoints struct {
		API string `json:"api"`
		SSH string `json:"ssh"`
	} `json:"endpoints"`
	Setup          bool `json:"setup"`
	Authentication struct {
		Local bool `json:"local"`
		SAML  bool `json:"saml"`
	} `json:"authentication"`
}

func (c *Client) Info(ctx context.Context) (*SystemInfo, error) {
	res, err := c.doAnonymous(ctx, http.MethodGet, "/info", nil, nil)
	if err != nil {
		return nil, err
	}

	var info SystemInfo

	if err := res.decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}
