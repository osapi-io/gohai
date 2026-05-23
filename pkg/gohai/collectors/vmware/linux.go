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

package vmware

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// procScsiPath is the SCSI device list on Linux. We check it for a
// VMware SCSI controller to confirm we're running on a VMware guest
// before shelling out to vmware-toolbox-cmd. Mirrors Ohai's vmware.rb
// which gates collection on virtualization[:systems][:vmware].
const procScsiPath = "/proc/scsi/scsi"

// Linux collects VMware Tools data on Linux hosts. Detection first checks
// /proc/scsi/scsi for a VMware SCSI controller (fast path without
// executing any binary). If detected, or if vmware-toolbox-cmd exists at
// the canonical path, we proceed to collect via the CLI.
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the real OS filesystem and executor.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: executor.New(),
	}
}

// Collect gathers VMware Tools statistics. Returns nil (no error) when not
// running on a VMware guest or when VMware Tools is not installed.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if !l.isVMwareGuest(ctx) {
		return nil, nil
	}
	return collectVMwareTools(ctx, l.Exec), nil
}

// isVMwareGuest returns true if running inside a VMware guest. It checks
// /proc/scsi/scsi for a VMware SCSI controller first (no exec needed),
// then falls back to testing whether vmware-toolbox-cmd is present by
// running `vmware-toolbox-cmd -v`. Either signal is sufficient.
func (l *Linux) isVMwareGuest(
	ctx context.Context,
) bool {
	b, err := l.FS.ReadFile(procScsiPath)
	if err == nil && strings.Contains(string(b), "VMware") {
		return true
	}
	if l.Exec == nil {
		return false
	}
	out, err := l.Exec.Execute(ctx, vmwareToolboxCmdPath, "-v")
	return err == nil && len(out) > 0
}

// collectVMwareTools runs vmware-toolbox-cmd for each stat/status and
// assembles an Info struct. Errors from individual commands are silently
// ignored — partial data is still useful and matches Ohai's rescue-based
// approach.
func collectVMwareTools(
	ctx context.Context,
	exec executor.Executor,
) *Info {
	info := &Info{}

	run := func(args ...string) string {
		out, err := exec.Execute(ctx, vmwareToolboxCmdPath, args...)
		if err != nil {
			return ""
		}
		return trimStat(string(out))
	}

	// stat subcommand parameters — mirrors Ohai's iteration.
	info.Version = run("-v")
	info.Hosttime = run("stat", "hosttime")
	info.Speed = run("stat", "speed")
	info.SessionID = run("stat", "sessionid")
	info.Balloon = run("stat", "balloon")
	info.Swap = run("stat", "swap")
	info.MemLimit = run("stat", "memlimit")
	info.MemRes = run("stat", "memres")
	info.CPURes = run("stat", "cpures")
	info.CPULimit = run("stat", "cpulimit")

	// status subcommand parameters.
	info.UpgradeStatus = run("upgrade", "status")
	info.TimesyncStatus = run("timesync", "status")

	// Distinguish desktop vs vSphere by querying the raw session JSON.
	// An empty response means VMware Workstation/Fusion (desktop).
	// A JSON response with a "version" key means vSphere.
	rawSession := run("stat", "raw", "json", "session")
	if rawSession == "" {
		info.HostType = "vmware_desktop"
	} else {
		info.HostType = "vmware_vsphere"
		var sess struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal([]byte(rawSession), &sess); err == nil {
			info.HostVersion = sess.Version
		}
	}

	return info
}
