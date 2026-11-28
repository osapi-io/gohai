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

package rpm

import (
	"context"
	"fmt"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects RPM macro definitions on Linux hosts. Exec runs
// `rpm --showrc` (production: real executor; tests: gomock mock).
type Linux struct {
	base

	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands()}
}

// Collect runs `rpm --showrc` and returns the parsed macro map. When
// rpm is not installed or the command fails, an empty macro map is
// returned with no error. This matches Ohai's rpm.rb "skip if rpm not
// found" behaviour and avoids erroring on non-RPM systems.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{Macros: map[string]string{}}
	if l.Exec == nil {
		return info, nil
	}
	out, err := l.Exec.Execute(ctx, "rpm", "--showrc")
	if err != nil {
		// rpm is not installed, so the host is not RPM-based: the source
		// is absent and the macros are empty. Any other failure of rpm is
		// an error.
		if collector.Absent(err) {
			return info, nil
		}
		return nil, fmt.Errorf("rpm --showrc: %w", err)
	}
	info.Macros = parseShowrc(out)
	return info, nil
}
