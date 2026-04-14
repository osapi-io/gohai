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

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
)

// linuxZoneinfoPrefix is stripped from the symlink target to yield the
// IANA name. Standard Linux (glibc tzdata) layout.
const linuxZoneinfoPrefix = "/usr/share/zoneinfo/"

// timezonePath is the Debian/Ubuntu fallback when /etc/localtime is a
// copied file rather than a symlink (common in container images).
const timezonePath = "/etc/timezone"

// Linux collects timezone facts on Linux hosts. Embeds base for
// Name/DefaultEnabled/Dependencies.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the timezone Info. Never errors — missing sources
// leave fields empty, clock values still populate from Go's runtime.
func (l *Linux) Collect(
	_ context.Context,
) (any, error) {
	abbrev, offset := clockZone()
	name := resolveName(
		l.FS.Readlink,
		func() (string, error) {
			b, err := l.FS.ReadFile(timezonePath)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		linuxZoneinfoPrefix,
	)
	return &Info{Name: name, Abbrev: abbrev, Offset: offset}, nil
}
