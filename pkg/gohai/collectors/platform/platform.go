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

// Package platform collects operating system platform identification facts.
package platform

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	plat "github.com/osapi-io/gohai/internal/platform"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// via SetHostInfoFn (export_test.go).
var hostInfoFn = host.InfoWithContext

// readPlatform is the production bridge. Wraps the private gopsutil
// call and returns a pre-populated *Info plus the raw KernelVersion
// string that Darwin consumes as Build (Linux's Info does not carry
// Build; it discards the kernel string). OS/Architecture are always
// set from runtime; Name/Version/Family come from gopsutil when
// available. Per-OS Collect wrappers layer additional fields
// (VersionExtra on darwin) on top of the returned Info.
func readPlatform(
	ctx context.Context,
) (*Info, string, error) {
	info := &Info{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	h, err := hostInfoFn(ctx)
	if err != nil {
		return nil, "", err
	}
	if h == nil {
		return info, "", nil
	}
	info.Name = canonicalizePlatform(h.Platform)
	info.Version = h.PlatformVersion
	info.Family = h.PlatformFamily
	return info, h.KernelVersion, nil
}

// Info holds platform identification data.
type Info struct {
	OS           string `json:"os"`                      // runtime.GOOS: "linux", "darwin", "windows"
	Name         string `json:"name"`                    // distro/product: "ubuntu", "redhat", "darwin"
	Version      string `json:"version"`                 // "24.04", "14.4.1"
	VersionExtra string `json:"version_extra,omitempty"` // extra version info (macOS RSR patches)
	Family       string `json:"family"`                  // "debian", "rhel", "mac_os_x"
	Architecture string `json:"architecture"`            // "amd64", "arm64"
	Build        string `json:"build,omitempty"`         // kernel build (macOS)
}

// Collector is the public interface every platform variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "platform" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the platform variant for the host OS.
func New() Collector {
	if plat.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// platformIDRemap is Ohai's ID normalization table — converts
// distro identifiers into a canonical form so consumers don't have to
// special-case per-distro quirks. Matches Ohai's
// OS_RELEASE_PLATFORM_REMAP list.
//
// Source: https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb
var platformIDRemap = map[string]string{
	"alinux":        "alibabalinux",
	"amzn":          "amazon",
	"ol":            "oracle",
	"sles":          "suse",
	"opensuse-leap": "opensuseleap",
	"rhel":          "redhat",
	"rocky":         "rocky",
	"xenenterprise": "xenserver",
}

// canonicalizePlatform applies platformIDRemap to the given platform
// string. Returns the input unchanged if no remap entry exists.
func canonicalizePlatform(
	p string,
) string {
	if m, ok := platformIDRemap[p]; ok {
		return m
	}
	return p
}
