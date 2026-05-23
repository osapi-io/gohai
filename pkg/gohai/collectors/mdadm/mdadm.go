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

// Package mdadm reports Linux software RAID arrays discovered via
// /proc/mdstat and enriched with `mdadm --detail /dev/<device>`.
package mdadm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Array holds the details of a single MD RAID array.
type Array struct {
	// Device is the kernel device name (e.g. "md0", "md127").
	Device string `json:"device"`
	// Level is the RAID level as reported by `mdadm --detail`
	// (e.g. "raid1", "raid5"). Empty when --detail is unavailable.
	Level string `json:"level,omitempty"`
	// State is the array state (e.g. "clean", "active", "degraded").
	State string `json:"state,omitempty"`
	// UUID is the array UUID from `mdadm --detail`.
	UUID string `json:"uuid,omitempty"`
	// ActiveDisks is the count of active member disks.
	ActiveDisks int `json:"active_disks"`
	// TotalDisks is the total number of configured member slots.
	TotalDisks int `json:"total_disks"`
	// SpareDIsks is the count of spare disks in the array.
	SpareDisks int `json:"spare_disks"`
	// Members lists the active member device names (e.g. ["sda1","sdb1"]).
	Members []string `json:"members"`
	// Spares lists the spare member device names.
	Spares []string `json:"spares"`
}

// Info holds the list of MD RAID arrays found on the host.
type Info struct {
	Arrays []Array `json:"arrays"`
}

// Collector is the public interface every mdadm variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "mdadm".
func (base) Name() string { return "mdadm" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — mdadm is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the mdadm collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// memberRE matches mdstat member entries like "sda1[0]", "sdb1[1](S)",
// "sdc1[2](J)" and captures the device name and optional type flag.
var memberRE = regexp.MustCompile(`^(.+)\[\d+\](?:\(([A-Z])\))?$`)

// parseMdstat parses /proc/mdstat content and returns a map of device name
// to partial Array (active members and spares from the mdstat line). The
// returned map is keyed by MD device name.
func parseMdstat(
	content []byte,
) map[string]*Array {
	devices := map[string]*Array{}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		// Lines with MD devices look like:
		//   md0 : active raid1 sda1[0] sdb1[1]
		//   md127 : inactive sdc1[0](S)
		mdIdx := strings.Index(line, " : ")
		if mdIdx < 0 {
			continue
		}
		devName := strings.TrimSpace(line[:mdIdx])
		if !strings.HasPrefix(devName, "md") {
			continue
		}

		rest := strings.Fields(line[mdIdx+3:])
		// rest[0] is "active" or "inactive"; rest[1] (optional) is raid level.
		// Members follow, possibly mixed with raid level token.
		arr := &Array{
			Device:  devName,
			Members: []string{},
			Spares:  []string{},
		}

		for _, token := range rest {
			m := memberRE.FindStringSubmatch(token)
			if m == nil {
				continue
			}
			memberDev := m[1]
			memberType := m[2] // "S" for spare, "J" for journal, "" for active
			switch memberType {
			case "S":
				arr.Spares = append(arr.Spares, memberDev)
			default:
				arr.Members = append(arr.Members, memberDev)
			}
		}

		devices[devName] = arr
	}
	return devices
}

// raidLevelRE matches "Raid Level : raidN" or "RAID Level : raidN".
var raidLevelRE = regexp.MustCompile(`(?i)Raid Level\s*:\s*(\S+)`)

// arrayStateRE matches "State : clean" etc.
var arrayStateRE = regexp.MustCompile(`State\s*:\s*(\S+)`)

// uuidRE matches "UUID : xxxxxxxx:xxxxxxxx:xxxxxxxx:xxxxxxxx".
var uuidRE = regexp.MustCompile(`UUID\s*:\s*(\S+)`)

// activeDevicesRE matches "Active Devices : N".
var activeDevicesRE = regexp.MustCompile(`Active Devices\s*:\s*(\d+)`)

// totalDevicesRE matches "Total Devices : N".
var totalDevicesRE = regexp.MustCompile(`Total Devices\s*:\s*(\d+)`)

// spareDevicesRE matches "Spare Devices : N".
var spareDevicesRE = regexp.MustCompile(`Spare Devices\s*:\s*(\d+)`)

// applyDetail enriches an Array with fields parsed from `mdadm --detail`
// output. Fields not matched in the output are left at their zero values.
func applyDetail(
	arr *Array,
	detail []byte,
) {
	scanner := bufio.NewScanner(bytes.NewReader(detail))
	for scanner.Scan() {
		line := scanner.Text()
		if m := raidLevelRE.FindStringSubmatch(line); m != nil {
			arr.Level = m[1]
		}
		if m := arrayStateRE.FindStringSubmatch(line); m != nil {
			arr.State = m[1]
		}
		if m := uuidRE.FindStringSubmatch(line); m != nil {
			arr.UUID = m[1]
		}
		if m := activeDevicesRE.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				arr.ActiveDisks = v
			}
		}
		if m := totalDevicesRE.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				arr.TotalDisks = v
			}
		}
		if m := spareDevicesRE.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				arr.SpareDisks = v
			}
		}
	}
}

// collectArrays builds the full Array list by parsing /proc/mdstat and
// optionally running mdadm --detail. readFn reads files; execFn runs
// commands. Both may be nil-safe-ish: readFn errors cause an empty
// result, execFn errors on individual arrays are silently skipped
// (mdadm may not be installed).
func collectArrays(
	ctx context.Context,
	readFn func(string) ([]byte, error),
	execFn func(context.Context, string, ...string) ([]byte, error),
) ([]Array, error) {
	b, err := readFn("/proc/mdstat")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "file does not exist") {
			return []Array{}, nil
		}
		return nil, fmt.Errorf("read /proc/mdstat: %w", err)
	}

	devices := parseMdstat(b)
	if len(devices) == 0 {
		return []Array{}, nil
	}

	// Sort device names for deterministic output.
	names := make([]string, 0, len(devices))
	for k := range devices {
		names = append(names, k)
	}
	sort.Strings(names)

	arrays := make([]Array, 0, len(names))
	for _, name := range names {
		arr := devices[name]
		if execFn != nil {
			detail, execErr := execFn(ctx, "mdadm", "--detail", "/dev/"+name)
			if execErr == nil {
				applyDetail(arr, detail)
			}
		}
		arrays = append(arrays, *arr)
	}
	return arrays, nil
}
