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

// Package users collects currently logged-in user sessions (utmp/who
// output). NOTE: despite the name, this collector only covers logged-in
// sessions — it does NOT enumerate /etc/passwd. Ohai splits this into
// two plugins (passwd.rb for account enumeration, linux/sessions.rb
// for loginctl). A future gohai `passwd` collector will fill the
// enumeration gap.
package users

import (
	"context"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the set of logged-in sessions.
type Info struct {
	LoggedIn []Session `json:"logged_in"`
}

// Session represents one logged-in user session.
type Session struct {
	User     string `json:"user"`
	Terminal string `json:"terminal,omitempty"`
	Host     string `json:"host,omitempty"`
	Started  uint64 `json:"started,omitempty"` // unix timestamp
}

// Collector is the public interface every users variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "users" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the users variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// usersFn is the injection seam for gopsutil's host.UsersWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// via SetUsersFn.
var usersFn = host.UsersWithContext

// listSessions is the production bridge to gopsutil (which reads
// utmp on Linux / utmpx on macOS).
func listSessions(
	ctx context.Context,
) ([]Session, error) {
	us, err := usersFn(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(us))
	for _, u := range us {
		out = append(out, Session{
			User:     u.User,
			Terminal: u.Terminal,
			Host:     u.Host,
			Started:  uint64(u.Started),
		})
	}
	return out, nil
}
