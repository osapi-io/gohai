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

// Darwin collects SSH host key fingerprints on macOS. macOS ships
// OpenSSH and stores host keys in the same /etc/ssh/ path as Linux.
// /etc/passwd on macOS has only the system users, and the other users
// are in the directory service, so Exec runs dscl to find each home.
// FS is the virtual filesystem — production uses the real OS filesystem;
// tests inject an avfs memfs with canned key files.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the real OS filesystem.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect reads SSH host public keys and returns their fingerprints,
// and the authorized keys of each user of the directory service. When
// dscl does not run, the users of /etc/passwd are used.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	return &Info{
		Keys:           collectHostKeys(d.FS),
		AuthorizedKeys: collectAuthorizedKeys(d.FS, d.homes(ctx), keyFiles(ctx, d.Exec)),
	}, nil
}

// homes is the home of each user from `dscl . -list /Users
// NFSHomeDirectory`, which a user that is not root can run. It is the
// users of /etc/passwd when there is no Exec or dscl fails.
func (d *Darwin) homes(
	ctx context.Context,
) []home {
	if d.Exec != nil {
		out, err := d.Exec.Execute(ctx, "dscl", ".", "-list", "/Users", "NFSHomeDirectory")
		if err == nil {
			return dsclHomes(out)
		}
	}

	return passwdHomes(d.FS)
}
