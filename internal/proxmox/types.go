package proxmox

// ClusterResource is one /api2/json/cluster/resources entry. Fields vary by
// type; pointers where zero would be ambiguous with absent.
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
	// 0/1, not a bool: use IsTemplate()
	Template int `json:"template"`
}

// IsTemplate reports whether a qemu/lxc entry is a template (always 0% usage, just noise).
func (r ClusterResource) IsTemplate() bool {
	return r.Template != 0
}

type clusterResourcesResponse struct {
	Data []ClusterResource `json:"data"`
}

// NodeStatus is the used subset of /api2/json/nodes/{node}/status.
// SensorsOutput is a JSON string; see ParseSensors.
type NodeStatus struct {
	SensorsOutput string `json:"sensorsOutput"`
}

type nodeStatusResponse struct {
	Data NodeStatus `json:"data"`
}
