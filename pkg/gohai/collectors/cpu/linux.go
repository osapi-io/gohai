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

package cpu

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects CPU facts on Linux. gopsutil's `/proc/cpuinfo` parse
// is the primary source (via the package-level readCPUFn seam); we
// layer two extensions on top:
//
//   - vulnerability mitigation status via /sys/devices/system/cpu/vulnerabilities/*
//     (read through FS — an avfs.VFS, so tests can inject memfs content).
//   - NUMA topology, per-level cache sizes, and — on s390x / ppc64le —
//     authoritative core/socket/thread counts via `lscpu` (run through
//     Exec — an executor.Executor, so tests can mock the command).
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to production dependencies:
// the real OS filesystem for sysfs, and a real `os/exec` wrapper for
// `lscpu`.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: executor.New(),
	}
}

// Collect returns the CPU Info with Linux-specific extensions merged
// on top of the gopsutil base.
func (l *Linux) Collect(
	ctx context.Context,
) (any, error) {
	info, err := readCPUFn(ctx)
	if err != nil {
		return nil, err
	}
	if v := readVulnerabilities(l.FS); v != nil {
		info.Vulnerabilities = v
	}
	if out, err := l.Exec.Execute(ctx, "lscpu"); err == nil {
		applyLscpuToInfo(info, parseLscpu(out))
	}
	return info, nil
}
