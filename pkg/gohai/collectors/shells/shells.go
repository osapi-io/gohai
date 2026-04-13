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

// Package shells reports the list of valid login shells installed on the
// host (contents of /etc/shells).
package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// etcShellsPath is the canonical location of the installed-shells list.
// Same path on Linux and macOS — both follow POSIX convention.
const etcShellsPath = "/etc/shells"

// Info holds the list of valid login shells.
type Info struct {
	Paths []string `json:"paths"`
}

// Collector is the public interface every shells variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "shells".
func (base) Name() string { return "shells" }

// DefaultEnabled returns true — shells is on by default.
func (base) DefaultEnabled() bool { return true }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the shells collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseShells parses /etc/shells format: one path per line. Skips
// comments (`#...`), blank lines, and non-absolute entries. Format is
// POSIX and identical across Linux and macOS, so this helper is shared.
func parseShells(
	content []byte,
) *Info {
	info := &Info{Paths: []string{}}
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "/") {
			continue
		}
		info.Paths = append(info.Paths, line)
	}
	return info
}

// wrapReadError decides whether a read error should soft-miss to an
// empty Info (file absent — distroless/scratch containers legitimately
// lack /etc/shells) or propagate as a real failure.
func wrapReadError(
	err error,
) (*Info, error) {
	// avfs returns os.ErrNotExist-wrapped errors for missing files.
	// We check via string match on the sentinel since avfs's error
	// wrapping doesn't preserve errors.Is semantics across all
	// backends.
	if strings.Contains(err.Error(), "no such file or directory") ||
		strings.Contains(err.Error(), "file does not exist") {
		return &Info{Paths: []string{}}, nil
	}
	return nil, fmt.Errorf("read %s: %w", etcShellsPath, err)
}
