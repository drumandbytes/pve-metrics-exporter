package proxmox

// ClusterResource is one entry from /api2/json/cluster/resources.
// Not every field is populated for every "type" - e.g. storages don't
// have "cpu", VMs/LXCs don't have "storage". Left as pointers where a
// zero value would be ambiguous with "field absent".
type ClusterResource struct {
	Type       string  `json:"type"` // node | qemu | lxc | storage | sdn | pool
	Node       string  `json:"node"`
	Name       string  `json:"name"`
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	CPU        float64 `json:"cpu"`
	MaxCPU     float64 `json:"maxcpu"`
	Mem        float64 `json:"mem"`
	MaxMem     float64 `json:"maxmem"`
	Disk       float64 `json:"disk"`
	MaxDisk    float64 `json:"maxdisk"`
	Storage    string  `json:"storage"`
	PluginType string  `json:"plugintype"`
	VMID       int     `json:"vmid"`
	// Proxmox reports this as 0/1, not a JSON bool - IsTemplate() below
	// is the thing to actually call.
	Template int `json:"template"`
}

// IsTemplate reports whether this qemu/lxc entry is a template rather
// than a real, runnable guest (Proxmox always reports 0% CPU/mem for
// templates since they never run - showing them in a resource-usage
// list is just noise).
func (r ClusterResource) IsTemplate() bool {
	return r.Template != 0
}

type clusterResourcesResponse struct {
	Data []ClusterResource `json:"data"`
}

// NodeStatus is the relevant subset of /api2/json/nodes/{node}/status.
// SensorsOutput is itself a JSON-encoded string (Proxmox embeds the raw
// `sensors -j` output as text rather than a nested object), so it's
// decoded in a second pass - see ParseSensors.
type NodeStatus struct {
	SensorsOutput string `json:"sensorsOutput"`
}

type nodeStatusResponse struct {
	Data NodeStatus `json:"data"`
}
