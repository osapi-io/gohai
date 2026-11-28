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

// Package vmware collects VMware Tools data from a VMware guest VM.
// It requires VMware Tools (vmware-toolbox-cmd) to be installed. When the
// host is not a VMware guest, Collect returns nil with no error.
package vmware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// toolboxCmd is the VMware Tools command. It is found on the PATH, so an
// install outside /usr/bin, such as /usr/sbin or /usr/local/bin, is found.
const toolboxCmd = "vmware-toolbox-cmd"

// Info holds VMware guest tools data. All fields are empty strings when
// VMware Tools is absent or the collector is not running in a VMware guest.
type Info struct {
	Version        string `json:"version,omitempty"`         // vmware-toolbox-cmd -v output (e.g. "12.3.0 build-21581411")
	Hosttime       string `json:"hosttime,omitempty"`        // vmware-toolbox-cmd stat hosttime
	Speed          string `json:"speed,omitempty"`           // vmware-toolbox-cmd stat speed
	SessionID      string `json:"session_id,omitempty"`      // vmware-toolbox-cmd stat sessionid
	Balloon        string `json:"balloon,omitempty"`         // vmware-toolbox-cmd stat balloon
	Swap           string `json:"swap,omitempty"`            // vmware-toolbox-cmd stat swap
	MemLimit       string `json:"mem_limit,omitempty"`       // vmware-toolbox-cmd stat memlimit
	MemRes         string `json:"mem_res,omitempty"`         // vmware-toolbox-cmd stat memres
	CPURes         string `json:"cpu_res,omitempty"`         // vmware-toolbox-cmd stat cpures
	CPULimit       string `json:"cpu_limit,omitempty"`       // vmware-toolbox-cmd stat cpulimit
	UpgradeStatus  string `json:"upgrade_status,omitempty"`  // vmware-toolbox-cmd upgrade status
	TimesyncStatus string `json:"timesync_status,omitempty"` // vmware-toolbox-cmd timesync status
	HostType       string `json:"host_type,omitempty"`       // "vmware_vsphere" or "vmware_desktop"
	HostVersion    string `json:"host_version,omitempty"`    // vSphere host version (empty for desktop)
}

// Collector is the public interface every vmware variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "vmware".
func (base) Name() string { return "vmware" }

// Category returns "virtualization".
func (base) Category() collector.Category { return collector.CategoryVirtualization }

// DefaultEnabled returns false, because vmware is opt-in only.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the vmware collector variant appropriate to the detected host OS.
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

// trimStat strips trailing whitespace and filters "UpdateInfo failed"
// responses that vmware-toolbox-cmd emits when the stat is unavailable.
// Mirrors Ohai's vmware.rb check on the stat return value.
func trimStat(
	s string,
) string {
	v := strings.TrimSpace(s)
	if strings.Contains(v, "UpdateInfo failed") {
		return ""
	}
	return v
}

// probeToolbox reports whether `vmware-toolbox-cmd -v` names a VMware Tools
// version. A host without the command, or one where the command exits with
// an error, has no tools to read. A call that does not finish, such as one
// that times out, is an error.
func probeToolbox(
	ctx context.Context,
	runner source.Runner,
) (bool, error) {
	out, err := runner.Execute(ctx, toolboxCmd, "-v")
	switch {
	case err == nil:
		return strings.TrimSpace(string(out)) != "", nil
	case collector.Absent(err), exited(err):
		return false, nil
	default:
		return false, fmt.Errorf("%s -v: %w", toolboxCmd, err)
	}
}

// collectVMwareTools runs vmware-toolbox-cmd for each stat and status that
// Ohai's vmware.rb reads, and assembles an Info. A stat that the guest does
// not offer exits with an error, as Ohai's rescue allows, and leaves its
// field empty. Any other failure, such as a timeout, is an error, so a
// vSphere guest whose call did not finish is not named a desktop guest.
func collectVMwareTools(
	ctx context.Context,
	runner source.Runner,
) (*Info, error) {
	var runErr error
	run := func(args ...string) string {
		if runErr != nil {
			return ""
		}
		out, err := runner.Execute(ctx, toolboxCmd, args...)
		switch {
		case err == nil:
			return trimStat(string(out))
		case exited(err):
			return ""
		default:
			runErr = fmt.Errorf("%s %s: %w", toolboxCmd, strings.Join(args, " "), err)
			return ""
		}
	}

	info := &Info{
		Version:        run("-v"),
		Hosttime:       run("stat", "hosttime"),
		Speed:          run("stat", "speed"),
		SessionID:      run("stat", "sessionid"),
		Balloon:        run("stat", "balloon"),
		Swap:           run("stat", "swap"),
		MemLimit:       run("stat", "memlimit"),
		MemRes:         run("stat", "memres"),
		CPURes:         run("stat", "cpures"),
		CPULimit:       run("stat", "cpulimit"),
		UpgradeStatus:  run("upgrade", "status"),
		TimesyncStatus: run("timesync", "status"),
	}

	// vSphere answers the raw session query with JSON that has a version;
	// Workstation and Fusion (desktop) answer with an error or nothing.
	rawSession := run("stat", "raw", "json", "session")
	if runErr != nil {
		// The calls before the failure are good, so they stay, with the
		// error. HostType stays empty: the raw session query did not answer.
		return info, runErr
	}
	if rawSession == "" {
		info.HostType = "vmware_desktop"
		return info, nil
	}
	info.HostType = "vmware_vsphere"
	var sess struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(rawSession), &sess); err == nil {
		info.HostVersion = sess.Version
	}

	return info, nil
}

// exited reports whether err is a command that ran to its end and exited
// with a status of its own, such as a stat the guest does not offer. A
// command that a timeout or a signal killed has no exit status (ExitCode is
// -1), so it is not one: source.Commands returns a killed command as an
// *exec.ExitError too.
func exited(
	err error,
) bool {
	var status interface{ ExitCode() int }

	return errors.As(err, &status) && status.ExitCode() >= 0
}
