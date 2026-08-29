// Package config loads settings from environment variables. Env vars
// only (no flags/files) - this is meant to run as a container, where
// env vars are the natural configuration surface.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TemperatureUnit controls unit conversion for the JSON API only.
// /metrics always reports Celsius regardless of this setting - that's
// the Prometheus convention (base units, so dashboards/alerts stay
// consistent no matter how any one exporter instance is configured);
// Grafana can convert to Fahrenheit in a panel if needed. The JSON API
// is meant for direct human display (Glance), so it honors this.
type TemperatureUnit string

const (
	Celsius    TemperatureUnit = "celsius"
	Fahrenheit TemperatureUnit = "fahrenheit"
)

type Config struct {
	ProxmoxURL         string
	ProxmoxAuthHeader  string
	InsecureSkipVerify bool
	ListenAddr         string
	RequestTimeout     time.Duration
	CacheTTL           time.Duration
	CacheMaxStale      time.Duration
	TemperatureUnit    TemperatureUnit
}

func FromEnv() (Config, error) {
	c := Config{
		ProxmoxURL:         os.Getenv("PROXMOX_URL"),
		ProxmoxAuthHeader:  os.Getenv("PROXMOX_TOKEN"),
		InsecureSkipVerify: envBool("PROXMOX_INSECURE_SKIP_VERIFY", true),
		ListenAddr:         envString("LISTEN_ADDR", ":9221"),
		RequestTimeout:     envDuration("PROXMOX_REQUEST_TIMEOUT", 10*time.Second),
		CacheTTL:           envDuration("CACHE_TTL", 15*time.Second),
		CacheMaxStale:      envDuration("CACHE_MAX_STALE", 5*time.Minute),
		TemperatureUnit:    TemperatureUnit(envString("TEMPERATURE_UNIT", string(Celsius))),
	}

	if c.ProxmoxURL == "" {
		return Config{}, fmt.Errorf("PROXMOX_URL is required")
	}
	if c.ProxmoxAuthHeader == "" {
		return Config{}, fmt.Errorf("PROXMOX_TOKEN is required")
	}
	if c.TemperatureUnit != Celsius && c.TemperatureUnit != Fahrenheit {
		return Config{}, fmt.Errorf("TEMPERATURE_UNIT must be %q or %q, got %q", Celsius, Fahrenheit, c.TemperatureUnit)
	}
	return c, nil
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
