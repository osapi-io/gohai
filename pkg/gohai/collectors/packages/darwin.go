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

package packages

import (
	"bufio"
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects Homebrew-installed packages on macOS hosts via
// `brew list --versions`. Embeds base for static collector metadata.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect runs `brew list --versions` and parses its output. Returns
// an empty list (not an error) if Homebrew is absent.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return &Info{Packages: []Package{}}, nil
	}
	out, err := d.Exec.Execute(ctx, "brew", "list", "--versions")
	if err != nil {
		return &Info{Packages: []Package{}}, nil
	}
	info := &Info{Packages: []Package{}}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// `brew list --versions` output: "pkg 1.2.3 1.2.4"
		// The first token is the name; remaining tokens are versions.
		// We take the last version in the list (most recent installed).
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := parts[len(parts)-1]
		info.Packages = append(info.Packages, Package{
			Name:           name,
			Version:        version,
			PackageManager: "brew",
		})
	}
	return info, nil
}
