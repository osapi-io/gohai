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

package users_test

import (
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*users.Linux)(nil)
	_ collector.Collector = (*users.Darwin)(nil)
)

const passwdFixture = `# comment
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
john:x:1000:1000:John Doe,,,:/home/john:/bin/zsh
dup:x:2:2:first:/home/a:/bin/sh
dup:x:99:99:second:/home/b:/bin/sh
malformed-line-missing-fields
`

const groupFixture = `# comment
root:x:0:
wheel:x:10:john,root
staff:x:20:
users:x:1000:john,daemon
malformed:x:`

type UsersPublicTestSuite struct {
	suite.Suite
}

func TestUsersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UsersPublicTestSuite))
}

func (s *UsersPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(users.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c users.Collector) {
				_, ok := c.(*users.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c users.Collector) {
				_, ok := c.(*users.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c users.Collector) {
				_, ok := c.(*users.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c users.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := users.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = users.New()
			}
			s.Equal("users", c.Name())
			s.Equal("users", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *UsersPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		files        map[string]string
		failRead     []string // paths that exist and do not read
		euid         int
		id           string // output of `id -u`; "" uses euid
		idName       string // output of `id -un`; "" is an id -un that fails
		idErr        error
		nilExec      bool
		dscl         string            // dscl output; "" is a dscl that fails
		dsclLists    map[string]string // output of other dscl lines
		dsclErrs     map[string]error  // error of other dscl lines
		validateFunc func(any, error)
	}{
		{
			name:    "linux: full fixture parses passwd + group + current_user",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": passwdFixture,
				"/etc/group":  groupFixture,
			},
			euid: 1000,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Require().Len(info.Passwd, 4)
				root := info.Passwd["root"]
				s.Equal(0, root.UID)
				s.Equal("/bin/bash", root.Shell)
				s.Equal("root", root.GECOS)
				john := info.Passwd["john"]
				s.Equal(1000, john.UID)
				s.Equal("John Doe,,,", john.GECOS)
				s.Equal("/home/john", john.Dir)
				// Duplicate "dup" entry kept first.
				dup := info.Passwd["dup"]
				s.Equal(2, dup.UID)
				s.Equal("/home/a", dup.Dir)

				wheel := info.Group["wheel"]
				s.Equal(10, wheel.GID)
				s.Equal([]string{"john", "root"}, wheel.Members)
				staff := info.Group["staff"]
				s.Equal(20, staff.GID)
				s.Empty(staff.Members)

				s.Equal("john", info.CurrentUser)
			},
		},
		{
			name:    "linux: missing files produces empty maps, no error",
			variant: "linux",
			files:   map[string]string{},
			euid:    0,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Empty(info.Passwd)
				s.Empty(info.Group)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:     "linux: an /etc/passwd that does not read is an error",
			variant:  "linux",
			failRead: []string{"/etc/passwd"},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/passwd")
				s.Exactly(nil, got)
			},
		},
		{
			name:     "linux: an /etc/group that does not read is an error",
			variant:  "linux",
			files:    map[string]string{"/etc/passwd": "root:x:0:0::/root:/bin/sh\n"},
			failRead: []string{"/etc/group"},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/group")
				s.Exactly(nil, got)
			},
		},
		{
			name:     "linux: an /etc/shadow that does not read leaves Locked unset",
			variant:  "linux",
			files:    map[string]string{"/etc/passwd": "root:x:0:0::/root:/bin/sh\n"},
			failRead: []string{"/etc/shadow"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Nil(info.Passwd["root"].Locked)
			},
		},
		{
			name:     "darwin: an /etc/passwd that does not read is an error",
			variant:  "darwin",
			nilExec:  true,
			failRead: []string{"/etc/passwd"},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/passwd")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: current user lookup misses when euid absent",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/root:/bin/bash\n",
			},
			euid: 9999,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Equal("", info.CurrentUser)
			},
		},
		{
			name:    "linux: two names on one UID: the first in sorted order",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "toor:x:0:0::/root:/bin/sh\nroot:x:0:0::/root:/bin/sh\n",
			},
			euid: 0,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Equal("root", info.CurrentUser)
			},
		},
		{
			name:    "linux: id -un names a user: that user",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0::/root:/bin/sh\ntoor:x:0:0::/root:/bin/sh\n",
			},
			euid:   0,
			idName: "toor\n",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Equal("toor", info.CurrentUser)
			},
		},
		{
			name:    "linux: id -un names no user in passwd: the UID decides",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0::/root:/bin/sh\n",
			},
			euid:   0,
			idName: "ldapuser\n",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Equal("root", info.CurrentUser)
			},
		},
		{
			name:    "linux: a NIS line with no UID names no account",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "+::::::\njohn:x:1000:1000::/home/john:/bin/sh\nbad:x:x:0::/:/bin/sh\n",
			},
			euid: 0,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Len(info.Passwd, 1)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:    "linux: a group line over 64 KiB keeps that group and the ones after it",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/root:/bin/sh\n",
				"/etc/group": "big:x:100:" + strings.TrimSuffix(
					strings.Repeat("member,", 12000),
					",",
				) +
					"\nafter:x:101:root\n",
			},
			euid: 0,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Len(info.Group["big"].Members, 12000)
				s.Equal([]string{"root"}, info.Group["after"].Members)
			},
		},
		{
			name:    "linux: a passwd line over 64 KiB keeps the users after it",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "long:x:5:5:" + strings.Repeat(
					"g",
					70000,
				) + ":/home/long:/bin/sh\n" +
					"root:x:0:0:root:/root:/bin/sh\n",
			},
			euid: 0,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Contains(info.Passwd, "long")
				s.Equal("root", info.CurrentUser)
			},
		},
		{
			name:    "linux: id fails: no current user",
			variant: "linux",
			files:   map[string]string{"/etc/passwd": "root:x:0:0:root:/root:/bin/sh\n"},
			idErr:   errors.New("exec id: not found"),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:    "linux: nil Exec: no current user",
			variant: "linux",
			files:   map[string]string{"/etc/passwd": "root:x:0:0:root:/root:/bin/sh\n"},
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:    "linux: id output not a number: no current user",
			variant: "linux",
			files:   map[string]string{"/etc/passwd": "root:x:0:0:root:/root:/bin/sh\n"},
			id:      "root\n",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:    "linux: locked from /etc/shadow",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/root:/bin/sh\ndaemon:x:1:1::/:/usr/sbin/nologin\njohn:x:1000:1000::/home/john:/bin/bash\nnoshadow:x:2:2::/:/bin/sh\n",
				"/etc/shadow": "root:!$6$abc:19000::::::\ndaemon:*:19000::::::\njohn:$6$xyz:19000::::::\nghost:!:1::::::\nshort\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Require().NotNil(info.Passwd["root"].Locked)
				// A shadow password that starts with ! or * is locked: root
				// has !, and daemon has *.
				s.True(*info.Passwd["root"].Locked)
				s.True(*info.Passwd["daemon"].Locked)
				s.False(*info.Passwd["john"].Locked)
				// No shadow line.
				s.Nil(info.Passwd["noshadow"].Locked)
			},
		},
		{
			name:    "darwin: locked from dscl's DisabledUser",
			variant: "darwin",
			files: map[string]string{
				"/etc/passwd": "root:*:0:0::/var/root:/bin/sh\nguest:*:201:201::/Users/Guest:/bin/bash\n",
			},
			dscl: "root ;ShadowHash;\nguest ;DisabledUser; ;ShadowHash;\nnobody-in-passwd ;ShadowHash;\n",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.False(*info.Passwd["root"].Locked)
				s.True(*info.Passwd["guest"].Locked)
			},
		},
		{
			// Most macOS users are in the directory service, not in
			// /etc/passwd or /etc/group.
			name:    "darwin: users and groups from the directory service",
			variant: "darwin",
			files: map[string]string{
				"/etc/passwd": "root:*:0:0:System Administrator:/var/root:/bin/sh\n",
				"/etc/group":  "wheel:*:0:root\n",
			},
			euid: 501,
			dscl: "root ;ShadowHash;\njohn ;ShadowHash;\nguest ;DisabledUser;\n",
			dsclLists: map[string]string{
				"dscl . -list /Users UniqueID": "root      0\n" +
					"john      501\n" +
					"guest     201\n" +
					"bad       x\n" +
					"noid\n",
				"dscl . -list /Users PrimaryGroupID":   "root 0\njohn 20\nguest 201\n",
				"dscl . -list /Users NFSHomeDirectory": "root /var/root /private/var/root\njohn /Users/john\n",
				"dscl . -list /Users UserShell":        "root /bin/sh\njohn /bin/zsh\n",
				"dscl . -list /Users RealName":         "root      System Administrator\njohn      John Doe\nguest\n",
				"dscl . -list /Groups PrimaryGroupID":  "admin  80\nstaff  20\nbad    x\n",
				"dscl . -list /Groups GroupMembership": "admin    root john\nstaff\nghost  root\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Require().Len(info.Passwd, 3)
				john := info.Passwd["john"]
				s.Equal(501, john.UID)
				s.Equal(20, john.GID)
				s.Equal("/Users/john", john.Dir)
				s.Equal("/bin/zsh", john.Shell)
				s.Equal("John Doe", john.GECOS)
				// The first home.
				s.Equal("/var/root", info.Passwd["root"].Dir)
				s.Equal(201, info.Passwd["guest"].GID)
				s.Empty(info.Passwd["guest"].Dir)
				s.True(*info.Passwd["guest"].Locked)
				s.False(*info.Passwd["john"].Locked)
				s.Equal("john", info.CurrentUser)
				s.Require().Len(info.Group, 2)
				s.Equal(
					users.GroupEntry{GID: 80, Members: []string{"root", "john"}},
					info.Group["admin"],
				)
				s.Equal(users.GroupEntry{GID: 20}, info.Group["staff"])
			},
		},
		{
			name:    "darwin: a dscl that fails to list the users is an error",
			variant: "darwin",
			dscl:    "root ;ShadowHash;\n",
			dsclErrs: map[string]error{
				"dscl . -list /Users UniqueID": errors.New("exit status 1"),
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "dscl . -list /Users UniqueID: exit status 1")
			},
		},
		{
			name:    "darwin: a dscl that fails to list a user attribute is an error",
			variant: "darwin",
			dscl:    "root ;ShadowHash;\n",
			dsclLists: map[string]string{
				"dscl . -list /Users UniqueID": "root 0\n",
			},
			dsclErrs: map[string]error{
				"dscl . -list /Users RealName": errors.New("exit status 1"),
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "dscl . -list /Users RealName: exit status 1")
			},
		},
		{
			name:    "darwin: a dscl that fails to list the groups is an error",
			variant: "darwin",
			dscl:    "root ;ShadowHash;\n",
			dsclErrs: map[string]error{
				"dscl . -list /Groups PrimaryGroupID": errors.New("exit status 1"),
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "dscl . -list /Groups PrimaryGroupID: exit status 1")
			},
		},
		{
			name:    "darwin: a dscl that fails to list the group members is an error",
			variant: "darwin",
			dscl:    "root ;ShadowHash;\n",
			dsclLists: map[string]string{
				"dscl . -list /Groups PrimaryGroupID": "admin 80\n",
			},
			dsclErrs: map[string]error{
				"dscl . -list /Groups GroupMembership": errors.New("exit status 1"),
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "dscl . -list /Groups GroupMembership: exit status 1")
			},
		},
		{
			name:    "darwin: a dscl that fails to list AuthenticationAuthority is an error",
			variant: "darwin",
			dscl:    "unused",
			dsclErrs: map[string]error{
				"dscl . -list /Users AuthenticationAuthority": errors.New("exit status 1"),
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "dscl . -list /Users AuthenticationAuthority: exit status 1")
			},
		},
		{
			name:    "darwin: groups from /etc/group when dscl cannot list them",
			variant: "darwin",
			files: map[string]string{
				"/etc/group": "wheel:*:0:root\n",
			},
			dsclLists: map[string]string{
				"dscl . -list /Users UniqueID": "root 0\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Equal(0, info.Passwd["root"].UID)
				s.Equal([]string{"root"}, info.Group["wheel"].Members)
			},
		},
		{
			name:    "darwin: nil Exec",
			variant: "darwin",
			files:   map[string]string{"/etc/passwd": "root:*:0:0::/var/root:/bin/sh\n"},
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Nil(info.Passwd["root"].Locked)
			},
		},
		{
			name:    "darwin: same POSIX parsing",
			variant: "darwin",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/var/root:/bin/zsh\njohn:x:501:20:John:/Users/john:/bin/zsh\n",
				"/etc/group":  "admin:x:80:root,john\n",
			},
			euid: 501,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*users.Info)
				s.Require().True(ok)
				s.Require().Len(info.Passwd, 2)
				s.Equal(501, info.Passwd["john"].UID)
				s.Equal([]string{"root", "john"}, info.Group["admin"].Members)
				s.Equal("john", info.CurrentUser)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			id := tt.id
			if id == "" {
				id = strconv.Itoa(tt.euid) + "\n"
			}
			r := sourcetest.Commands{Out: map[string]string{"id -u": id}, Err: map[string]error{}}
			if tt.idErr != nil {
				r.Err["id -u"] = tt.idErr
			}
			if tt.idName != "" {
				r.Out["id -un"] = tt.idName
			} else {
				r.Err["id -un"] = errors.New("exec id: not found")
			}
			const dscl = "dscl . -list /Users AuthenticationAuthority"
			if tt.dscl != "" {
				r.Out[dscl] = tt.dscl
			} else {
				r.Err[dscl] = fmt.Errorf("exec dscl: %w", osexec.ErrNotFound)
			}
			for line, out := range tt.dsclLists {
				r.Out[line] = out
			}
			for line, err := range tt.dsclErrs {
				r.Err[line] = err
			}
			var exec source.Runner = r.Mock(s.T())
			if tt.nilExec {
				exec = nil
			}
			var c users.Collector
			switch tt.variant {
			case "linux":
				c = &users.Linux{
					FS: sourcetest.FailRead(
						s.T(),
						sourcetest.FS(s.T(), tt.files),
						tt.failRead...),
					Exec: exec,
				}
			case "darwin":
				c = &users.Darwin{
					FS: sourcetest.FailRead(
						s.T(),
						sourcetest.FS(s.T(), tt.files),
						tt.failRead...),
					Exec: exec,
				}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
