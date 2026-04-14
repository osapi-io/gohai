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

package users

import (
	"context"

	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects logged-in sessions on Linux. Prefers
// `loginctl list-sessions` on systemd hosts (catches GDM/KDE
// graphical sessions, remote desktop, systemd-run sessions that
// never reach utmp). Falls back to gopsutil's utmp read when
// loginctl is absent or errors (non-systemd hosts, minimized
// containers).
type Linux struct {
	base

	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: executor.New()}
}

// Collect returns logged-in session Info.
func (l *Linux) Collect(
	ctx context.Context,
) (any, error) {
	if l.Exec != nil {
		out, err := l.Exec.Execute(ctx,
			"loginctl", "--no-pager", "--no-legend", "--no-ask-password", "list-sessions")
		if err == nil {
			return &Info{LoggedIn: parseLoginctlSessions(out)}, nil
		}
	}
	ss, err := listSessions(ctx)
	if err != nil {
		return nil, err
	}
	return &Info{LoggedIn: ss}, nil
}
