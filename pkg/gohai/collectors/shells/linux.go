//go:build linux

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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// openFn opens a file for reading. Swappable for tests.
var openFn = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// etcShellsPath is the canonical location of the installed-shells list.
const etcShellsPath = "/etc/shells"

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFunc(openFn)
}

// collectFromFunc opens /etc/shells via the supplied opener, parses it into
// a list of shell paths, and returns the Info.
func collectFromFunc(
	open func(string) (io.ReadCloser, error),
) (*Info, error) {
	rc, err := open(etcShellsPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", etcShellsPath, err)
	}
	defer func() { _ = rc.Close() }()
	return parseShells(rc)
}

// parseShells parses /etc/shells format: one path per line, comments
// start with '#', blank lines ignored.
func parseShells(
	r io.Reader,
) (*Info, error) {
	info := &Info{Paths: []string{}}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		info.Paths = append(info.Paths, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read shells: %w", err)
	}
	return info, nil
}
