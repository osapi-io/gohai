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

package timezone

import (
	"context"
	"strings"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// timezonePath is the Debian/Ubuntu fallback when /etc/localtime is a
// copied file rather than a symlink (common in container images).
const timezonePath = "/etc/timezone"

// sysconfigClock is the RHEL and SUSE fallback: its ZONE= line names the
// zone when /etc/localtime is a copied file.
const sysconfigClock = "/etc/sysconfig/clock"

// Linux collects timezone facts on Linux hosts. Embeds base for
// Name/DefaultEnabled/Dependencies.
type Linux struct {
	base

	FS  avfs.VFS
	Now func() time.Time
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Now: time.Now}
}

// Collect returns the timezone Info. It never errors: a missing source
// leaves the name empty.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	abbrev, offset := clockZone(l.FS, l.Now)
	name := resolveName(
		l.FS.Readlink,
		func() (string, error) {
			b, err := l.FS.ReadFile(timezonePath)
			return string(b), err
		},
		func() (string, error) {
			b, err := l.FS.ReadFile(sysconfigClock)
			if err != nil {
				return "", err
			}
			for _, line := range strings.Split(string(b), "\n") {
				if v, ok := strings.CutPrefix(strings.TrimSpace(line), "ZONE="); ok {
					return strings.Trim(v, `"'`), nil
				}
			}
			return "", nil
		},
	)
	return &Info{Name: name, Abbrev: abbrev, Offset: offset}, nil
}
