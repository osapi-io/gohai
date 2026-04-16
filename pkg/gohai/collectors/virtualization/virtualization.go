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

// Package virtualization detects every hypervisor and container
// runtime the host participates in — as guest, host, or both. A
// single host can legitimately report multiple systems (a Docker host
// that is itself a KVM guest on EC2, an LXD host on bare metal,
// etc.). Mirrors Ohai's linux/virtualization.rb and
// darwin/virtualization.rb cascades.
package virtualization

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds virtualization detection data.
type Info struct {
	System         string            `json:"system,omitempty"`          // primary / innermost runtime ("docker", "kvm", "vmware", "")
	Role           string            `json:"role,omitempty"`            // "host" | "guest" | ""
	Systems        map[string]string `json:"systems,omitempty"`         // every detected layer: {"kvm": "guest", "docker": "host"}
	HypervisorHost string            `json:"hypervisor_host,omitempty"` // hostname of the hypervisor, when the guest can see it (Hyper-V KVP pool)
}

// Collector is the public interface every variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string         { return "virtualization" }
func (base) Category() string     { return collector.CategoryVirtualization }
func (base) DefaultEnabled() bool { return true }

// Dependencies declares cpu — the Linux cascade consults the cpu
// prior result's HypervisorVendor/VirtualizationType as a KVM
// fallback when /sys/devices/virtual/misc/kvm isn't exposed
// (nested VMs). Every other signal reads /proc or /sys directly.
func (base) Dependencies() []string { return []string{"cpu"} }

// New returns the variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// addSystem records a detected layer in info.Systems and updates the
// primary System/Role to the most recent positive detection. Layers
// are preserved on conflict unless force is true (used for `.dockerenv`
// and other authoritative signals Ohai overrides on). Callers always
// pass non-empty literal name/role; no defensive empty-string check.
func addSystem(
	info *Info,
	name, role string,
	force bool,
) {
	if info.Systems == nil {
		info.Systems = map[string]string{}
	}
	if _, exists := info.Systems[name]; !exists || force {
		info.Systems[name] = role
	}
	info.System = name
	info.Role = role
}
