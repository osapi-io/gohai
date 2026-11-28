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
	"fmt"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
)

// idSource is one place Linux keeps a machine ID. valid says whether the
// trimmed, lower-case value is a real ID.
type idSource struct {
	path  string
	valid func(id string) bool
}

// machineIDRE is the machine-id(5) format: 32 lower-case hex digits. It
// rules out the `uninitialized` that systemd writes on a first-boot image.
var machineIDRE = pattern.New(`^[0-9a-f]{32}$`)

// idSources are where Linux keeps a stable machine ID, in the order Ohai
// reads them: systemd's, then the dbus one of hosts before systemd, then
// the DMI product UUID, which only root can read on most hosts.
var idSources = []idSource{
	{path: "/etc/machine-id", valid: machineIDRE.MatchString},
	{path: "/var/lib/dbus/machine-id", valid: machineIDRE.MatchString},
	{path: "/sys/class/dmi/id/product_uuid", valid: func(id string) bool { return id != "" }},
}

// Linux reads the machine ID from the first of idSources that has one.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the machine ID and the file it came from, or an empty
// one when no file has a valid ID. A missing file is skipped, and a file
// that can't be read is an error.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	for _, src := range idSources {
		b, err := l.FS.ReadFile(src.path)
		if collector.Absent(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", src.path, err)
		}
		if id := strings.ToLower(strings.TrimSpace(string(b))); src.valid(id) {
			return &Info{ID: id, Source: src.path}, nil
		}
	}

	return &Info{}, nil
}
