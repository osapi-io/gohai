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
)

// Darwin collects SSH host key fingerprints on macOS. macOS ships
// OpenSSH and stores host keys in the same /etc/ssh/ path as Linux.
// FS is the virtual filesystem — production uses the real OS filesystem;
// tests inject an avfs memfs with canned key files.
type Darwin struct {
	base

	FS avfs.VFS
}

// NewDarwin returns a Darwin variant wired to the real OS filesystem.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm()}
}

// Collect reads SSH host public keys and returns their fingerprints.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	keys, err := collectHostKeys(d.FS)
	if err != nil {
		return nil, err
	}
	return &Info{Keys: keys}, nil
}
