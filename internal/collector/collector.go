// Package collector adapts a summary.Fetcher into a prometheus.Collector,
// so metrics are computed fresh from the shared cache on every scrape
// rather than accumulated/pushed.
package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/drumandbytes/pve-metrics-exporter/internal/proxmox"
	"github.com/drumandbytes/pve-metrics-exporter/internal/summary"
)

const namespace = "pve"

type Collector struct {
	fetcher *summary.Fetcher

	up            *prometheus.Desc
	nodeCPU       *prometheus.Desc
	nodeMemUsed   *prometheus.Desc
	nodeMemTotal  *prometheus.Desc
	nodeDiskUsed  *prometheus.Desc
	nodeDiskTotal *prometheus.Desc
	nodeTempC     *prometheus.Desc
	nodeTempF     *prometheus.Desc
	nodeTempCritC *prometheus.Desc
	nodeTempCritF *prometheus.Desc
	guestUp       *prometheus.Desc
	guestCPU      *prometheus.Desc
	guestMemUsed  *prometheus.Desc
	guestMemTotal *prometheus.Desc
	storageUsed   *prometheus.Desc
	storageTotal  *prometheus.Desc
}

// New builds the collector. cacheTTL is only used to document the
// caching behavior in each metric's HELP text, per Prometheus's own
// guidance ("if a metric is particularly expensive to retrieve... it
// is acceptable to cache it. This should be noted in the HELP
// string.") - it doesn't change the actual caching (that's
// summary.Fetcher's job, shared with the JSON API).
func New(fetcher *summary.Fetcher, cacheTTL time.Duration) *Collector {
	cacheNote := fmt.Sprintf(" Cached for up to %s to limit load on the Proxmox API.", cacheTTL)
	desc := func(subsystem, name, help string, labels []string) *prometheus.Desc {
		return prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, name), help+cacheNote, labels, nil)
	}

	return &Collector{
		fetcher: fetcher,
		up: desc("", "up",
			"Whether the last scrape of the Proxmox API succeeded (1) or a stale cache is being served (0).", nil),
		nodeCPU: desc("node", "cpu_percent",
			"Node CPU usage in percent.", []string{"node"}),
		nodeMemUsed: desc("node", "memory_used_bytes",
			"Node memory in use, in bytes.", []string{"node"}),
		nodeMemTotal: desc("node", "memory_total_bytes",
			"Node total memory, in bytes.", []string{"node"}),
		nodeDiskUsed: desc("node", "disk_used_bytes",
			"Node root filesystem usage, in bytes.", []string{"node"}),
		nodeDiskTotal: desc("node", "disk_total_bytes",
			"Node root filesystem size, in bytes.", []string{"node"}),
		// Both unit variants are always emitted, unconditionally -
		// not gated by an env var. The unit lives in the metric name
		// per Prometheus convention (see node_exporter's
		// node_hwmon_temp_celsius), so toggling it via config would
		// mean either lying about the name or renaming the metric out
		// from under anyone's saved dashboard/alert. Emitting both
		// side by side avoids that entirely: nothing to break, pick
		// whichever in your query.
		nodeTempC: desc("node", "temperature_celsius",
			"Hardware sensor temperature reading, in Celsius.", []string{"node", "kind", "chip", "label"}),
		nodeTempF: desc("node", "temperature_fahrenheit",
			"Hardware sensor temperature reading, in Fahrenheit.", []string{"node", "kind", "chip", "label"}),
		// Only emitted for readings whose chip actually reports a
		// crit/max threshold (see proxmox.Reading.HasCritical) - some
		// don't, e.g. an ACPI thermal zone typically has neither.
		// Missing series for those label combinations is normal
		// Prometheus behavior, not a bug.
		nodeTempCritC: desc("node", "temperature_critical_celsius",
			"Sensor's own critical/max threshold, in Celsius. Compare against temperature_celsius to gauge how close a reading is to its limit.", []string{"node", "kind", "chip", "label"}),
		nodeTempCritF: desc("node", "temperature_critical_fahrenheit",
			"Sensor's own critical/max threshold, in Fahrenheit. Compare against temperature_fahrenheit to gauge how close a reading is to its limit.", []string{"node", "kind", "chip", "label"}),
		guestUp: desc("guest", "up",
			"Whether the VM/LXC is running (1) or not (0).", []string{"type", "node", "name", "vmid"}),
		guestCPU: desc("guest", "cpu_percent",
			"Guest CPU usage in percent.", []string{"type", "node", "name", "vmid"}),
		guestMemUsed: desc("guest", "memory_used_bytes",
			"Guest memory in use, in bytes.", []string{"type", "node", "name", "vmid"}),
		guestMemTotal: desc("guest", "memory_total_bytes",
			"Guest total memory, in bytes.", []string{"type", "node", "name", "vmid"}),
		storageUsed: desc("storage", "used_bytes",
			"Storage pool usage, in bytes.", []string{"node", "storage", "plugin"}),
		storageTotal: desc("storage", "total_bytes",
			"Storage pool size, in bytes.", []string{"node", "storage", "plugin"}),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	s, err := c.fetcher.Get(context.Background())
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

	for _, n := range s.Nodes {
		ch <- prometheus.MustNewConstMetric(c.nodeCPU, prometheus.GaugeValue, n.CPUPercent, n.Name)
		ch <- prometheus.MustNewConstMetric(c.nodeMemUsed, prometheus.GaugeValue, n.MemUsedBytes, n.Name)
		ch <- prometheus.MustNewConstMetric(c.nodeMemTotal, prometheus.GaugeValue, n.MemTotalBytes, n.Name)
		ch <- prometheus.MustNewConstMetric(c.nodeDiskUsed, prometheus.GaugeValue, n.DiskUsedBytes, n.Name)
		ch <- prometheus.MustNewConstMetric(c.nodeDiskTotal, prometheus.GaugeValue, n.DiskTotalBytes, n.Name)
		emitTemps(ch, c.nodeTempC, c.nodeTempF, c.nodeTempCritC, c.nodeTempCritF, n.Name, n.Temperatures)
	}

	emitGuests(ch, c, s.VMs)
	emitGuests(ch, c, s.LXCs)

	for _, st := range s.Storages {
		ch <- prometheus.MustNewConstMetric(c.storageUsed, prometheus.GaugeValue, st.UsedBytes, st.Node, st.Name, st.PluginType)
		ch <- prometheus.MustNewConstMetric(c.storageTotal, prometheus.GaugeValue, st.TotalBytes, st.Node, st.Name, st.PluginType)
	}
}

