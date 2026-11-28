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

package hostnamectl

import (
	"context"
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux reports hostnamectl fields on Linux by running `hostnamectl`
// and parsing its "Key: value" output. Matches Ohai's linux/hostnamectl
// plugin. A host with no `hostnamectl` gives an empty Info; a
// hostnamectl that fails is an error.
type Linux struct {
	base

	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands()}
}

// Collect runs `hostnamectl` and returns the parsed Info. A missing
// binary yields an empty Info; any other exec error is returned.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return &Info{}, nil
	}
	out, err := l.Exec.Execute(ctx, "hostnamectl")
	if collector.Absent(err) || noSystemd(err) {
		return &Info{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read hostnamectl: %w", err)
	}
	return parseHostnamectl(string(out)), nil
}

// noSystemd reports whether err says that hostnamectl is installed but
// systemd does not run, as in a container with the systemd package and
// another init. Such a host has no hostnamed to ask, the same as a host
// without hostnamectl. The error has hostnamectl's stderr.
func noSystemd(
	err error,
) bool {
	if err == nil {
		return false
	}
	msg := err.Error()

	return strings.Contains(msg, "Failed to connect to bus") ||
		strings.Contains(msg, "System has not been booted with systemd")
}
