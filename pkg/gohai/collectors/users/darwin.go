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
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin enumerates users and groups from the local node of the directory
// service with dscl. On macOS, /etc/passwd and /etc/group have only the
// system accounts, not the accounts that a person made. When dscl cannot
// list them, the flat files are used.
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
		if p, ok := d.dsclPasswd(ctx); ok {
			info.Passwd = p
		}
		if g, ok := d.dsclGroup(ctx); ok {
			info.Group = g
		}
		if out, err := d.list(ctx, "/Users", "AuthenticationAuthority"); err == nil {
			applyDisabled(info.Passwd, out)
		}
	}
	setCurrentUser(ctx, d.Exec, info)

	return info, nil
}

// list runs `dscl . -list <path> <attr>`: one "name value..." line for
// each record that has the attribute. A user that is not root can run it.
func (d *Darwin) list(
	ctx context.Context,
	path string,
	attr string,
) ([]byte, error) {
	return d.Exec.Execute(ctx, "dscl", ".", "-list", path, attr)
}

// values is the value of attr for each record of path, by record name. A
// record with many values, such as two homes, keeps the first, unless all
// is true, as for a RealName with spaces. A failed run gives no values.
func (d *Darwin) values(
	ctx context.Context,
	path string,
	attr string,
	all bool,
) map[string]string {
	out := map[string]string{}
	b, err := d.list(ctx, path, attr)
	if err != nil {
		return out
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

	return out
}

// dsclPasswd is each user that has a UniqueID, with its group, home,
// shell and real name. It is false when dscl cannot list the users.
func (d *Darwin) dsclPasswd(
	ctx context.Context,
) (map[string]PasswdEntry, bool) {
	uids := d.values(ctx, "/Users", "UniqueID", false)
	if len(uids) == 0 {
		return nil, false
	}
	gids := d.values(ctx, "/Users", "PrimaryGroupID", false)
	homes := d.values(ctx, "/Users", "NFSHomeDirectory", false)
	shells := d.values(ctx, "/Users", "UserShell", false)
	names := d.values(ctx, "/Users", "RealName", true)
	out := map[string]PasswdEntry{}
	for name, v := range uids {
		uid, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		gid, _ := strconv.Atoi(gids[name])
		out[name] = PasswdEntry{
			UID:   uid,
			GID:   gid,
			Dir:   homes[name],
			Shell: shells[name],
			GECOS: names[name],
		}
	}

	return out, true
}

// dsclGroup is each group that has a PrimaryGroupID, with its members. It
// is false when dscl cannot list the groups.
func (d *Darwin) dsclGroup(
	ctx context.Context,
) (map[string]GroupEntry, bool) {
	gids := d.values(ctx, "/Groups", "PrimaryGroupID", false)
	if len(gids) == 0 {
		return nil, false
	}
	members := d.values(ctx, "/Groups", "GroupMembership", true)
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

	return out, true
}
