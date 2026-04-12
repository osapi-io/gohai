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

package platform

import (
	"context"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

// Darwin collects platform identification on macOS. Wraps gopsutil
// for the basic name/version/family and extends with `sw_vers
// -productVersionExtra` for the RSR patch version Ohai exposes as
// platform_version_extra.
type Darwin struct {
	base

	HostInfoFn func(context.Context) (*host.InfoStat, error)
	// RunCmdFn invokes an OS command and returns its stdout. Injectable
	// for tests — production uses exec.Command.
	RunCmdFn func(name string, args ...string) ([]byte, error)
}

// NewDarwin returns a Darwin variant wired to gopsutil + exec.Command.
func NewDarwin() *Darwin {
	return &Darwin{
		HostInfoFn: host.InfoWithContext,
		RunCmdFn:   runSwVers,
	}
}

// runSwVers wraps exec.Command for sw_vers. Named helper so the
// factory assigns a function reference (no closure body to cover).
func runSwVers(
	name string,
	args ...string,
) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// Collect returns platform Info. Probes sw_vers -productVersionExtra
// to pick up RSR patch suffixes (e.g. "(a)" on macOS 14.4.1 (a)).
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	h, err := d.HostInfoFn(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if h != nil {
		info.Name = canonicalizePlatform(h.Platform)
		info.Version = h.PlatformVersion
		info.Family = h.PlatformFamily
		info.Build = h.KernelVersion
	}
	if extra := readVersionExtra(d.RunCmdFn); extra != "" {
		info.VersionExtra = extra
	}
	return info, nil
}

// readVersionExtra reads `sw_vers -productVersionExtra`. Returns empty
// string on any failure — RSR version is optional and absent on most
// macOS versions.
func readVersionExtra(
	run func(string, ...string) ([]byte, error),
) string {
	out, err := run("sw_vers", "-productVersionExtra")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
