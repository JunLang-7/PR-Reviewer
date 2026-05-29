package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v69/github"
)

type Client struct {
	appID      int64
	privateKey []byte
}

func NewClient(appID int64, privateKey string) *Client {
	return &Client{
		appID:      appID,
		privateKey: []byte(privateKey),
	}
}

type loggingTransport struct {
	base http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("[github] %s %s", req.Method, req.URL.String())
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		log.Printf("[github] error: %v", err)
		return nil, err
	}
	log.Printf("[github] %s %s → %d (rate: %s/%s)",
		req.Method, req.URL.Path, resp.StatusCode,
		resp.Header.Get("X-Ratelimit-Remaining"),
		resp.Header.Get("X-Ratelimit-Limit"))
	return resp, nil
}

func (c *Client) NewInstallationClient(ctx context.Context, installationID int64) (*github.Client, error) {
	tr, err := ghinstallation.New(&loggingTransport{base: http.DefaultTransport}, c.appID, installationID, c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create installation transport: %w", err)
	}
	return github.NewClient(&http.Client{Transport: tr, Timeout: 30 * time.Second}), nil
}
