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
	"errors"
	"fmt"
	"io"
	"os"
)

// Darwin collects the shell list on macOS hosts. /etc/shells on macOS
// ships with a small curated set (/bin/bash, /bin/zsh, /bin/sh, etc.)
// and is managed by the OS installer. Embeds base for Name /
// DefaultEnabled / Dependencies. OpenFn is injected so tests can stub
// the file read.
type Darwin struct {
	base

	OpenFn func(string) (io.ReadCloser, error)
}

// NewDarwin returns a Darwin variant wired to the package-level openFile
// helper. Named helper (not inline closure) keeps the factory a plain
// assignment.
func NewDarwin() *Darwin {
	return &Darwin{OpenFn: openFile}
}

// Collect reads /etc/shells and returns the list of valid login shells.
// Behavior matches Linux: missing file soft-misses to an empty list.
// Kept as a separate implementation rather than a shared helper so
// darwin can diverge freely if Apple ever changes /etc/shells handling.
func (d *Darwin) Collect(
	_ context.Context,
) (any, error) {
	rc, err := d.OpenFn(etcShellsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Info{Paths: []string{}}, nil
		}
		return nil, fmt.Errorf("open %s: %w", etcShellsPath, err)
	}
	defer func() { _ = rc.Close() }()
	return parseShells(rc)
}
