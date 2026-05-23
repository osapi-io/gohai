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

package virtualbox

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects VirtualBox guest additions data on Linux hosts via
// `VBoxControl guestproperty enumerate`. The command is only available when
// VirtualBox Guest Additions are installed. If the command fails (not a
// VirtualBox guest or Guest Additions not installed) Collect returns nil
// with no error — matching Ohai's rescue-based approach.
type Linux struct {
	base

	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: executor.New()}
}

// Collect runs `VBoxControl guestproperty enumerate` and parses the output.
// Returns nil (no error) when not running on a VirtualBox guest or when
// Guest Additions are not installed.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return nil, nil
	}
	out, err := l.Exec.Execute(ctx, "VBoxControl", "guestproperty", "enumerate")
	if err != nil {
		return nil, nil
	}
	return parseVBoxControlOutput(out), nil
}
