// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package hardware collects macOS-specific hardware facts — the
// identity, memory, and firmware fields surfaced by
// `system_profiler SPHardwareDataType`, attached storage from
// `SPStorageDataType`, and battery / AC charger info from
// `SPPowerDataType`. Mirrors Ohai's `darwin/hardware.rb` methodology.
//
// Non-Darwin platforms return an empty Info — this collector has no
// Linux analogue. Linux hardware identity lives in `dmi`, `cpu`,
// `memory`, and the planned `pci` / `block_device` collectors.
package hardware

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info is the hardware view. Top-level fields mirror the
// `SPHardwareDataType` keys Ohai merges into `node['hardware']`,
// with storage + battery + charger as typed sub-objects.
type Info struct {
	// Identity.
	MachineModel     string `json:"machine_model,omitempty"`
	MachineName      string `json:"machine_name,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
	PlatformUUID     string `json:"platform_uuid,omitempty"`
	ProvisioningUDID string `json:"provisioning_udid,omitempty"`

	// CPU identity strings from system_profiler. Not a replacement for
	// the cpu collector — these carry Apple's marketing labels
	// ("Intel Core i7", "Apple M1 Pro") that aren't in gopsutil.
	CPUType               string `json:"cpu_type,omitempty"`                // Intel Macs
	ChipType              string `json:"chip_type,omitempty"`               // Apple Silicon
	CurrentProcessorSpeed string `json:"current_processor_speed,omitempty"` // Intel only
	NumberProcessors      string `json:"number_processors,omitempty"`
	Packages              int    `json:"packages,omitempty"`
	L2CacheCore           string `json:"l2_cache_core,omitempty"`
	L3Cache               string `json:"l3_cache,omitempty"`

	// Memory (verbatim string from system_profiler, e.g. "16 GB").
	PhysicalMemory string `json:"physical_memory,omitempty"`

	// Firmware.
	BootROMVersion   string `json:"boot_rom_version,omitempty"`
	OSLoaderVersion  string `json:"os_loader_version,omitempty"`
	SMCVersionSystem string `json:"smc_version_system,omitempty"`

	// Attached storage, one entry per logical volume.
	Storage []Storage `json:"storage,omitempty"`

	// Battery details — nil on desktop Macs without a battery.
	Battery *Battery `json:"battery,omitempty"`

	// AC charger — nil when no charger is connected or reported.
	Charger *Charger `json:"charger,omitempty"`
}

// Storage mirrors one entry of `SPStorageDataType`. DriveType,
// SmartStatus, and Partitions are CoreStorage-era signals and stay
// empty on modern APFS volumes.
type Storage struct {
	Name        string `json:"name,omitempty"`
	BSDName     string `json:"bsd_name,omitempty"`
	Capacity    int64  `json:"capacity,omitempty"`
	FileSystem  string `json:"file_system,omitempty"`
	MountPoint  string `json:"mount_point,omitempty"`
	FreeSpace   int64  `json:"free_space,omitempty"`
	Writable    bool   `json:"writable,omitempty"`
	DriveType   string `json:"drive_type,omitempty"`
	SmartStatus string `json:"smart_status,omitempty"`
	Partitions  int    `json:"partitions,omitempty"`
}

// Battery mirrors the `spbattery_information` item of
// `SPPowerDataType`. Remaining is computed as
// `(current / max) * 100` per Ohai.
type Battery struct {
	CurrentCapacity  int    `json:"current_capacity,omitempty"`
	MaxCapacity      int    `json:"max_capacity,omitempty"`
	FullyCharged     bool   `json:"fully_charged,omitempty"`
	IsCharging       bool   `json:"is_charging,omitempty"`
	ChargeCycleCount int    `json:"charge_cycle_count,omitempty"`
	Health           string `json:"health,omitempty"`
	Serial           string `json:"serial,omitempty"`
	Remaining        int    `json:"remaining,omitempty"`
	Amperage         int    `json:"amperage,omitempty"`
	Voltage          int    `json:"voltage,omitempty"`
}

// Charger mirrors the `sppower_ac_charger_information` item of
// `SPPowerDataType`. Ohai skips this data; we surface it because
// fleet-management use cases care about adapter wattage mismatches.
type Charger struct {
	ID           string `json:"id,omitempty"`
	Family       string `json:"family,omitempty"`
	Revision     string `json:"revision,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Watts        string `json:"watts,omitempty"`
	Connected    bool   `json:"connected,omitempty"`
}

// Collector is the public interface every hardware variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "hardware" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the hardware variant for the host OS. Linux returns an
// empty Info — this collector is Darwin-only, matching Ohai.
func New() Collector {
	if platform.IsDarwin() {
		return NewDarwin()
	}
	return NewLinux()
}
