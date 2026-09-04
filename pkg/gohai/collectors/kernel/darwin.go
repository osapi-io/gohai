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
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects kernel identity on macOS. Runs `sysctl -n
// hw.optional.x86_64` via the shared Executor for Rosetta 2 detection
// (issue #31) — a return of "1" with uname reporting `x86_64` means
// the process is running translated on Apple Silicon; we overwrite
// Machine to `arm64` (the real hardware) and set RosettaTranslated.
// Module/kext enumeration now lives in the kernel_modules collector.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns kernel Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	u, err := defaultUname()
	if err != nil {
		return nil, err
	}
	info := &Info{
		Name:      u.Name,
		Release:   u.Release,
		Version:   u.Version,
		Machine:   u.Machine,
		Processor: u.Machine,
		OS:        "Darwin",
	}
	if d.Exec != nil && detectRosetta(ctx, d.Exec, u.Machine) {
		info.Machine = "arm64"
		info.Processor = "arm64"
		info.RosettaTranslated = true
	}
	return info, nil
}

// detectRosetta returns true when we are executing under Rosetta 2:
// sysctl reports x86_64 capability AND uname's machine is x86_64.
// Either signal alone is ambiguous — sysctl returns 1 on native Intel
// and on Apple Silicon (where Rosetta is available); uname returns
// x86_64 on native Intel and under Rosetta. The conjunction pins it
// to the translated-on-Apple-Silicon case per issue #31.
func detectRosetta(
	ctx context.Context,
	exec executor.Executor,
	machine string,
) bool {
	if machine != "x86_64" {
		return false
	}
	out, err := exec.Execute(ctx, "sysctl", "-n", "hw.optional.x86_64")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "1"
}
