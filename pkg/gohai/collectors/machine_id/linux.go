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

package machineid

import (
	"context"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

// dbusMachineIDPath is the pre-systemd machine-id location Ohai reads
// that gopsutil's HostID doesn't check. Present on older Debian/Ubuntu
// hosts that ship dbus without systemd.
const dbusMachineIDPath = "/var/lib/dbus/machine-id"

// Linux resolves the machine ID on Linux hosts. Wraps gopsutil's
// host.Info (which reads /etc/machine-id → /sys/class/dmi/id/product_uuid
// → /proc/sys/kernel/random/boot_id) and extends with a
// /var/lib/dbus/machine-id fallback so pre-systemd Debian hosts
// without DMI still get a stable ID.
type Linux struct {
	base

	// HostInfoFn wraps gopsutil.host.InfoWithContext.
	HostInfoFn func(context.Context) (*host.InfoStat, error)
	// ReadFileFn reads a file. Used for the /var/lib/dbus/machine-id
	// fallback. Wired to os.ReadFile; tests inject stubs.
	ReadFileFn func(string) ([]byte, error)
}

// NewLinux returns a Linux variant wired to gopsutil + os.ReadFile.
func NewLinux() *Linux {
	return &Linux{
		HostInfoFn: host.InfoWithContext,
		ReadFileFn: os.ReadFile,
	}
}

// Collect returns the machine ID. Extends gopsutil — if gopsutil
// returns an empty ID (no /etc/machine-id, no DMI product_uuid),
// fall back to /var/lib/dbus/machine-id before giving up. This
// mirrors Ohai's fallback chain without re-implementing gopsutil's
// existing work.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	info, err := l.HostInfoFn(ctx)
	if err != nil {
		return nil, err
	}
	if info != nil && info.HostID != "" {
		return &Info{ID: info.HostID}, nil
	}
	// gopsutil returned empty — try the pre-systemd dbus location
	// before reporting unknown.
	if b, err := l.ReadFileFn(dbusMachineIDPath); err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return &Info{ID: id}, nil
		}
	}
	return &Info{}, nil
}
