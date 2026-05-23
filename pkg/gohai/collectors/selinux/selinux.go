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

// Package selinux reports SELinux status, mode, and policy information.
// On Darwin the collector returns nil gracefully — SELinux is a
// Linux-only security framework. On Linux the collector reads
// /etc/selinux/config for the configured policy and mode, and runs
// `sestatus` for the runtime mode, policy version, and kernel policy
// version. The collector is opt-in (DefaultEnabled false) because
// sestatus requires root on some kernels.
package selinux

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// selinuxConfigPath is the standard SELinux configuration file.
const selinuxConfigPath = "/etc/selinux/config"

// Info holds SELinux status and policy information.
type Info struct {
	// Status is "enabled" when SELinux is compiled into the kernel and
	// the config sets SELINUX to enforcing or permissive, "disabled"
	// when SELINUX=disabled in the config or SELinux is not compiled in.
	Status string `json:"status"`
	// CurrentMode is the runtime enforcement mode reported by sestatus:
	// "enforcing", "permissive", or "disabled".
	CurrentMode string `json:"current_mode,omitempty"`
	// ConfigMode is the SELINUX= value from /etc/selinux/config:
	// "enforcing", "permissive", or "disabled".
	ConfigMode string `json:"config_mode,omitempty"`
	// PolicyVersion is the running policy version string (e.g. "33").
	PolicyVersion string `json:"policy_version,omitempty"`
	// MaxKernelPolicyVersion is the maximum policy version the running
	// kernel supports (e.g. "33").
	MaxKernelPolicyVersion string `json:"max_kernel_policy_version,omitempty"`
	// LoadedPolicyName is the name of the loaded policy module
	// (e.g. "targeted", "mls", "minimum").
	LoadedPolicyName string `json:"loaded_policy_name,omitempty"`
}

// Collector is the public interface every selinux variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "selinux".
func (base) Name() string { return "selinux" }

// Category returns "security".
func (base) Category() string { return collector.CategorySecurity }

// DefaultEnabled returns false — SELinux collection is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the selinux collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseConfigFile reads SELINUX= and SELINUXTYPE= from /etc/selinux/config.
// Lines beginning with '#' and blank lines are skipped. Values are
// lowercased before being returned.
func parseConfigFile(
	fs avfs.VFS,
) (selinuxMode string, selinuxType string) {
	b, err := fs.ReadFile(selinuxConfigPath)
	if err != nil {
		return "", ""
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "SELINUX":
			selinuxMode = strings.ToLower(strings.TrimSpace(val))
		case "SELINUXTYPE":
			selinuxType = strings.ToLower(strings.TrimSpace(val))
		}
	}
	return selinuxMode, selinuxType
}

// parseSestatus parses the output of `sestatus` and populates the
// runtime fields in info. Ohai uses `sestatus -v -b` for booleans and
// contexts; we use plain `sestatus` for the core status fields.
//
// Example sestatus output:
//
//	SELinux status:                 enabled
//	SELinuxfs mount:                /sys/fs/selinux
//	Loaded policy name:             targeted
//	Current mode:                   enforcing
//	Mode from config file:          enforcing
//	Policy MLS status:              enabled
//	Policy deny_unknown status:     allowed
//	Memory protection checking:     actual (secure)
//	Max kernel policy version:      33
func parseSestatus(
	output []byte,
	info *Info,
) {
	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(val)
		switch key {
		case "selinux status":
			if val == "enabled" {
				info.Status = "enabled"
			} else {
				info.Status = "disabled"
			}
		case "current mode":
			info.CurrentMode = strings.ToLower(val)
		case "loaded policy name":
			info.LoadedPolicyName = val
		case "max kernel policy version":
			info.MaxKernelPolicyVersion = val
		case "policy version":
			info.PolicyVersion = val
		}
	}
}
