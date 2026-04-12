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

// Linux collects the shell list on Linux hosts. Embeds base for the
// static Name / DefaultEnabled / Dependencies methods. OpenFn is
// injected so tests can stub the file read.
type Linux struct {
	base

	OpenFn func(string) (io.ReadCloser, error)
}

// NewLinux returns a Linux variant wired to the package-level openFile
// helper. Named helper (not inline closure) keeps the factory a plain
// assignment — no closure body that needs test coverage.
func NewLinux() *Linux {
	return &Linux{OpenFn: openFile}
}

// Collect reads /etc/shells and returns the list of valid login shells.
// A missing file soft-misses to an empty list rather than erroring —
// distroless/scratch containers legitimately lack /etc/shells.
func (l *Linux) Collect(
	_ context.Context,
) (any, error) {
	rc, err := l.OpenFn(etcShellsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Info{Paths: []string{}}, nil
		}
		return nil, fmt.Errorf("open %s: %w", etcShellsPath, err)
	}
	defer func() { _ = rc.Close() }()
	return parseShells(rc)
}
