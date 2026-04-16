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
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// passwdPath / groupPath are the standard POSIX flat-file locations
// shared by Linux and macOS.
const (
	passwdPath = "/etc/passwd"
	groupPath  = "/etc/group"
)

// geteuidFn is the injection seam for os.Geteuid — tests swap it to
// pin a deterministic current user.
var geteuidFn = os.Geteuid

// New returns the users variant for the host OS. Both Linux and
// Darwin read the same /etc/passwd and /etc/group files — separate
// structs preserve the per-OS dispatch pattern for consistency.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// collectPOSIX is the shared implementation used by both Linux and
// Darwin: read /etc/passwd + /etc/group from the injected VFS and
// look up the effective current user in the parsed passwd map.
func collectPOSIX(
	fs avfs.VFS,
) (*Info, error) {
	info := &Info{
		Passwd: map[string]PasswdEntry{},
		Group:  map[string]GroupEntry{},
	}
	if b, err := fs.ReadFile(passwdPath); err == nil {
		info.Passwd = parsePasswd(b)
	}
	if b, err := fs.ReadFile(groupPath); err == nil {
		info.Group = parseGroup(b)
	}
	euid := geteuidFn()
	for name, entry := range info.Passwd {
		if entry.UID == euid {
			info.CurrentUser = name
			break
		}
	}
	return info, nil
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

func (base) Name() string     { return "users" }
func (base) Category() string { return collector.CategoryUsers }

// DefaultEnabled is false — enumerating /etc/passwd is opt-in
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
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
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
		uid, _ := strconv.Atoi(fields[2])
		gid, _ := strconv.Atoi(fields[3])
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
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		gid, _ := strconv.Atoi(fields[2])
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
