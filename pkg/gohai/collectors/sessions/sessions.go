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

// Package sessions reports currently logged-in user sessions. On systemd
// hosts we prefer `loginctl list-sessions` — the same data loginctl
// itself shows — which surfaces graphical (GDM/KDE), remote-desktop,
// and `systemd-run` sessions that never reach utmp. On non-systemd
// hosts and macOS we read utmp / utmpx, which is what `who` / `w`
// print.
//
// This collector used to live under `users` but was split out: `users`
// now matches Ohai's passwd plugin (enumerate /etc/passwd + /etc/group
// + current_user), and logged-in sessions moved here.
package sessions

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds the set of logged-in sessions.
type Info struct {
	LoggedIn []Session `json:"logged_in"`
}

// Session represents one logged-in user session.
type Session struct {
	User      string `json:"user"`
	Terminal  string `json:"terminal,omitempty"`
	Host      string `json:"host,omitempty"`
	Started   uint64 `json:"started,omitempty"`    // unix timestamp (utmp path only)
	SessionID string `json:"session_id,omitempty"` // systemd session id (loginctl path only)
	UID       string `json:"uid,omitempty"`        // numeric UID (loginctl path only)
	Seat      string `json:"seat,omitempty"`       // systemd seat (loginctl path only)
}

// Collector is the public interface every sessions variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "sessions" }
func (base) Category() collector.Category { return collector.CategoryUsers }

// DefaultEnabled is false: sessions are niche and change per-invocation.
// Opt in via --collector.sessions or WithEnabled("sessions").
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the sessions variant for the host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// The records of utmp (Linux) and utmpx (macOS), as glibc and Darwin lay
// them out, little-endian. A record of type userProcess is a logged-in
// user.
const (
	utmpPath    = "/var/run/utmp"
	utmpxPath   = "/var/run/utmpx"
	userProcess = 7
)

// utmpLayout is where each field of one record is.
type utmpLayout struct {
	size, typ, line, lineLen, user, userLen, host, hostLen, sec int
	// header is a record at the start of the file that is not a session.
	header bool
}

var (
	// linuxUtmp is glibc's struct utmp: 384 bytes.
	linuxUtmp = utmpLayout{
		size: 384, typ: 0, line: 8, lineLen: 32, user: 44, userLen: 32,
		host: 76, hostLen: 256, sec: 340,
	}
	// darwinUtmpx is Darwin's struct utmpx: 628 bytes, after a header
	// record.
	darwinUtmpx = utmpLayout{
		size: 628, user: 0, userLen: 256, line: 260, lineLen: 32, typ: 296,
		sec: 300, host: 308, hostLen: 256, header: true,
	}
)

// readUtmp is each logged-in user of the utmp file at path, read with
// layout l, as who prints them.
func readUtmp(
	fs avfs.VFS,
	path string,
	l utmpLayout,
) ([]Session, error) {
	buf, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if l.header && len(buf) >= l.size {
		buf = buf[l.size:]
	}
	out := []Session{}
	for ; len(buf) >= l.size; buf = buf[l.size:] {
		r := buf[:l.size]
		if binary.LittleEndian.Uint16(r[l.typ:]) != userProcess {
			continue
		}
		out = append(out, Session{
			User:     cString(r[l.user : l.user+l.userLen]),
			Terminal: cString(r[l.line : l.line+l.lineLen]),
			Host:     cString(r[l.host : l.host+l.hostLen]),
			Started:  uint64(binary.LittleEndian.Uint32(r[l.sec:])),
		})
	}

	return out, nil
}

// cString is b up to its first NUL.
func cString(
	b []byte,
) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}

	return string(b)
}

// parseLoginctlSessions parses `loginctl --no-pager --no-legend
// --no-ask-password list-sessions` output. Each non-empty line is
// whitespace-split into `session uid user [seat]`. Lines with fewer
// than 3 fields are skipped (defensive — Ohai assumes the format).
func parseLoginctlSessions(
	raw []byte,
) []Session {
	var out []Session
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		s := Session{
			SessionID: fields[0],
			UID:       fields[1],
			User:      fields[2],
		}
		if len(fields) >= 4 {
			s.Seat = fields[3]
		}
		out = append(out, s)
	}
	return out
}
