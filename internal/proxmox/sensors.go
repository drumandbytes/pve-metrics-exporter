package proxmox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Proxmox embeds raw `sensors -j` output as a JSON string in
// NodeStatus.SensorsOutput, so it takes a second Unmarshal. Decoded:
//
//	{
//	  "<chip>-<bus>-<addr>": {
//	    "Adapter": "...",
//	    "<label>": { "<field>_input": 53.0, "<field>_max": 82.0, ... },
//	    ...
//	  }
//	}
//
// <chip> and <label> vary by vendor (coretemp on Intel, k10temp/zenpower on AMD).
type rawSensorTree map[string]map[string]json.RawMessage

var inputFieldRe = regexp.MustCompile(`^(temp|fan|in|curr|power)(\d*)_input$`)

// Kind classifies a chip by lm-sensors name prefix. Unknown chips report as
// "other" rather than being dropped.
type Kind string

const (
	KindCPU     Kind = "cpu"
	KindGPU     Kind = "gpu"
	KindNVMe    Kind = "nvme"
	KindDrive   Kind = "drive"
	KindChipset Kind = "chipset"
	KindACPI    Kind = "acpi"
	KindOther   Kind = "other"
)

func classify(chip string) Kind {
	switch {
	case strings.HasPrefix(chip, "coretemp-"), strings.HasPrefix(chip, "k10temp-"), strings.HasPrefix(chip, "zenpower-"):
		return KindCPU
	case strings.HasPrefix(chip, "nouveau-"), strings.HasPrefix(chip, "amdgpu-"), strings.HasPrefix(chip, "nvidia-"):
		return KindGPU
	case strings.HasPrefix(chip, "nvme-"):
		return KindNVMe
	case strings.HasPrefix(chip, "drivetemp-"):
		return KindDrive
	case strings.HasPrefix(chip, "pch_"):
		return KindChipset
	case strings.HasPrefix(chip, "acpitz-"):
		return KindACPI
	default:
		return KindOther
	}
}

// Reading is one value, e.g. coretemp-isa-0000 / "Package id 0" / temp1_input -> 53.0.
type Reading struct {
	Chip    string
	Adapter string
	Label   string
	Field   string // metric kind derived from the field name: temp, fan, in, curr, power
	Value   float64
	Kind    Kind

	// Critical is "_crit", else "_max", when the chip reports one (ACPI zones often don't).
	Critical    float64
	HasCritical bool
}

// sane bounds for "_crit"/"_max"; some NVMe firmwares report sentinels like
// 65261.85 for unimplemented thresholds.
const (
	minSaneCritical = 40.0
	maxSaneCritical = 150.0
)

func findCritical(fields map[string]float64, base string) (float64, bool) {
	for _, suffix := range []string{"_crit", "_max"} {
		if v, ok := fields[base+suffix]; ok && v > minSaneCritical && v < maxSaneCritical {
			return v, true
		}
	}
	return 0, false
}

// ParseSensors flattens the payload to one Reading per "*_input" field, with
// its sibling "_crit"/"_max" as Critical. "_hyst" is ignored.
func ParseSensors(raw string) ([]Reading, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var tree rawSensorTree
	if err := json.Unmarshal([]byte(raw), &tree); err != nil {
		return nil, fmt.Errorf("decoding sensorsOutput: %w", err)
	}

	var readings []Reading
	for chip, labels := range tree {
		var adapter string
		if adapterRaw, ok := labels["Adapter"]; ok {
			_ = json.Unmarshal(adapterRaw, &adapter)
		}
		kind := classify(chip)

		for label, fieldsRaw := range labels {
			if label == "Adapter" {
				continue
			}
			var fields map[string]float64
			if err := json.Unmarshal(fieldsRaw, &fields); err != nil {
				// some labels (e.g. "pwm1": {}) have no numeric fields; skip, don't fail the node
				continue
			}
			for field, value := range fields {
				m := inputFieldRe.FindStringSubmatch(field)
				if m == nil {
					continue
				}
				base := m[1] + m[2] // e.g. "temp1", matching the sibling "temp1_crit"/"temp1_max" keys
				critical, hasCritical := findCritical(fields, base)
				readings = append(readings, Reading{
					Chip:        chip,
					Adapter:     adapter,
					Label:       label,
					Field:       m[1], // temp | fan | in | curr | power
					Value:       value,
					Kind:        kind,
					Critical:    critical,
					HasCritical: hasCritical,
				})
			}
		}
	}
	return readings, nil
}

// Temperatures filters readings to "temp*_input" (Celsius).
func Temperatures(readings []Reading) []Reading {
	var out []Reading
	for _, r := range readings {
		if r.Field == "temp" {
			out = append(out, r)
		}
	}
	return out
}
