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

// Package source is what a collector reads a host through: a filesystem, a
// command runner, a clock and the host's platform family. Host is the real
// host. A caller that gives other sources, such as osapi's fake host in its
// tests, gets the same collectors reading that host instead.
package source

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/pkg/gohai/family"
)

// Runner runs a command and returns its combined stdout and stderr.
// Context cancellation stops the process.
type Runner interface {
	Execute(
		ctx context.Context,
		name string,
		args ...string,
	) ([]byte, error)
}

// Sources is what a collector reads. A zero field is the real host's.
type Sources struct {
	// FS is the filesystem: /proc, /sys, /etc and the rest.
	FS avfs.VFS
	// Exec runs the host's programs, such as sysctl, df and vm_stat.
	Exec Runner
	// Now is the clock, for a value computed from the time, such as uptime
	// from the boot time.
	Now func() time.Time
	// Family picks the collector's variant, as family.Detect names it, such
	// as debian or darwin.
	Family string
}

// Host is the real host: its filesystem, its programs, its clock and its
// family.
func Host() Sources {
	return Sources{}.Defaults()
}

// Defaults fills each zero field of s with the real host's.
func (s Sources) Defaults() Sources {
	if s.FS == nil {
		s.FS = osfs.NewWithNoIdm()
	}
	if s.Exec == nil {
		s.Exec = Commands()
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Family == "" {
		s.Family = family.Detect()
	}

	return s
}

// commands runs the host's programs.
type commands struct{}

// Commands is the Runner of the real host.
func Commands() Runner {
	return commands{}
}

// Execute runs name with args, and returns the combined output. On a
// non-zero exit it returns the output and an error that wraps the
// *exec.ExitError.
func (commands) Execute(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("exec %s: %w", name, err)
	}

	return out, nil
}
