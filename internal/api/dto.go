// Package api renders summary.Summary as JSON for HTTP consumers
// (Glance's custom-api widget, curl, etc). Kept separate from the
// summary package so presentation choices (rounding, convenience
// fields, JSON field names) don't leak into the core data model.
package api

import (
	"strings"
	"time"

	"github.com/drumandbytes/pve-metrics-exporter/internal/config"
	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
	"github.com/drumandbytes/pve-metrics-exporter/internal/summary"
)

// temperature's Value/Critical fields are deliberately not called
// "celsius" - their unit depends on the configured
// config.TemperatureUnit, and a stale/misleading field name would be
// worse than a generic one. Check the response's top-level
// "temperature_unit" field to interpret them.
//
// Critical/CriticalPercent are omitted when the sensor chip doesn't
// report a usable threshold (e.g. an ACPI thermal zone typically has
// no crit/max at all) - a consumer showing this as a progress bar
// should fall back to a plain number in that case.
type temperature struct {
	Kind            string   `json:"kind"`
	Chip            string   `json:"chip"`
	Label           string   `json:"label"`
	Value           float64  `json:"value"`
	Critical        *float64 `json:"critical,omitempty"`
	CriticalPercent *float64 `json:"critical_percent,omitempty"`
}

type node struct {
	Name           string        `json:"name"`
	Status         string        `json:"status"`
	CPUPercent     float64       `json:"cpu_percent"`
	MemUsedBytes   float64       `json:"mem_used_bytes"`
	MemTotalBytes  float64       `json:"mem_total_bytes"`
	MemPercent     float64       `json:"mem_percent"`
	DiskUsedBytes  float64       `json:"disk_used_bytes"`
	DiskTotalBytes float64       `json:"disk_total_bytes"`
	DiskPercent    float64       `json:"disk_percent"`
	Temperatures   []temperature `json:"temperatures"`
	// Best-effort convenience picks out of Temperatures, for the
	// common case of "just show me the CPU/GPU/NVMe temp" without
	// having to filter the list. Omitted (null) when not found -
	// e.g. a node with no lm-sensors configured, or no discrete GPU.
	CPUTemp  *temperature `json:"cpu_temp,omitempty"`
	GPUTemp  *temperature `json:"gpu_temp,omitempty"`
	NVMeTemp *temperature `json:"nvme_temp,omitempty"`
}

type guest struct {
	Type          string  `json:"type"`
	Node          string  `json:"node"`
	Name          string  `json:"name"`
	VMID          int     `json:"vmid"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsedBytes  float64 `json:"mem_used_bytes"`
	MemTotalBytes float64 `json:"mem_total_bytes"`
	MemPercent    float64 `json:"mem_percent"`
}

type storage struct {
	Node       string  `json:"node"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	PluginType string  `json:"plugin_type"`
	UsedBytes  float64 `json:"used_bytes"`
	TotalBytes float64 `json:"total_bytes"`
	Percent    float64 `json:"percent"`
}

type response struct {
	GeneratedAt     time.Time `json:"generated_at"`
	TemperatureUnit string    `json:"temperature_unit"`
	Nodes           []node    `json:"nodes"`
	VMs             []guest   `json:"vms"`
	LXCs            []guest   `json:"lxcs"`
	Storages        []storage `json:"storages"`
}

func convertTemp(celsius float64, unit config.TemperatureUnit) float64 {
	if unit == config.Fahrenheit {
		return celsius*9/5 + 32
	}
	return celsius
}

func toDTO(s summary.Summary, unit config.TemperatureUnit) response {
	out := response{GeneratedAt: s.GeneratedAt, TemperatureUnit: string(unit)}

	for _, n := range s.Nodes {
		dtoNode := node{
			Name:           n.Name,
			Status:         n.Status,
			CPUPercent:     n.CPUPercent,
			MemUsedBytes:   n.MemUsedBytes,
			MemTotalBytes:  n.MemTotalBytes,
			MemPercent:     n.MemPercent,
			DiskUsedBytes:  n.DiskUsedBytes,
			DiskTotalBytes: n.DiskTotalBytes,
			DiskPercent:    n.DiskPercent,
		}
		for _, t := range n.Temperatures {
			dtoNode.Temperatures = append(dtoNode.Temperatures, toTemperature(t, unit))
		}
		dtoNode.CPUTemp = pickTemp(n.Temperatures, unit, proxmox.KindCPU, "Package", "Tctl", "Tdie")
		dtoNode.GPUTemp = pickTemp(n.Temperatures, unit, proxmox.KindGPU)
		dtoNode.NVMeTemp = pickTemp(n.Temperatures, unit, proxmox.KindNVMe, "Composite")
		out.Nodes = append(out.Nodes, dtoNode)
	}

	out.VMs = toGuests(s.VMs)
	out.LXCs = toGuests(s.LXCs)

	for _, st := range s.Storages {
		out.Storages = append(out.Storages, storage{
			Node: st.Node, Name: st.Name, Status: st.Status, PluginType: st.PluginType,
			UsedBytes: st.UsedBytes, TotalBytes: st.TotalBytes, Percent: st.Percent,
		})
	}

	return out
}

func toGuests(guests []summary.GuestSummary) []guest {
	var out []guest
	for _, g := range guests {
		out = append(out, guest{
			Type: g.Type, Node: g.Node, Name: g.Name, VMID: g.VMID, Status: g.Status,
			CPUPercent: g.CPUPercent, MemUsedBytes: g.MemUsedBytes,
			MemTotalBytes: g.MemTotalBytes, MemPercent: g.MemPercent,
		})
	}
	return out
}

// toTemperature converts one proxmox.Reading into its JSON shape.
// CriticalPercent is deliberately computed from the raw Celsius value
// and threshold, never from unit-converted ones: Celsius and
// Fahrenheit have different zero points, so a ratio computed after
// converting to Fahrenheit would not equal the same ratio in Celsius
// (e.g. 0°C/100°C crit = 0%, but the equivalent 32°F/212°F is not 0%).
// Value and Critical are still unit-converted for display.
func toTemperature(r proxmox.Reading, unit config.TemperatureUnit) temperature {
	t := temperature{
		Kind:  string(r.Kind),
		Chip:  r.Chip,
		Label: r.Label,
		Value: convertTemp(r.Value, unit),
	}
	if r.HasCritical {
		critical := convertTemp(r.Critical, unit)
		t.Critical = &critical
		percent := r.Value / r.Critical * 100
		t.CriticalPercent = &percent
	}
	return t
}

// pickTemp returns the first reading of the given kind whose label
// matches one of preferredLabels (substring match, e.g. "Package"
// matches "Package id 0"). With no preferredLabels given, it returns
// the first reading of that kind at all (fine for GPUs, which
// typically report a single "temp1" reading).
func pickTemp(readings []proxmox.Reading, unit config.TemperatureUnit, kind proxmox.Kind, preferredLabels ...string) *temperature {
	var fallback *temperature
	for _, r := range readings {
		if r.Kind != kind {
			continue
		}
		if fallback == nil {
			t := toTemperature(r, unit)
			fallback = &t
		}
		for _, want := range preferredLabels {
			if strings.Contains(r.Label, want) {
				t := toTemperature(r, unit)
				return &t
			}
		}
	}
	return fallback
}
