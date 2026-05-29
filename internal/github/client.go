package github

import (
	"context"
	"fmt"
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

func (c *Client) NewInstallationClient(ctx context.Context, installationID int64) (*github.Client, error) {
	tr, err := ghinstallation.New(http.DefaultTransport, c.appID, installationID, c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create installation transport: %w", err)
	}
	return github.NewClient(&http.Client{Transport: tr, Timeout: 30 * time.Second}), nil
}
