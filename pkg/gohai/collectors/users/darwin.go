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
	"fmt"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin enumerates users and groups from the local node of the directory
// service with dscl. On macOS, /etc/passwd and /etc/group have only the
// system accounts, not the accounts that a person made. On a host with no
// dscl, or a directory service with no records, the flat files are used. A
// dscl that fails is an error.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the real OS filesystem.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect reads the users and groups of the directory service, or of
// /etc/passwd and /etc/group, and resolves the effective current user.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := readPOSIX(d.FS)
	if d.Exec != nil {
		if err := d.applyDscl(ctx, info); err != nil {
			return nil, err
		}
	}
	setCurrentUser(ctx, d.Exec, info)

	return info, nil
}

// applyDscl puts the users and groups of the directory service in info,
// in place of the flat files' when dscl lists any, and marks the disabled
// users. A list that dscl does not have leaves its part of info as it is.
// Its error is the first dscl run that failed.
func (d *Darwin) applyDscl(
	ctx context.Context,
	info *Info,
) error {
	p, err := d.dsclPasswd(ctx)
	switch {
	case err == nil && p != nil:
		info.Passwd = p
	case err != nil && !collector.Absent(err):
		return err
	}
	g, err := d.dsclGroup(ctx)
	switch {
	case err == nil && g != nil:
		info.Group = g
	case err != nil && !collector.Absent(err):
		return err
	}
	out, err := d.list(ctx, "/Users", "AuthenticationAuthority")
	switch {
	case err == nil:
		applyDisabled(info.Passwd, out)
	case !collector.Absent(err):
		return err
	}

	return nil
}

// list runs `dscl . -list <path> <attr>`: one "name value..." line for
// each record that has the attribute. A user that is not root can run it.
func (d *Darwin) list(
	ctx context.Context,
	path string,
	attr string,
) ([]byte, error) {
	out, err := d.Exec.Execute(ctx, "dscl", ".", "-list", path, attr)
	if err != nil {
		return nil, fmt.Errorf("dscl . -list %s %s: %w", path, attr, err)
	}

	return out, nil
}

// values is the value of attr for each record of path, by record name. A
// record with many values, such as two homes, keeps the first, unless all
// is true, as for a RealName with spaces. Its error is the run's.
func (d *Darwin) values(
	ctx context.Context,
	path string,
	attr string,
	all bool,
) (map[string]string, error) {
	out := map[string]string{}
	b, err := d.list(ctx, path, attr)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if all {
			out[f[0]] = strings.Join(f[1:], " ")
		} else {
			out[f[0]] = f[1]
		}
	}

	return out, nil
}

// dsclPasswd is each user that has a UniqueID, with its group, home,
// shell and real name. It is nil when the directory service lists no
// users, and its error is the first dscl run that failed.
func (d *Darwin) dsclPasswd(
	ctx context.Context,
) (map[string]PasswdEntry, error) {
	uids, err := d.values(ctx, "/Users", "UniqueID", false)
	if err != nil || len(uids) == 0 {
		return nil, err
	}
	attrs := map[string]map[string]string{}
	for _, a := range []struct {
		name string
		all  bool
	}{
		{"PrimaryGroupID", false},
		{"NFSHomeDirectory", false},
		{"UserShell", false},
		{"RealName", true},
	} {
		v, err := d.values(ctx, "/Users", a.name, a.all)
		if err != nil && !collector.Absent(err) {
			return nil, err
		}
		attrs[a.name] = v
	}
	gids, homes := attrs["PrimaryGroupID"], attrs["NFSHomeDirectory"]
	shells, names := attrs["UserShell"], attrs["RealName"]
	out := map[string]PasswdEntry{}
	for name, v := range uids {
		uid, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		// A user with no PrimaryGroupID, or a bad one, would read as gid
		// 0, the wheel group, so the user is skipped.
		gid, err := strconv.Atoi(gids[name])
		if err != nil {
			continue
		}
		out[name] = PasswdEntry{
			UID:   uid,
			GID:   gid,
			Dir:   homes[name],
			Shell: shells[name],
			GECOS: names[name],
		}
	}

	return out, nil
}

// dsclGroup is each group that has a PrimaryGroupID, with its members. It
// is nil when the directory service lists no groups, and its error is the
// first dscl run that failed.
func (d *Darwin) dsclGroup(
	ctx context.Context,
) (map[string]GroupEntry, error) {
	gids, err := d.values(ctx, "/Groups", "PrimaryGroupID", false)
	if err != nil || len(gids) == 0 {
		return nil, err
	}
	members, err := d.values(ctx, "/Groups", "GroupMembership", true)
	if err != nil && !collector.Absent(err) {
		return nil, err
	}
	out := map[string]GroupEntry{}
	for name, v := range gids {
		gid, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		e := GroupEntry{GID: gid}
		if m := members[name]; m != "" {
			e.Members = strings.Fields(m)
		}
		out[name] = e
	}

	return out, nil
}
