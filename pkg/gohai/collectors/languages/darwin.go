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

package languages

import (
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin detects language runtimes on macOS by shelling out to each
// runtime's version command. Same probes as Linux — version flags are
// identical across platforms.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect probes each language runtime and returns detected versions.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	stub := map[string]bool{}
	return collectLanguages(ctx, d.Exec, func(cmd string) bool {
		if v, ok := stub[cmd]; ok {
			return v
		}
		stub[cmd] = d.isStub(ctx, cmd)
		return stub[cmd]
	}), nil
}

// stubChecks names, for each runtime that macOS ships as a stub in
// /usr/bin, the command that succeeds only when the real runtime is
// installed. Without the Command Line Tools or a JDK, running the stub
// opens an install dialog on the user's desktop.
var stubChecks = map[string][]string{
	"python3": {"xcode-select", "-p"},
	"java":    {"/usr/libexec/java_home"},
}

// isStub reports whether running cmd would run a macOS stub, or whether
// cmd is not on the PATH at all. A runtime from another place, such as
// Homebrew, is never a stub.
func (d *Darwin) isStub(
	ctx context.Context,
	cmd string,
) bool {
	check, ok := stubChecks[cmd]
	if !ok {
		return false
	}
	out, err := d.Exec.Execute(ctx, "sh", "-c", "command -v "+cmd)
	if err != nil {
		return true
	}
	if strings.TrimSpace(string(out)) != "/usr/bin/"+cmd {
		return false
	}
	_, err = d.Exec.Execute(ctx, check[0], check[1:]...)
	return err != nil
}
