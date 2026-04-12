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

// Package virtualization detects the hypervisor / container runtime the
// host is running under.
//
// Known limitation vs. Ohai: Ohai's virtualization plugin additionally
// reports a `systems` map (for hosts that are both docker host AND kvm
// host, both entries appear) plus detailed container detection
// (nspawn, LXD, podman host, hyper-v guest). Our current coverage is
// what gopsutil exposes — system + role only. `systems` map and
// container detail are tracked as a follow-up.
package virtualization

import (
	"context"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds virtualization detection data.
type Info struct {
	System string `json:"system,omitempty"` // hypervisor/container: "docker", "kvm", "vmware", "lxc", ""
	Role   string `json:"role,omitempty"`   // "host" | "guest" | ""
}

// Collector is the public interface every virtualization variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "virtualization" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the virtualization variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// hostVirtualization is the injection seam for gopsutil's
// host.VirtualizationWithContext. Tests swap this to exercise both the
// success and error branches of detect on any host OS without hitting
// the real syscall.
var hostVirtualization = host.VirtualizationWithContext

// detect is the production bridge to gopsutil's host.VirtualizationWithContext.
func detect(
	ctx context.Context,
) (*Info, error) {
	system, role, err := hostVirtualization(ctx)
	if err != nil {
		return nil, err
	}
	return &Info{System: system, Role: role}, nil
}
