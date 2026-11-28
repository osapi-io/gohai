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

package vmware

import (
	"context"
	"fmt"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// procScsiPath is the SCSI device list on Linux. We check it for a
// VMware SCSI controller to confirm we're running on a VMware guest
// before shelling out to vmware-toolbox-cmd. Mirrors Ohai's vmware.rb
// which gates collection on virtualization[:systems][:vmware].
const procScsiPath = "/proc/scsi/scsi"

// Linux collects VMware Tools data on Linux hosts. Detection first checks
// /proc/scsi/scsi for a VMware SCSI controller (fast path without
// executing any binary). If detected, or if vmware-toolbox-cmd exists at
// the canonical path, we proceed to collect via the CLI.
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the real OS filesystem and commands.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: source.Commands(),
	}
}

// Collect gathers VMware Tools statistics. It returns no fact when the
// host is not a VMware guest or VMware Tools is not installed. When a call
// to vmware-toolbox-cmd does not finish, it returns the values read before
// that call with the error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	guest, err := l.isVMwareGuest(ctx)
	if err != nil || !guest {
		return nil, err
	}

	return collectVMwareTools(ctx, l.Exec)
}

// isVMwareGuest reports whether the host is a VMware guest with VMware
// Tools. /proc/scsi/scsi naming a VMware controller is the fast signal, and
// `vmware-toolbox-cmd -v` the other. A host with no tools is not a guest
// for this collector. A tools command that fails is an error.
func (l *Linux) isVMwareGuest(
	ctx context.Context,
) (bool, error) {
	b, err := l.FS.ReadFile(procScsiPath)
	if err == nil && strings.Contains(string(b), "VMware") {
		return l.Exec != nil, nil
	}
	if err != nil && !collector.Absent(err) {
		return false, fmt.Errorf("read %s: %w", procScsiPath, err)
	}
	if l.Exec == nil {
		return false, nil
	}

	return probeToolbox(ctx, l.Exec)
}
