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
	"strings"
)

// Darwin collects platform identification on macOS. ReadFn is typed in
// our *Info so importers don't need gopsutil; RunCmdFn wraps
// exec.Command for the sw_vers -productVersionExtra call.
type Darwin struct {
	base

	ReadFn   func(context.Context) (*Info, string, error)
	RunCmdFn func(name string, args ...string) ([]byte, error)
}

// NewDarwin returns a Darwin variant wired to production bridges.
func NewDarwin() *Darwin {
	return &Darwin{
		ReadFn:   readPlatform,
		RunCmdFn: runSwVers,
	}
}

// runSwVers wraps exec.Command for sw_vers.
func runSwVers(
	name string,
	args ...string,
) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// Collect returns platform Info with Build set from the kernel version
// and VersionExtra populated from sw_vers -productVersionExtra.
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	info, kernelVer, err := d.ReadFn(ctx)
	if err != nil {
		return nil, err
	}
	info.Build = kernelVer
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
