# CLAUDE.md

## Project Overview

pve-metrics-exporter is a small Go exporter for Proxmox VE, serving two outputs — `/metrics` (Prometheus, via `client_golang`) and `/api/summary` (flat pre-computed JSON for dashboards like Glance) — from **one shared cache** (`internal/summary.Fetcher`). Both handlers call `Fetcher.Get`, which fetches from the Proxmox API only when the cached `Summary` is older than `CACHE_TTL`; a scrape and a Glance poll landing close together share the same fetch instead of each hitting Proxmox. Do not "simplify" this into two independent fetch paths — that reintroduces duplicate API calls and lets the two outputs drift out of sync with each other. On a refresh failure, the previous result keeps being served up to `CACHE_MAX_STALE` before an error surfaces, so a brief Proxmox hiccup doesn't flap the Glance widget or a Prometheus scrape.

## Architecture

- `internal/proxmox` — API client + parsing of `sensorsOutput` (Proxmox embeds raw `sensors -j`/lm-sensors output as a JSON-encoded *string*; this is the one place that gets unpacked).
- `internal/summary` — `Build` fetches and normalizes everything into one `Summary`; `Fetcher` is the shared cache described above.
- `internal/collector` — adapts `Summary` into Prometheus metrics for `/metrics`.
- `internal/api` — adapts `Summary` into the `/api/summary` JSON DTO.
- `internal/config` — env var parsing (see README's Configuration table for the full var list/defaults).

`TEMPERATURE_UNIT` only affects `/api/summary`; `/metrics` always exposes both `_celsius` and `_fahrenheit` series unconditionally (Prometheus convention bakes units into the metric name, so it can't be toggled without breaking saved panels). Full rationale in README.

See [README.md](README.md) for the config var table, `/api/summary` response shape, and full `/metrics` metric list — don't duplicate it here.

## Build / test / run

```bash
go build ./...
go vet ./...
go test ./...
docker build -t pve-metrics-exporter .
```

CI (`.github/workflows/validate.yml`) gates on the `drumandbytes/reusable-actions` `go-ci.yml` lint job, a Docker smoke test (`/healthz` against fake credentials — never a real Proxmox backend), and Trivy image scan. `build.yml` pushes multi-arch images to GHCR with SLSA provenance attestation on push to `main`/tags.

## Conventions

- Commits: Conventional Commits (`feat:`, `fix:`, `chore:`, …) — `release-please` reads them for versioning/changelog. No `Co-Authored-By` trailers.
- Requires a real Proxmox host + API token to exercise end-to-end (see README's "Proxmox API token" section); there's no mock/fixture server in this repo, so manual testing against `/api/summary` and `/metrics` needs `PROXMOX_URL`/`PROXMOX_TOKEN` pointed at an actual cluster.
