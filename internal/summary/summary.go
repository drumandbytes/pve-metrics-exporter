// Package summary flattens raw Proxmox API responses (percentages
// computed, sensor data parsed) into shapes that are easy for both
// the JSON API (Glance) and the Prometheus collector to consume
// without re-deriving the same math twice.
package summary

import (
	"context"
	"time"

	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
)

type NodeSummary struct {
	Name           string
	Status         string
	CPUPercent     float64
	MemUsedBytes   float64
	MemTotalBytes  float64
	MemPercent     float64
	DiskUsedBytes  float64
	DiskTotalBytes float64
	DiskPercent    float64
	Temperatures   []proxmox.Reading
}

// GuestSummary covers both QEMU VMs and LXC containers - Proxmox
// reports them with near-identical fields in cluster/resources, and
// callers distinguish via Type.
type GuestSummary struct {
	Type          string // qemu | lxc
	Node          string
	Name          string
	VMID          int
	Status        string
	CPUPercent    float64
	MemUsedBytes  float64
	MemTotalBytes float64
	MemPercent    float64
}

type StorageSummary struct {
	Node       string
	Name       string
	Status     string
	PluginType string
	UsedBytes  float64
	TotalBytes float64
	Percent    float64
}

type Summary struct {
	GeneratedAt time.Time
	Nodes       []NodeSummary
	VMs         []GuestSummary
	LXCs        []GuestSummary
	Storages    []StorageSummary
}

func percent(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total * 100
}

// Build fetches cluster/resources plus per-node sensor data and
// flattens it all into a Summary. Sensor lookups only happen for
// nodes reporting status "online" - Proxmox returns an API error for
// nodes it can't currently reach, and there's nothing useful to show
// for those anyway.
func Build(ctx context.Context, client *proxmox.Client) (Summary, error) {
	resources, err := client.ClusterResources(ctx)
	if err != nil {
		return Summary{}, err
	}

	s := Summary{GeneratedAt: time.Now()}

	for _, r := range resources {
		switch r.Type {
		case "node":
			node := NodeSummary{
				Name:           r.Node,
				Status:         r.Status,
				CPUPercent:     r.CPU * 100,
				MemUsedBytes:   r.Mem,
				MemTotalBytes:  r.MaxMem,
				MemPercent:     percent(r.Mem, r.MaxMem),
				DiskUsedBytes:  r.Disk,
				DiskTotalBytes: r.MaxDisk,
				DiskPercent:    percent(r.Disk, r.MaxDisk),
			}
			if r.Status == "online" {
				if status, err := client.NodeStatus(ctx, r.Node); err == nil {
					if readings, err := proxmox.ParseSensors(status.SensorsOutput); err == nil {
						node.Temperatures = proxmox.Temperatures(readings)
					}
					// Sensor parsing failures are non-fatal - a node
					// with no lm-sensors configured simply reports no
					// temperatures, rather than failing the whole
					// summary.
				}
			}
			s.Nodes = append(s.Nodes, node)

		case "qemu", "lxc":
			if r.IsTemplate() {
				continue
			}
			guest := GuestSummary{
				Type:          r.Type,
				Node:          r.Node,
				Name:          r.Name,
				VMID:          r.VMID,
				Status:        r.Status,
				CPUPercent:    r.CPU * 100,
				MemUsedBytes:  r.Mem,
				MemTotalBytes: r.MaxMem,
				MemPercent:    percent(r.Mem, r.MaxMem),
			}
			if r.Type == "qemu" {
				s.VMs = append(s.VMs, guest)
			} else {
				s.LXCs = append(s.LXCs, guest)
			}

		case "storage":
			s.Storages = append(s.Storages, StorageSummary{
				Node:       r.Node,
				Name:       r.Storage,
				Status:     r.Status,
				PluginType: r.PluginType,
				UsedBytes:  r.Disk,
				TotalBytes: r.MaxDisk,
				Percent:    percent(r.Disk, r.MaxDisk),
			})
		}
	}

	return s, nil
}
