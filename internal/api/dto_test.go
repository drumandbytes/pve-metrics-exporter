package api

import (
	"testing"

	"github.com/drumandbytes/pve-metrics-exporter/internal/config"
	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
)

func TestToTemperature_CriticalPercentIsUnitIndependent(t *testing.T) {
	// 50°C of 100°C critical = 50%, always - regardless of which unit
	// Value/Critical are displayed in. Computing the percent from
	// Fahrenheit-converted numbers (122°F / 212°F) would wrongly give
	// ~57.5%, since Celsius and Fahrenheit don't share a zero point.
	r := proxmox.Reading{Kind: proxmox.KindCPU, Value: 50.0, Critical: 100.0, HasCritical: true}

	celsius := toTemperature(r, config.Celsius)
	fahrenheit := toTemperature(r, config.Fahrenheit)

	if celsius.CriticalPercent == nil || *celsius.CriticalPercent != 50.0 {
		t.Errorf("celsius CriticalPercent = %v, want 50.0", celsius.CriticalPercent)
	}
	if fahrenheit.CriticalPercent == nil || *fahrenheit.CriticalPercent != 50.0 {
		t.Errorf("fahrenheit CriticalPercent = %v, want 50.0 (must match celsius)", fahrenheit.CriticalPercent)
	}

	// Display values (Value/Critical themselves) should still convert.
	if fahrenheit.Value != 122.0 {
		t.Errorf("fahrenheit Value = %v, want 122.0", fahrenheit.Value)
	}
	if *fahrenheit.Critical != 212.0 {
		t.Errorf("fahrenheit Critical = %v, want 212.0", *fahrenheit.Critical)
	}
}

func TestToTemperature_NoCritical(t *testing.T) {
	r := proxmox.Reading{Kind: proxmox.KindACPI, Value: 30.0, HasCritical: false}
	got := toTemperature(r, config.Celsius)
	if got.Critical != nil || got.CriticalPercent != nil {
		t.Errorf("expected nil Critical/CriticalPercent, got %v / %v", got.Critical, got.CriticalPercent)
	}
}
