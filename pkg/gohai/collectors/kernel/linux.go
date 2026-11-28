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

package kernel

import (
	"context"
	"fmt"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// The files of /proc/sys/kernel that have the fields of uname(2).
const (
	procOSType    = "/proc/sys/kernel/ostype"
	procOSRelease = "/proc/sys/kernel/osrelease"
	procVersion   = "/proc/sys/kernel/version"
	// procArch is the machine, on Linux 6.1 and later. An older kernel does
	// not have it, and the collector runs uname -m.
	procArch = "/proc/sys/kernel/arch"
)

// Linux collects kernel identity on Linux from /proc/sys/kernel, the
// fields that uname(2) reports, and the machine from
// /proc/sys/kernel/arch or uname -m. `processor` and `os` are synthesized
// per Option A of issue #29 (Machine and the static string "GNU/Linux")
// rather than shelling out to `uname -p` / `uname -o`.
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant that reads the host's filesystem and
// runs its commands.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect returns kernel Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	name, err := l.FS.ReadFile(procOSType)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procOSType, err)
	}
	release, err := l.FS.ReadFile(procOSRelease)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procOSRelease, err)
	}
	// The build string is optional, as uname -v is.
	version, _ := l.FS.ReadFile(procVersion)
	machine, err := l.machine(ctx)
	if err != nil {
		return nil, err
	}

	return &Info{
		Name:      field(name),
		Release:   field(release),
		Version:   field(version),
		Machine:   machine,
		Processor: machine,
		OS:        "GNU/Linux",
	}, nil
}

// machine is the hardware architecture: /proc/sys/kernel/arch, or uname -m
// on a kernel without it. With no Exec, a kernel without arch gives an
// empty machine.
func (l *Linux) machine(
	ctx context.Context,
) (string, error) {
	if b, err := l.FS.ReadFile(procArch); err == nil && field(b) != "" {
		return field(b), nil
	}
	if l.Exec == nil {
		return "", nil
	}
	out, err := l.Exec.Execute(ctx, "uname", "-m")
	if err != nil {
		return "", fmt.Errorf("uname -m: %w", err)
	}

	return field(out), nil
}
