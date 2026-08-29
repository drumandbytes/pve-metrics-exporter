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

// temperature's Value field is deliberately not called "celsius" -
// its unit depends on the configured config.TemperatureUnit, and a
// stale/misleading field name would be worse than a generic one. Check
// the response's top-level "temperature_unit" field to interpret it.
type temperature struct {
	Kind  string  `json:"kind"`
	Chip  string  `json:"chip"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
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
	// Unit follows the response's top-level "temperature_unit".
	CPUTemp  *float64 `json:"cpu_temp,omitempty"`
	GPUTemp  *float64 `json:"gpu_temp,omitempty"`
	NVMeTemp *float64 `json:"nvme_temp,omitempty"`
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
			dtoNode.Temperatures = append(dtoNode.Temperatures, temperature{
				Kind: string(t.Kind), Chip: t.Chip, Label: t.Label, Value: convertTemp(t.Value, unit),
			})
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

// pickTemp returns the first reading of the given kind whose label
// matches one of preferredLabels (substring match, e.g. "Package"
// matches "Package id 0"). With no preferredLabels given, it returns
// the first reading of that kind at all (fine for GPUs, which
// typically report a single "temp1" reading).
func pickTemp(readings []proxmox.Reading, unit config.TemperatureUnit, kind proxmox.Kind, preferredLabels ...string) *float64 {
	var fallback *float64
	for _, r := range readings {
		if r.Kind != kind {
			continue
		}
		if fallback == nil {
			v := convertTemp(r.Value, unit)
			fallback = &v
		}
		for _, want := range preferredLabels {
			if strings.Contains(r.Label, want) {
				v := convertTemp(r.Value, unit)
				return &v
			}
		}
	}
	return fallback
}
