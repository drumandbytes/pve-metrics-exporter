package proxmox

import "testing"

// Fixture captured from a real Proxmox host's sensorsOutput during
// development (see git history) - includes the real cases that make
// this parsing non-trivial: a bogus NVMe secondary-sensor threshold
// sentinel (65261.85, not a real temperature limit) and an ACPI zone
// with no crit/max at all.
const realSensorsFixture = `{
  "coretemp-isa-0000": {
    "Adapter": "ISA adapter",
    "Package id 0": {"temp1_input": 50.0, "temp1_max": 82.0, "temp1_crit": 100.0, "temp1_crit_alarm": 0.0},
    "Core 0": {"temp2_input": 49.0, "temp2_max": 82.0, "temp2_crit": 100.0}
  },
  "nouveau-pci-0100": {
    "Adapter": "PCI adapter",
    "temp1": {"temp1_input": 55.0, "temp1_max": 95.0, "temp1_crit": 105.0, "temp1_emergency": 135.0}
  },
  "acpitz-acpi-0": {
    "Adapter": "ACPI interface",
    "temp1": {"temp1_input": 30.0}
  },
  "nvme-pci-0200": {
    "Adapter": "PCI adapter",
    "Composite": {"temp1_input": 48.85, "temp1_max": 80.85, "temp1_crit": 81.85},
    "Sensor 1": {"temp2_input": 48.85, "temp2_max": 65261.85}
  }
}`

func TestParseSensors_CriticalThresholds(t *testing.T) {
	readings, err := ParseSensors(realSensorsFixture)
	if err != nil {
		t.Fatalf("ParseSensors: %v", err)
	}

	byLabel := map[string]Reading{}
	for _, r := range readings {
		byLabel[r.Chip+"/"+r.Label] = r
	}

	cases := []struct {
		key         string
		wantValue   float64
		wantHasCrit bool
		wantCrit    float64
	}{
		{"coretemp-isa-0000/Package id 0", 50.0, true, 100.0},
		{"nouveau-pci-0100/temp1", 55.0, true, 105.0},
		{"acpitz-acpi-0/temp1", 30.0, false, 0}, // no crit/max reported at all
		{"nvme-pci-0200/Composite", 48.85, true, 81.85},
		// The 65261.85 "_max" sentinel some NVMe firmwares report for
		// an unimplemented threshold must be rejected, not surfaced as
		// a real critical value.
		{"nvme-pci-0200/Sensor 1", 48.85, false, 0},
	}

	for _, c := range cases {
		r, ok := byLabel[c.key]
		if !ok {
			t.Errorf("%s: reading not found", c.key)
			continue
		}
		if r.Value != c.wantValue {
			t.Errorf("%s: Value = %v, want %v", c.key, r.Value, c.wantValue)
		}
		if r.HasCritical != c.wantHasCrit {
			t.Errorf("%s: HasCritical = %v, want %v", c.key, r.HasCritical, c.wantHasCrit)
		}
		if c.wantHasCrit && r.Critical != c.wantCrit {
			t.Errorf("%s: Critical = %v, want %v", c.key, r.Critical, c.wantCrit)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]Kind{
		"coretemp-isa-0000":        KindCPU,
		"k10temp-pci-00c3":         KindCPU,
		"nouveau-pci-0100":         KindGPU,
		"amdgpu-pci-0300":          KindGPU,
		"nvme-pci-0200":            KindNVMe,
		"drivetemp-scsi-0":         KindDrive,
		"pch_cannonlake-virtual-0": KindChipset,
		"acpitz-acpi-0":            KindACPI,
		"whatever-isa-0000":        KindOther,
	}
	for chip, want := range cases {
		if got := classify(chip); got != want {
			t.Errorf("classify(%q) = %v, want %v", chip, got, want)
		}
	}
}
