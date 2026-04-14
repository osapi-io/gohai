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

package shells

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin collects the shell list on macOS hosts. /etc/shells on macOS
// ships with a small curated set (/bin/bash, /bin/zsh, /bin/sh, etc.)
// and is managed by the OS installer. Embeds base for Name /
// DefaultEnabled / Dependencies. FS is the virtual filesystem — real
// OS on production, avfs memfs in tests.
type Darwin struct {
	base

	FS avfs.VFS
}

// NewDarwin returns a Darwin variant wired to the real OS filesystem.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm()}
}

// Collect reads /etc/shells and returns the list of valid login shells.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := d.FS.ReadFile(etcShellsPath)
	if err != nil {
		return wrapReadError(err)
	}
	return parseShells(b), nil
}
