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

// Package users enumerates system user and group accounts from
// /etc/passwd and /etc/group, plus the effective current user.
// Matches Ohai's passwd plugin on POSIX hosts. Logged-in session
// data has moved to the sessions collector.
package users

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// passwdPath / groupPath are the standard POSIX flat-file locations
// shared by Linux and macOS.
const (
	passwdPath = "/etc/passwd"
	groupPath  = "/etc/group"
	shadowPath = "/etc/shadow"
)

// New returns the users variant for the host OS. Linux reads
// /etc/passwd and /etc/group. Darwin reads the directory service, and
// the same files when dscl cannot list the accounts.
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

// collectPOSIX is the shared implementation used by both Linux and
// Darwin: read /etc/passwd + /etc/group from the injected VFS and
// look up the effective current user in the parsed passwd map.
func collectPOSIX(
	ctx context.Context,
	fs avfs.VFS,
	exec source.Runner,
) (*Info, error) {
	info, err := readPOSIX(fs)
	if err != nil {
		return nil, err
	}
	setCurrentUser(ctx, exec, info)

	return info, nil
}

// readPOSIX reads /etc/passwd, /etc/group and /etc/shadow from fs. A file
// that does not exist gives an empty map. An /etc/passwd or /etc/group
// that exists and does not read is an error. /etc/shadow is readable only
// by root, so a shadow that does not read leaves Locked unset.
func readPOSIX(
	fs avfs.VFS,
) (*Info, error) {
	info := &Info{
		Passwd: map[string]PasswdEntry{},
		Group:  map[string]GroupEntry{},
	}
	b, err := fs.ReadFile(passwdPath)
	switch {
	case err == nil:
		info.Passwd = parsePasswd(b)
	case !collector.Absent(err):
		return nil, fmt.Errorf("read %s: %w", passwdPath, err)
	}
	b, err = fs.ReadFile(groupPath)
	switch {
	case err == nil:
		info.Group = parseGroup(b)
	case !collector.Absent(err):
		return nil, fmt.Errorf("read %s: %w", groupPath, err)
	}
	if b, err := fs.ReadFile(shadowPath); err == nil {
		applyShadow(info.Passwd, b)
	}

	return info, nil
}

// setCurrentUser sets CurrentUser to the effective user: the name that
// `id -un` gives when info.Passwd has it, else the first name, in sorted
// order, of the users whose UID is the one `id -u` gives, so that two names
// on one UID give the same answer each run. It stays empty when exec is nil,
// when id fails, or when no user has the UID.
func setCurrentUser(
	ctx context.Context,
	exec source.Runner,
	info *Info,
) {
	if exec == nil {
		return
	}
	if out, err := exec.Execute(ctx, "id", "-un"); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			if _, ok := info.Passwd[name]; ok {
				info.CurrentUser = name

				return
			}
		}
	}
	out, err := exec.Execute(ctx, "id", "-u")
	if err != nil {
		return
	}
	euid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	names := make([]string, 0, len(info.Passwd))
	for name := range info.Passwd {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if info.Passwd[name].UID == euid {
			info.CurrentUser = name

			return
		}
	}
}

// Info holds the enumerated user and group databases plus current user.
type Info struct {
	Passwd      map[string]PasswdEntry `json:"passwd"`
	Group       map[string]GroupEntry  `json:"group"`
	CurrentUser string                 `json:"current_user,omitempty"`
}

// PasswdEntry mirrors the per-user fields Ohai surfaces under
// etc[:passwd][<username>].
type PasswdEntry struct {
	UID   int    `json:"uid"`
	GID   int    `json:"gid"`
	Dir   string `json:"dir,omitempty"`
	Shell string `json:"shell,omitempty"`
	GECOS string `json:"gecos,omitempty"`
	// Locked is whether the account cannot log in with a password: a
	// shadow password that starts with "!" or "*", as `passwd -S` reports
	// L, or DisabledUser on macOS. It is unset when the host does not let
	// the caller read it.
	Locked *bool `json:"locked,omitempty"`
}

