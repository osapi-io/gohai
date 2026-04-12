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
	"os"
	"time"
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

	// ReadlinkFn reads /etc/localtime's symlink target.
	ReadlinkFn func(string) (string, error)
	// ReadFileFn reads /etc/timezone for the Debian-style fallback.
	ReadFileFn func(string) ([]byte, error)
	// NowFn returns the current time for zone abbreviation + offset.
	NowFn func() time.Time
}

// NewLinux returns a Linux variant wired to stdlib.
func NewLinux() *Linux {
	return &Linux{
		ReadlinkFn: os.Readlink,
		ReadFileFn: os.ReadFile,
		NowFn:      time.Now,
	}
}

// Collect returns the timezone Info. Never errors — missing sources
// leave fields empty, clock values still populate from Go's runtime.
func (l *Linux) Collect(_ context.Context) (any, error) {
	abbrev, offset := clockZone(l.NowFn)
	name := resolveName(
		l.ReadlinkFn,
		func() (string, error) {
			b, err := l.ReadFileFn(timezonePath)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		linuxZoneinfoPrefix,
	)
	return &Info{Name: name, Abbrev: abbrev, Offset: offset}, nil
}
