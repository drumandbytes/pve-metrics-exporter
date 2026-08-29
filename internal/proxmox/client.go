// Package proxmox is a minimal client for the pieces of the Proxmox VE
// API this exporter needs: cluster-wide resource listing and per-node
// hardware sensor readings. It deliberately does not try to be a
// general-purpose PVE API client.
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

// NewClient builds a client. insecureSkipVerify controls whether the
// server's TLS certificate is validated - Proxmox ships a self-signed
// cert by default, so this is commonly needed on homelab setups that
// haven't replaced it with one from a trusted CA.
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response from %s: %w", path, err)
	}
	return nil
}

// ClusterResources returns every node/VM/LXC/storage entry the token
// has visibility into.
func (c *Client) ClusterResources(ctx context.Context) ([]ClusterResource, error) {
	var out clusterResourcesResponse
	if err := c.get(ctx, "/api2/json/cluster/resources", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// NodeStatus returns hardware/sensor status for a single node. Only
// online nodes can be queried - Proxmox returns an error for nodes
// that are offline/unreachable.
func (c *Client) NodeStatus(ctx context.Context, node string) (NodeStatus, error) {
	var out nodeStatusResponse
	if err := c.get(ctx, "/api2/json/nodes/"+node+"/status", &out); err != nil {
		return NodeStatus{}, err
	}
	return out.Data, nil
}
