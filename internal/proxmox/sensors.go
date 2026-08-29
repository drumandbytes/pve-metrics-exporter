package proxmox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Proxmox embeds the raw `sensors -j` (lm-sensors) output as a
// JSON-encoded string inside NodeStatus.SensorsOutput, rather than as
// a nested object - so it needs a second json.Unmarshal pass. Its
// shape, once decoded, is:
//
//	{
//	  "<chip>-<bus>-<addr>": {
//	    "Adapter": "...",
//	    "<label>": { "<field>_input": 53.0, "<field>_max": 82.0, ... },
//	    ...
//	  }
//	}
//
// "<chip>" and "<label>" vary by hardware/vendor (e.g. "coretemp" on
// Intel vs "k10temp"/"zenpower" on AMD for CPU package temps).
type rawSensorTree map[string]map[string]json.RawMessage

var inputFieldRe = regexp.MustCompile(`^(temp|fan|in|curr|power)(\d*)_input$`)

// Kind classifies a sensor chip into a broad hardware category, based
// on the chip-name prefixes lm-sensors uses across common drivers.
// Unrecognized chips are still reported (Kind "other") rather than
// dropped, since new hardware/drivers show up constantly and a strict
// allowlist would just silently hide readings.
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

// Reading is a single numeric value from one sensor chip/label/field,
// e.g. chip "coretemp-isa-0000", label "Package id 0", field
// "temp1_input" -> 53.0.
type Reading struct {
	Chip    string
	Adapter string
	Label   string
	Field   string // metric kind derived from the field name: temp, fan, in, curr, power
	Value   float64
	Kind    Kind
}

// ParseSensors decodes the doubly-JSON-encoded sensors payload into a
// flat list of readings. Only "*_input" fields are kept (the *_max/
// *_crit/*_hyst thresholds alongside them aren't currently exposed).
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
				// Some labels (e.g. "pwm1": {}) have no numeric
				// sub-fields, or fields aren't all numeric - skip
				// rather than fail the whole node's readings.
				continue
			}
			for field, value := range fields {
				m := inputFieldRe.FindStringSubmatch(field)
				if m == nil {
					continue
				}
				readings = append(readings, Reading{
					Chip:    chip,
					Adapter: adapter,
					Label:   label,
					Field:   m[1], // temp | fan | in | curr | power
					Value:   value,
					Kind:    kind,
				})
			}
		}
	}
	return readings, nil
}

// Temperatures filters a reading list down to temperature-only
// readings ("temp*_input" fields, reported by lm-sensors in Celsius).
func Temperatures(readings []Reading) []Reading {
	var out []Reading
	for _, r := range readings {
		if r.Field == "temp" {
			out = append(out, r)
		}
	}
	return out
}
