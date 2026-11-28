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

package services

import (
	"context"
	"errors"
	"fmt"
	osexec "os/exec"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects systemd service states on Linux hosts by running
// `systemctl list-units --type=service --all --no-pager --plain` for each
// unit's state and description, and `systemctl list-unit-files` for
// whether it is enabled.
type Linux struct {
	base

	// FS tells a host with systemd from one without: systemd makes
	// /run/systemd/system when it boots.
	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// systemdRun is the directory systemd makes when it is the init system.
const systemdRun = "/run/systemd/system"

// Collect runs systemctl and parses its output. A host that systemd did not
// boot, such as a container, has no services and no error. On a host that
// systemd booted, a systemctl that fails is an error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return &Info{Services: []Service{}}, nil
	}
	if l.FS != nil {
		if _, err := l.FS.Stat(systemdRun); err != nil {
			return &Info{Services: []Service{}}, nil
		}
	}
	out, err := l.Exec.Execute(
		ctx,
		"systemctl",
		"list-units",
		"--type=service",
		"--all",
		"--no-pager",
		"--plain",
	)
	if err != nil {
		if errors.Is(err, osexec.ErrNotFound) {
			return &Info{Services: []Service{}}, nil
		}
		return nil, fmt.Errorf("systemctl list-units: %w", err)
	}
	// Enabled is the unit-file state, not whether the unit runs now.
	files, err := l.Exec.Execute(
		ctx,
		"systemctl",
		"list-unit-files",
		"--type=service",
		"--no-legend",
		"--no-pager",
	)
	if err != nil {
		// The loaded services are good. Without the unit-file states each
		// reads as not enabled, so the result is partial.
		return parseSystemctlOutput(string(out), nil),
			fmt.Errorf("systemctl list-unit-files: %w", err)
	}
	return parseSystemctlOutput(string(out), parseUnitFiles(string(files))), nil
}
