// Package proxmox is a minimal client for PVE cluster resources and node sensors.
package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	authHeader string // full "Authorization" header value, e.g. "PVEAPIToken=user@pve!id=secret"
	httpClient *http.Client
}

// NewClient builds a client. insecureSkipVerify is for PVE's default self-signed cert.
func NewClient(baseURL, authHeader string, insecureSkipVerify bool, timeout time.Duration) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify}, //nolint:gosec // opt-in via config
	}
	return &Client{
		baseURL:    baseURL,
		authHeader: authHeader,
		httpClient: &http.Client{Transport: transport, Timeout: timeout},
	}
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response from %s: %w", path, err)
	}
	return nil
}

// ClusterResources returns every node/VM/LXC/storage entry the token can see.
func (c *Client) ClusterResources(ctx context.Context) ([]ClusterResource, error) {
	var out clusterResourcesResponse
	if err := c.get(ctx, "/api2/json/cluster/resources", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// NodeStatus returns one node's hardware status; errors for offline nodes.
func (c *Client) NodeStatus(ctx context.Context, node string) (NodeStatus, error) {
	var out nodeStatusResponse
	if err := c.get(ctx, "/api2/json/nodes/"+node+"/status", &out); err != nil {
		return NodeStatus{}, err
	}
	return out.Data, nil
}
