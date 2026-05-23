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
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// vmwareToolboxCmdPath is the canonical path for vmware-toolbox-cmd on Linux.
const vmwareToolboxCmdPath = "/usr/bin/vmware-toolbox-cmd"

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
func (base) Category() string { return collector.CategoryVirtualization }

// DefaultEnabled returns false — vmware is opt-in only.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the vmware collector variant appropriate to the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
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
