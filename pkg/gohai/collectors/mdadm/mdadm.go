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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Array holds the details of a single MD RAID array.
type Array struct {
	// Device is the kernel device name (e.g. "md0", "md127").
	Device string `json:"device"`
	// Level is the RAID level as reported by `mdadm --detail`
	// (e.g. "raid1", "raid5"). Empty when --detail is unavailable.
	Level string `json:"level,omitempty"`
	// State is the array state as `mdadm --detail` prints it: a list
	// separated by ", ", such as "clean", "active, resyncing" or
	// "clean, degraded, recovering".
	State string `json:"state,omitempty"`
	// UUID is the array UUID from `mdadm --detail`.
	UUID string `json:"uuid,omitempty"`
	// ActiveDisks is the count of active member disks.
	ActiveDisks int `json:"active_disks"`
	// TotalDisks is the total number of configured member slots.
	TotalDisks int `json:"total_disks"`
	// SpareDisks is the count of spare disks in the array.
	SpareDisks int `json:"spare_disks"`
	// FailedDisks is the count of failed member disks, from `mdadm
	// --detail`'s Failed Devices.
	FailedDisks int `json:"failed_disks"`
	// WorkingDisks is the count of working member disks, from `mdadm
	// --detail`'s Working Devices.
	WorkingDisks int `json:"working_disks"`
	// Members lists the active member device names (e.g. ["sda1","sdb1"]).
	Members []string `json:"members"`
	// Spares lists the spare member device names, marked (S) in
	// /proc/mdstat.
	Spares []string `json:"spares"`
	// Faulty lists the member device names that /proc/mdstat marks (F),
	// which the kernel has failed out of the array.
	Faulty []string `json:"faulty"`
	// Journal lists the journal device names, marked (J) in /proc/mdstat.
	// A journal device holds the write journal and is not a data member.
	Journal []string `json:"journal"`
	// DetailError is the failure of `mdadm --detail` for this array, such
	// as an inactive array or a user that can't open the device. The
	// array keeps its /proc/mdstat fields, and the --detail fields stay
	// empty. Empty when --detail ran or mdadm is not on the host.
	DetailError string `json:"detail_error,omitempty"`
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
func (base) Category() collector.Category { return collector.CategoryLinux }

// DefaultEnabled returns false, because mdadm is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the mdadm collector variant appropriate to the detected
// host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
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
			Faulty:  []string{},
			Journal: []string{},
		}

		for _, token := range rest {
			m := memberRE.FindStringSubmatch(token)
			if m == nil {
				continue
			}
			memberDev := m[1]
			memberType := m[2] // "S" spare, "F" faulty, "J" journal, "" active
			switch memberType {
			case "S":
				arr.Spares = append(arr.Spares, memberDev)
			case "F":
				arr.Faulty = append(arr.Faulty, memberDev)
			case "J":
				arr.Journal = append(arr.Journal, memberDev)
			default:
				arr.Members = append(arr.Members, memberDev)
			}
		}

		devices[devName] = arr
	}
	return devices
}

// The fields of `mdadm --detail`. Each is anchored to the start of its
// line, so "Array State :" or "Resync Status :" does not match "State :".
var (
	// raidLevelRE matches "Raid Level : raidN" or "RAID Level : raidN".
	raidLevelRE = regexp.MustCompile(`(?i)^\s*Raid Level\s*:\s*(\S+)`)
	// arrayStateRE matches "State : clean, degraded", the whole list.
	arrayStateRE = regexp.MustCompile(`^\s*State\s*:\s*(.+?)\s*$`)
	// uuidRE matches "UUID : xxxxxxxx:xxxxxxxx:xxxxxxxx:xxxxxxxx".
	uuidRE = regexp.MustCompile(`^\s*UUID\s*:\s*(\S+)`)
	// activeDevicesRE matches "Active Devices : N".
	activeDevicesRE = regexp.MustCompile(`^\s*Active Devices\s*:\s*(\d+)`)
	// totalDevicesRE matches "Total Devices : N".
	totalDevicesRE = regexp.MustCompile(`^\s*Total Devices\s*:\s*(\d+)`)
	// spareDevicesRE matches "Spare Devices : N".
	spareDevicesRE = regexp.MustCompile(`^\s*Spare Devices\s*:\s*(\d+)`)
	// failedDevicesRE matches "Failed Devices : N".
	failedDevicesRE = regexp.MustCompile(`^\s*Failed Devices\s*:\s*(\d+)`)
	// workingDevicesRE matches "Working Devices : N".
	workingDevicesRE = regexp.MustCompile(`^\s*Working Devices\s*:\s*(\d+)`)
)

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
		if m := failedDevicesRE.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				arr.FailedDisks = v
			}
		}
		if m := workingDevicesRE.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				arr.WorkingDisks = v
			}
		}
	}
}

// collectArrays builds the full Array list by parsing /proc/mdstat and
// running `mdadm --detail` for each array. fs is the host's filesystem;
// exec runs its commands, and a nil exec skips --detail. A missing /proc/mdstat or a
// host with no mdadm gives the mdstat view. A --detail that fails for one
// array, such as an inactive one, is that array's DetailError, so the
// other arrays keep their facts. A failed /proc/mdstat read or a done ctx
// is an error.
func collectArrays(
	ctx context.Context,
	fs avfs.VFS,
	exec source.Runner,
) ([]Array, error) {
	b, err := fs.ReadFile("/proc/mdstat")
	if err != nil {
		if collector.Absent(err) {
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
	var errs []error
	for _, name := range names {
		arr := devices[name]
		if exec != nil {
			detail, execErr := exec.Execute(ctx, "mdadm", "--detail", "/dev/"+name)
			switch {
			case execErr == nil:
				applyDetail(arr, detail)
			case ctx.Err() != nil:
				return nil, fmt.Errorf("mdadm --detail /dev/%s: %w", name, ctx.Err())
			case !collector.Absent(execErr):
				err := fmt.Errorf("mdadm --detail /dev/%s: %w", name, execErr)
				arr.DetailError = err.Error()
				errs = append(errs, err)
			}
		}
		arrays = append(arrays, *arr)
	}
	// An array whose detail failed keeps its mdstat facts and its
	// DetailError: the result is partial, with each failure joined.
	return arrays, errors.Join(errs...)
}