// GroupEntry mirrors etc[:group][<groupname>].
type GroupEntry struct {
	GID     int      `json:"gid"`
	Members []string `json:"members,omitempty"`
}

// Collector is the public interface every users variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "users" }
func (base) Category() collector.Category { return collector.CategoryUsers }

// DefaultEnabled is false. Enumerating /etc/passwd is opt-in
// because consumers rarely need a full account dump.
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// parsePasswd turns the contents of /etc/passwd into a name→entry map.
// Duplicate usernames keep the first occurrence (matches Ohai's
// Etc.passwd iteration, which the C library de-dupes the same way).
// Malformed lines (fewer than 7 colon-separated fields) are skipped.
func parsePasswd(
	content []byte,
) map[string]PasswdEntry {
	out := make(map[string]PasswdEntry)
	// strings.Split, not a bufio.Scanner, which stops with no error at a
	// line over 64 KiB, such as a group of an LDAP-synced host with
	// thousands of members.
	for _, line := range strings.Split(string(content), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}
		name := fields[0]
		if _, exists := out[name]; exists {
			continue
		}
		// A line whose UID or GID is not a number, such as NIS's
		// "+::::::", names no account.
		uid, uidErr := strconv.Atoi(fields[2])
		gid, gidErr := strconv.Atoi(fields[3])
		if uidErr != nil || gidErr != nil {
			continue
		}
		out[name] = PasswdEntry{
			UID:   uid,
			GID:   gid,
			GECOS: fields[4],
			Dir:   fields[5],
			Shell: fields[6],
		}
	}
	return out
}

// parseGroup turns the contents of /etc/group into a name→entry map.
// /etc/group format: name:passwd:gid:members(comma-separated).
// Malformed lines (fewer than 4 fields) are skipped.
func parseGroup(
	content []byte,
) map[string]GroupEntry {
	out := make(map[string]GroupEntry)
	// strings.Split, not a bufio.Scanner, which stops with no error at a
	// line over 64 KiB, such as a group of an LDAP-synced host with
	// thousands of members.
	for _, line := range strings.Split(string(content), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		// A bad or empty GID would read as 0, the root group, so the entry
		// is skipped, as parsePasswd skips a bad UID or GID. The first entry
		// for a name wins, as in parsePasswd.
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		if _, exists := out[fields[0]]; exists {
			continue
		}
		var members []string
		if m := strings.TrimSpace(fields[3]); m != "" {
			for _, part := range strings.Split(m, ",") {
				if p := strings.TrimSpace(part); p != "" {
					members = append(members, p)
				}
			}
		}
		out[fields[0]] = GroupEntry{GID: gid, Members: members}
	}
	return out
}

// applyShadow sets Locked from the password field of each /etc/shadow
// line, "name:password:...".
func applyShadow(
	users map[string]PasswdEntry,
	b []byte,
) {
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 2 {
			continue
		}
		e, ok := users[f[0]]
		if !ok {
			continue
		}
		locked := strings.HasPrefix(f[1], "!") || strings.HasPrefix(f[1], "*")
		e.Locked = &locked
		users[f[0]] = e
	}
}

// applyDisabled sets Locked from `dscl . -list /Users
// AuthenticationAuthority`: a user whose authority has DisabledUser is
// locked, and the others are not.
func applyDisabled(
	users map[string]PasswdEntry,
	out []byte,
) {
	for _, line := range strings.Split(string(out), "\n") {
		name, authority, _ := strings.Cut(strings.TrimSpace(line), " ")
		e, ok := users[name]
		if !ok {
			continue
		}
		// An account with no AuthenticationAuthority, as many _service
		// accounts have, does not say whether it is locked, so Locked stays
		// unset.
		if strings.TrimSpace(authority) == "" {
			continue
		}
		locked := strings.Contains(authority, "DisabledUser")
		e.Locked = &locked
		users[name] = e
	}
}
