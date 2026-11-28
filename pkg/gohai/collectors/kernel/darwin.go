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
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects kernel identity on macOS: kern.ostype, kern.osrelease,
// kern.version and hw.machine through `sysctl -n`, the values that uname(3)
// reports. It also runs `sysctl -n sysctl.proc_translated` for Rosetta
// 2 detection (issue #31): a "1" with uname reporting `x86_64` means the
// process runs translated on Apple Silicon, so Machine is set to `arm64`
// (the real hardware) and RosettaTranslated is set.
// Module/kext enumeration now lives in the kernel_modules collector.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns kernel Info. With no Exec it gives no fact, as every
// value comes from sysctl.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	// The four values of uname(3), one to a line, in the order asked.
	out, err := d.Exec.Execute(ctx, "sysctl", "-n",
		"kern.ostype", "kern.osrelease", "kern.version", "hw.machine")
	if err != nil {
		return nil, fmt.Errorf("sysctl kern.ostype kern.osrelease kern.version hw.machine: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 4 {
		return nil, fmt.Errorf("sysctl gave %d values, not 4: %q", len(lines), out)
	}
	name, release, version, machine := field([]byte(lines[0])), field([]byte(lines[1])),
		field([]byte(lines[2])), field([]byte(lines[3]))
	info := &Info{
		Name:      name,
		Release:   release,
		Version:   version,
		Machine:   machine,
		Processor: machine,
		OS:        "Darwin",
	}
	if detectRosetta(ctx, d.Exec, machine) {
		info.Machine = "arm64"
		info.Processor = "arm64"
		info.RosettaTranslated = true
	}
	return info, nil
}

// detectRosetta returns true when the process runs under Rosetta 2.
// uname gives x86_64 both on a native Intel Mac and under Rosetta, so
// it asks sysctl.proc_translated, which is 1 only for a translated
// process. The key is 0 for a native process on Apple Silicon, and an
// Intel Mac does not have it, which reads as not translated.
// hw.optional.x86_64 does not tell the two apart: it is 1 on an Intel
// Mac too. Not tested on Intel hardware.
func detectRosetta(
	ctx context.Context,
	exec source.Runner,
	machine string,
) bool {
	if machine != "x86_64" {
		return false
	}
	out, err := exec.Execute(ctx, "sysctl", "-n", "sysctl.proc_translated")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "1"
}
