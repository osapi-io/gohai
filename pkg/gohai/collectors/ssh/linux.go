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

package ssh

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects SSH host key fingerprints on Linux. Reads the standard
// host public key files from /etc/ssh/. FS is the virtual filesystem —
// production uses the real OS filesystem; tests inject an avfs memfs
// with canned key files. Exec runs `getent passwd`, for the users of
// LDAP or sssd, and `sshd -T`, for the AuthorizedKeysFile.
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// programs.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect reads SSH host public keys and returns their fingerprints,
// and the authorized keys of each user.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	return &Info{
		Keys:           collectHostKeys(l.FS),
		AuthorizedKeys: collectAuthorizedKeys(l.FS, l.homes(ctx), keyFiles(ctx, l.Exec)),
	}, nil
}

// homes is the home of each user of `getent passwd`, which has the users
// of LDAP and sssd when their service lists them, or of /etc/passwd when
// there is no Exec or getent fails.
func (l *Linux) homes(
	ctx context.Context,
) []home {
	if l.Exec != nil {
		if out, err := l.Exec.Execute(ctx, "getent", "passwd"); err == nil {
			if homes := parsePasswd(out); len(homes) > 0 {
				return homes
			}
		}
	}

	return passwdHomes(l.FS)
}
