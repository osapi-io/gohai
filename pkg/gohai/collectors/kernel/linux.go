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

package kernel

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Linux collects kernel identity on Linux. `processor` and `os` are
// synthesized per Option A of issue #29 (Machine and the static string
// "GNU/Linux") rather than shelling out to `uname -p` / `uname -o`.
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect returns kernel Info.
func (*Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	u, err := defaultUname()
	if err != nil {
		return nil, err
	}
	return &Info{
		Name:      u.Name,
		Release:   u.Release,
		Version:   u.Version,
		Machine:   u.Machine,
		Processor: u.Machine,
		OS:        "GNU/Linux",
	}, nil
}
