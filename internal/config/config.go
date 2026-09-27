// Package config reads env vars only; it runs as a container.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TemperatureUnit applies to the JSON API only; /metrics stays Celsius per
// Prometheus convention.
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
