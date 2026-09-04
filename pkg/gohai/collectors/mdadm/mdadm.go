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
	"slices"
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
		// An MD device line reads "md0 : active raid1 sda1[0] sdb1[1]".
		devName, rest, ok := strings.Cut(scanner.Text(), " : ")
		if !ok {
			continue
		}

		devName = strings.TrimSpace(devName)
		if !strings.HasPrefix(devName, "md") {
			continue
		}

		devices[devName] = mdstatArray(devName, strings.Fields(rest))
	}

	return devices
}

// mdstatArray reads one device's members. The tokens after the state and
// raid level name the members, each suffixed with its role.
func mdstatArray(
	devName string,
	tokens []string,
) *Array {
	arr := &Array{
		Device:  devName,
		Members: []string{},
		Spares:  []string{},
	}

	for _, token := range tokens {
		m := memberRE.FindStringSubmatch(token)
		if m == nil {
			continue
		}

		// "S" marks a spare, "J" a journal, empty an active member.
		if m[2] == "S" {
			arr.Spares = append(arr.Spares, m[1])

			continue
		}

		arr.Members = append(arr.Members, m[1])
	}

	return arr
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

		applyDetailStrings(arr, line)
		applyDetailCounts(arr, line)
	}
}

// applyDetailStrings records the fields mdadm --detail reports as text.
func applyDetailStrings(
	arr *Array,
	line string,
) {
	for _, f := range []struct {
		re  *regexp.Regexp
		dst *string
	}{
		{raidLevelRE, &arr.Level},
		{arrayStateRE, &arr.State},
		{uuidRE, &arr.UUID},
	} {
		if m := f.re.FindStringSubmatch(line); m != nil {
			*f.dst = m[1]
		}
	}
}

// applyDetailCounts records the disk counts, leaving a count that will
// not parse at zero.
func applyDetailCounts(
	arr *Array,
	line string,
) {
	for _, f := range []struct {
		re  *regexp.Regexp
		dst *int
	}{
		{activeDevicesRE, &arr.ActiveDisks},
		{totalDevicesRE, &arr.TotalDisks},
		{spareDevicesRE, &arr.SpareDisks},
	} {
		m := f.re.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		if v, err := strconv.Atoi(m[1]); err == nil {
			*f.dst = v
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
		// A host with no MD support has no mdstat, which is not a failure.
		if isNotExist(err) {
			return []Array{}, nil
		}

		return nil, fmt.Errorf("read /proc/mdstat: %w", err)
	}

	devices := parseMdstat(b)
	if len(devices) == 0 {
		return []Array{}, nil
	}

	// Sorted, so the output does not depend on map order.
	names := make([]string, 0, len(devices))
	for k := range devices {
		names = append(names, k)
	}

	slices.Sort(names)

	arrays := make([]Array, 0, len(names))

	for _, name := range names {
		arr := devices[name]
		enrichFromDetail(ctx, execFn, name, arr)
		arrays = append(arrays, *arr)
	}

	return arrays, nil
}

// isNotExist recognises a missing file across the filesystem
// implementations this reads through, which report it differently.
func isNotExist(
	err error,
) bool {
	msg := err.Error()

	return strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "file does not exist")
}

// enrichFromDetail adds what mdadm --detail knows. It is best-effort:
// mdstat alone already describes the array.
func enrichFromDetail(
	ctx context.Context,
	execFn func(context.Context, string, ...string) ([]byte, error),
	name string,
	arr *Array,
) {
	if execFn == nil {
		return
	}

	detail, err := execFn(ctx, "mdadm", "--detail", "/dev/"+name)
	if err != nil {
		return
	}

	applyDetail(arr, detail)
}
