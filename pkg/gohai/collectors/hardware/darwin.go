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

package hardware

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin runs three system_profiler calls concurrently and merges
// their outputs into a typed Info. Same data sources as Ohai's
// darwin/hardware.rb: SPHardwareDataType (identity + memory +
// firmware), SPStorageDataType (attached volumes), and
// SPPowerDataType (battery + AC charger).
type Darwin struct {
	base
	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect runs the three system_profiler calls in parallel and
// populates Info. Each call's failure is tolerated silently — a
// host without a battery returns nil Battery; a host whose storage
// query fails returns no Storage; etc.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	if d.Exec == nil {
		return info, nil
	}

	var wg sync.WaitGroup
	var hwOut, storageOut, powerOut []byte

	wg.Add(3)
	go func() {
		defer wg.Done()
		hwOut, _ = d.Exec.Execute(ctx, "system_profiler", "SPHardwareDataType", "-json")
	}()
	go func() {
		defer wg.Done()
		storageOut, _ = d.Exec.Execute(ctx, "system_profiler", "SPStorageDataType", "-json")
	}()
	go func() {
		defer wg.Done()
		powerOut, _ = d.Exec.Execute(ctx, "system_profiler", "SPPowerDataType", "-json")
	}()
	wg.Wait()

	applyHardware(info, hwOut)
	applyStorage(info, storageOut)
	applyPower(info, powerOut)
	return info, nil
}

// applyHardware parses SPHardwareDataType JSON and populates
// identity / CPU / firmware fields on info.
func applyHardware(
	info *Info,
	out []byte,
) {
	var payload struct {
		Items []struct {
			MachineModel          string `json:"machine_model"`
			MachineName           string `json:"machine_name"`
			SerialNumber          string `json:"serial_number"`
			PlatformUUID          string `json:"platform_UUID"`
			ProvisioningUDID      string `json:"provisioning_UDID"`
			CPUType               string `json:"cpu_type"`
			ChipType              string `json:"chip_type"`
			CurrentProcessorSpeed string `json:"current_processor_speed"`
			NumberProcessors      any    `json:"number_processors"` // int on Intel, string on Apple Silicon
			Packages              any    `json:"packages"`
			L2CacheCore           string `json:"l2_cache_core"`
			L3Cache               string `json:"l3_cache"`
			PhysicalMemory        string `json:"physical_memory"`
			BootROMVersion        string `json:"boot_rom_version"`
			OSLoaderVersion       string `json:"os_loader_version"`
			SMCVersionSystem      string `json:"SMC_version_system"`
		} `json:"SPHardwareDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil || len(payload.Items) == 0 {
		return
	}
	h := payload.Items[0]
	info.MachineModel = h.MachineModel
	info.MachineName = h.MachineName
	info.SerialNumber = h.SerialNumber
	info.PlatformUUID = h.PlatformUUID
	info.ProvisioningUDID = h.ProvisioningUDID
	info.CPUType = h.CPUType
	info.ChipType = h.ChipType
	info.CurrentProcessorSpeed = h.CurrentProcessorSpeed
	info.NumberProcessors = anyToString(h.NumberProcessors)
	info.Packages = anyToInt(h.Packages)
	info.L2CacheCore = h.L2CacheCore
	info.L3Cache = h.L3Cache
	info.PhysicalMemory = h.PhysicalMemory
	info.BootROMVersion = h.BootROMVersion
	info.OSLoaderVersion = h.OSLoaderVersion
	info.SMCVersionSystem = h.SMCVersionSystem
}

// applyStorage parses SPStorageDataType and fills Info.Storage.
func applyStorage(
	info *Info,
	out []byte,
) {
	var payload struct {
		Items []struct {
			Name        string `json:"_name"`
			BSDName     string `json:"bsd_name"`
			SizeInBytes int64  `json:"size_in_bytes"`
			FreeInBytes int64  `json:"free_space_in_bytes"`
			FileSystem  string `json:"file_system"`
			MountPoint  string `json:"mount_point"`
			Writable    string `json:"writable"` // "yes"/"no"
			CoreStorage []struct {
				MediumType  string `json:"medium_type"`
				SmartStatus string `json:"smart_status"`
			} `json:"com.apple.corestorage.pv"`
		} `json:"SPStorageDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return
	}
	for _, item := range payload.Items {
		entry := Storage{
			Name:       item.Name,
			BSDName:    item.BSDName,
			Capacity:   item.SizeInBytes,
			FreeSpace:  item.FreeInBytes,
			FileSystem: item.FileSystem,
			MountPoint: item.MountPoint,
			Writable:   strings.EqualFold(item.Writable, "yes"),
		}
		if len(item.CoreStorage) > 0 {
			entry.DriveType = item.CoreStorage[0].MediumType
			entry.SmartStatus = item.CoreStorage[0].SmartStatus
			entry.Partitions = len(item.CoreStorage)
		}
		info.Storage = append(info.Storage, entry)
	}
}