func emitTemps(ch chan<- prometheus.Metric, descC, descF, descCritC, descCritF *prometheus.Desc, node string, readings []proxmox.Reading) {
	for _, r := range readings {
		labels := []string{node, string(r.Kind), r.Chip, r.Label}
		ch <- prometheus.MustNewConstMetric(descC, prometheus.GaugeValue, r.Value, labels...)
		ch <- prometheus.MustNewConstMetric(descF, prometheus.GaugeValue, r.Value*9/5+32, labels...)
		if r.HasCritical {
			ch <- prometheus.MustNewConstMetric(descCritC, prometheus.GaugeValue, r.Critical, labels...)
			ch <- prometheus.MustNewConstMetric(descCritF, prometheus.GaugeValue, r.Critical*9/5+32, labels...)
		}
	}
}

func emitGuests(ch chan<- prometheus.Metric, c *Collector, guests []summary.GuestSummary) {
	for _, g := range guests {
		vmid := strconv.Itoa(g.VMID)
		up := 0.0
		if g.Status == "running" {
			up = 1
		}
		ch <- prometheus.MustNewConstMetric(c.guestUp, prometheus.GaugeValue, up, g.Type, g.Node, g.Name, vmid)
		ch <- prometheus.MustNewConstMetric(c.guestCPU, prometheus.GaugeValue, g.CPUPercent, g.Type, g.Node, g.Name, vmid)
		ch <- prometheus.MustNewConstMetric(c.guestMemUsed, prometheus.GaugeValue, g.MemUsedBytes, g.Type, g.Node, g.Name, vmid)
		ch <- prometheus.MustNewConstMetric(c.guestMemTotal, prometheus.GaugeValue, g.MemTotalBytes, g.Type, g.Node, g.Name, vmid)
	}
}
