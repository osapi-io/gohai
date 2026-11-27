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

package lsb

import (
	"context"
	"fmt"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux reports LSB identification fields via the `lsb_release` CLI —
// the authoritative source used by Ohai and mandated on RHEL-family
// hosts with `redhat-lsb-core` installed. Ohai deliberately removed
// the legacy /etc/lsb-release file fallback (chef/ohai#1562) because
// on modern Debian/Ubuntu the `lsb-release` package ships both the
// file and the CLI; we match that stance — when the CLI is absent,
// Info stays empty rather than parsing a possibly-stale file.
type Linux struct {
	base

	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands()}
}

// Collect runs `lsb_release -a` and parses its labelled lines. A host
// without lsb_release, or with empty output, gives an empty Info. A
// failed run is an error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return &Info{}, nil
	}
	out, err := l.Exec.Execute(ctx, "lsb_release", "-a")
	if collector.Absent(err) {
		return &Info{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read lsb_release: %w", err)
	}
	return parseLsbReleaseCLI(string(out)), nil
}
