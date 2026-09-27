package summary

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
)

// Fetcher shares one cached Summary between the JSON API and the collector.
// On refresh failure the last good Summary is served up to maxStale, so a PVE
// hiccup doesn't flap Glance or a scrape.
type Fetcher struct {
	client   *proxmox.Client
	ttl      time.Duration
	maxStale time.Duration
	log      *slog.Logger

	mu        sync.Mutex
	cached    Summary
	fetchedAt time.Time
	haveData  bool
}

func NewFetcher(client *proxmox.Client, ttl, maxStale time.Duration, log *slog.Logger) *Fetcher {
	return &Fetcher{client: client, ttl: ttl, maxStale: maxStale, log: log}
}

func (f *Fetcher) Get(ctx context.Context) (Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.haveData && time.Since(f.fetchedAt) < f.ttl {
		return f.cached, nil
	}

	fresh, err := Build(ctx, f.client)
	if err != nil {
		if f.haveData && time.Since(f.fetchedAt) < f.maxStale {
			f.log.Warn("refresh failed, serving stale cached data",
				"error", err, "age", time.Since(f.fetchedAt))
			return f.cached, nil
		}
		return Summary{}, err
	}

	f.cached = fresh
	f.fetchedAt = time.Now()
	f.haveData = true
	return f.cached, nil
}
