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

package systemdpaths

import (
	"context"
	"fmt"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux reports standard systemd directory paths on Linux by running
// `systemd-path`. When the binary is absent or returns an error, an
// empty Paths map is returned without error — matches Ohai's no-panic
// behaviour.
type Linux struct {
	base

	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands()}
}

// Collect runs `systemd-path` and parses its output. A host without
// systemd-path gives an empty Info, and a failed run is an error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return &Info{Paths: make(map[string]string)}, nil
	}
	out, err := l.Exec.Execute(ctx, "systemd-path")
	if err != nil {
		if collector.Absent(err) {
			return &Info{Paths: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("systemd-path: %w", err)
	}
	return parseSystemdPath(string(out)), nil
}
