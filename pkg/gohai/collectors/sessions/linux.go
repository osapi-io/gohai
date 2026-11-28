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

package sessions

import (
	"context"
	"errors"
	"fmt"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects logged-in sessions on Linux. Prefers
// `loginctl list-sessions` on systemd hosts (catches GDM/KDE
// graphical sessions, remote desktop, systemd-run sessions that
// never reach utmp). Falls back to reading utmp when
// loginctl is absent or errors (non-systemd hosts, minimized
// containers).
type Linux struct {
	base

	Exec source.Runner
	FS   avfs.VFS
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands(), FS: osfs.NewWithNoIdm()}
}

// Collect returns logged-in session Info: loginctl's sessions, or the utmp
// file's when loginctl does not answer. A host with neither has no
// sessions. A failed loginctl with no utmp file, or a utmp file that
// cannot be read, is an error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	var loginctlErr error
	if l.Exec != nil {
		out, err := l.Exec.Execute(ctx,
			"loginctl", "--no-pager", "--no-legend", "--no-ask-password", "list-sessions")
		if err == nil {
			return &Info{LoggedIn: parseLoginctlSessions(out)}, nil
		}
		if !collector.Absent(err) {
			loginctlErr = fmt.Errorf("loginctl list-sessions: %w", err)
		}
	}
	ss, err := readUtmp(l.FS, utmpPath, linuxUtmp)
	switch {
	case err == nil:
		return &Info{LoggedIn: ss}, nil
	case !collector.Absent(err):
		return nil, err
	case loginctlErr != nil && !exited(loginctlErr):
		// A loginctl that did not finish, such as a timeout, cannot say
		// there are no sessions.
		return nil, loginctlErr
	default:
		// A loginctl that exits with a status of its own has no systemd to
		// ask, as in a container. With no utmp either, there are no login
		// sessions.
		return &Info{LoggedIn: []Session{}}, nil
	}
}

// exited reports whether err is a command that ran to its end and exited
// with a status of its own, such as loginctl with no systemd to connect
// to. A command that a timeout or a signal killed has no exit status
// (ExitCode is -1), so it is not one.
func exited(
	err error,
) bool {
	var status interface{ ExitCode() int }

	return errors.As(err, &status) && status.ExitCode() >= 0
}
