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

// Linux collects the shell list on Linux hosts. Embeds base for the
// static Name / DefaultEnabled / Dependencies methods. FS is the
// virtual filesystem the collector reads from — production is the real
// OS FS via avfs/osfs; tests inject an avfs memfs with canned content.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect reads /etc/shells and returns the list of valid login shells.
// A missing file soft-misses to an empty list — distroless/scratch
// containers legitimately lack /etc/shells.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := l.FS.ReadFile(etcShellsPath)
	if err != nil {
		return wrapReadError(err)
	}
	return parseShells(b), nil
}
