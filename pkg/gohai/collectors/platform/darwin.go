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
	"bufio"
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects platform identification on macOS. gopsutil's
// host.Info provides Name/Version/Family; we additionally run
// `sw_vers` through the shared Executor and parse:
//
//   - ProductVersionExtra → VersionExtra (Apple Rapid Security
//     Response patch suffix, e.g., "(a)" — empty when no RSR is
//     applied).
//   - BuildVersion → Build (e.g., "23E224") — preferred over
//     gopsutil's KernelVersion field as the macOS-canonical build id.
//
// Falls back to gopsutil's KernelVersion for Build when sw_vers fails.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns platform Info.
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	info, kernelVer, err := readPlatform(ctx)
	if err != nil {
		return nil, err
	}
	info.Build = kernelVer
	if d.Exec != nil {
		applySwVers(ctx, d.Exec, info)
	}
	return info, nil
}

// applySwVers parses `sw_vers` output (key: value lines) and
// supplements info with Build + VersionExtra. Silent on exec failure.
func applySwVers(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	out, err := exec.Execute(ctx, "sw_vers")
	if err != nil {
		return
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "BuildVersion":
			if val != "" {
				info.Build = val
			}
		case "ProductVersionExtra":
			if val != "" {
				info.VersionExtra = val
			}
		}
	}
}
