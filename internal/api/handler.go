package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/drumandbytes/pve-metrics-exporter/internal/config"
	"github.com/drumandbytes/pve-metrics-exporter/internal/summary"
)

// Handler serves GET /api/summary.
func Handler(fetcher *summary.Fetcher, unit config.TemperatureUnit, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := fetcher.Get(r.Context())
		if err != nil {
			log.Error("fetching summary failed", "error", err)
			http.Error(w, "failed to fetch data from Proxmox", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toDTO(s, unit)); err != nil {
			log.Error("encoding response failed", "error", err)
		}
	}
}
