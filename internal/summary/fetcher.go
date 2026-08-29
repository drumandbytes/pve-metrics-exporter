package summary

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
)

// Fetcher shares one cached Summary between the JSON API and the
// Prometheus collector, so a scrape and a Glance poll landing close
// together don't each trigger their own round trip to Proxmox.
//
// On a refresh failure, the previous successful Summary keeps being
// served (up to maxStale) rather than surfacing an error immediately -
// a brief Proxmox API hiccup shouldn't flap Glance's widget or a
// Prometheus scrape.
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
