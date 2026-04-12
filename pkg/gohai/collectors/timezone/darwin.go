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
	"errors"
	"os"
	"time"
)

// darwinZoneinfoPrefix is the macOS-specific zoneinfo database path
// (Apple maintains its own tzdata symlinked under /var/db/timezone).
const darwinZoneinfoPrefix = "/var/db/timezone/zoneinfo/"

// Darwin collects timezone facts on macOS hosts. Embeds base for
// Name/DefaultEnabled/Dependencies.
type Darwin struct {
	base

	ReadlinkFn func(string) (string, error)
	NowFn      func() time.Time
}

// NewDarwin returns a Darwin variant wired to stdlib.
func NewDarwin() *Darwin {
	return &Darwin{
		ReadlinkFn: os.Readlink,
		NowFn:      time.Now,
	}
}

// Collect returns the timezone Info. macOS has no /etc/timezone
// equivalent — if /etc/localtime isn't a symlink the Name stays empty
// (rare on real macs; seen in some CI sandboxes).
func (d *Darwin) Collect(_ context.Context) (any, error) {
	abbrev, offset := clockZone(d.NowFn)
	name := resolveName(
		d.ReadlinkFn,
		func() (string, error) { return "", errors.New("no fallback on darwin") },
		darwinZoneinfoPrefix,
	)
	return &Info{Name: name, Abbrev: abbrev, Offset: offset}, nil
}
