// Command pve-metrics-exporter exposes a Proxmox VE cluster's resource
// and hardware-sensor data as both a flat JSON endpoint (for Glance's
// custom-api widget) and Prometheus metrics.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/drumandbytes/pve-metrics-exporter/internal/api"
	"github.com/drumandbytes/pve-metrics-exporter/internal/collector"
	"github.com/drumandbytes/pve-metrics-exporter/internal/config"
	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
	"github.com/drumandbytes/pve-metrics-exporter/internal/summary"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.FromEnv()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	client := proxmox.NewClient(cfg.ProxmoxURL, cfg.ProxmoxAuthHeader, cfg.InsecureSkipVerify, cfg.RequestTimeout)
	fetcher := summary.NewFetcher(client, cfg.CacheTTL, cfg.CacheMaxStale, log)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector.New(fetcher, cfg.CacheTTL))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.Handle("/api/summary", api.Handler(fetcher, cfg.TemperatureUnit, log))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Info("listening", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil { //nolint:gosec // homelab-internal, timeouts not critical
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