// applyPower parses SPPowerDataType and fills Info.Battery + Charger.
func applyPower(
	info *Info,
	out []byte,
) {
	var payload struct {
		Items []map[string]any `json:"SPPowerDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return
	}
	for _, item := range payload.Items {
		name, _ := item["_name"].(string)
		switch name {
		case "spbattery_information":
			info.Battery = parseBattery(item)
		case "sppower_ac_charger_information":
			info.Charger = parseCharger(item)
		}
	}
}

func parseBattery(
	item map[string]any,
) *Battery {
	// Charge info is nested under sppower_battery_charge_info;
	// health info under sppower_battery_health_info; model info under
	// sppower_battery_model_info. We access leaf keys directly by
	// searching the item and the three nested objects.
	charge := nestedMap(item, "sppower_battery_charge_info")
	health := nestedMap(item, "sppower_battery_health_info")
	model := nestedMap(item, "sppower_battery_model_info")

	cur := firstInt(item, charge, "sppower_battery_current_capacity")
	maxc := firstInt(item, charge, "sppower_battery_max_capacity")

	b := &Battery{
		CurrentCapacity:  cur,
		MaxCapacity:      maxc,
		FullyCharged:     firstBool(item, charge, "sppower_battery_fully_charged"),
		IsCharging:       firstBool(item, charge, "sppower_battery_is_charging"),
		ChargeCycleCount: firstInt(item, health, "sppower_battery_cycle_count"),
		Health:           firstString(item, health, "sppower_battery_health"),
		Serial:           firstString(item, model, "sppower_battery_serial_number"),
		Amperage:         firstInt(item, nil, "sppower_current_amperage"),
		Voltage:          firstInt(item, nil, "sppower_current_voltage"),
	}
	if maxc > 0 {
		b.Remaining = cur * 100 / maxc
	}
	return b
}

func parseCharger(
	item map[string]any,
) *Charger {
	return &Charger{
		ID:           firstString(item, nil, "sppower_ac_charger_ID"),
		Family:       firstString(item, nil, "sppower_ac_charger_family"),
		Revision:     firstString(item, nil, "sppower_ac_charger_revision"),
		SerialNumber: firstString(item, nil, "sppower_ac_charger_serial_number"),
		Watts:        firstString(item, nil, "sppower_ac_charger_watts"),
		Connected:    firstBool(item, nil, "sppower_battery_charger_connected"),
	}
}

// nestedMap returns the map at key, or nil when the key is absent /
// not a map.
func nestedMap(
	m map[string]any,
	key string,
) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

// firstString returns the first non-empty string found at key, checking
// `primary` first then `secondary`. system_profiler's JSON output
// occasionally places a field directly under the item and occasionally
// under a nested sub-object; this tries both.
func firstString(
	primary, secondary map[string]any,
	key string,
) string {
	if primary != nil {
		if s, ok := primary[key].(string); ok && s != "" {
			return s
		}
	}
	if secondary != nil {
		if s, ok := secondary[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// firstInt returns the first integer-like value at key. JSON numbers
// come back as float64; system_profiler also uses "TRUE"/"FALSE"
// strings for booleans (handled by firstBool, not here).
func firstInt(
	primary, secondary map[string]any,
	key string,
) int {
	for _, m := range []map[string]any{primary, secondary} {
		if m == nil {
			continue
		}
		switch v := m[key].(type) {
		case float64:
			return int(v)
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	return 0
}

// firstBool normalizes system_profiler's string-boolean convention
// ("TRUE" / "FALSE") into a real Go bool. Native bools in the JSON
// (rare) are passed through.
func firstBool(
	primary, secondary map[string]any,
	key string,
) bool {
	for _, m := range []map[string]any{primary, secondary} {
		if m == nil {
			continue
		}
		switch v := m[key].(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(v, "TRUE") || strings.EqualFold(v, "yes")
		}
	}
	return false
}

// anyToString flattens system_profiler's sometimes-int-sometimes-string
// number fields into a canonical string representation.
func anyToString(
	v any,
) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.Itoa(int(x))
	}
	return ""
}

// anyToInt does the reverse when we know the value is countable.
func anyToInt(
	v any,
) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		if n, err := strconv.Atoi(x); err == nil {
			return n
		}
	}
	return 0
}
