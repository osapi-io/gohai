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
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects kernel facts on macOS. Uses unix.Uname for the
// top-level identity fields (via the package-level unameSyscall seam)
// and shells out via the shared Executor for:
//
//   - `sysctl -n hw.optional.x86_64` — Rosetta 2 detection (issue #31).
//     A return of "1" with uname reporting `x86_64` means the process
//     is running translated on Apple Silicon; we overwrite Machine to
//     `arm64` (the real hardware) and set RosettaTranslated = true.
//   - `kextstat -k -l` — legacy kernel extension enumeration (issue
//     #30). System Extensions (macOS 11+) are not queried; those live
//     under /Library/SystemExtensions/ and require
//     systemextensionsctl.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns kernel Info.
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	name, release, version, machine, err := defaultUname()
	if err != nil {
		return nil, err
	}
	info := &Info{
		Name:      name,
		Release:   release,
		Version:   version,
		Machine:   machine,
		Processor: machine,
		OS:        "Darwin",
	}

	if d.Exec != nil {
		if rosetta := detectRosetta(ctx, d.Exec, machine); rosetta {
			info.Machine = "arm64"
			info.Processor = "arm64"
			info.RosettaTranslated = true
		}
		if out, err := d.Exec.Execute(ctx, "kextstat", "-k", "-l"); err == nil {
			info.Modules = parseKextstat(out)
		}
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

// kextstatLine matches `kextstat -k -l` fixed-width rows. Captures:
//
//	1: index, 2: refcount, 3: size (hex), 4: name, 5: version.
//
// Example:
//
//	1    0 0xffffff7f80000000 0x8a8 0xa8 com.apple.iokit.IOPCIFamily (2.9)
//
// Character classes match Ohai's regex exactly (kernel.rb :darwin
// branch) — field-tested on real macOS output. Deliberate parity.
var kextstatLine = regexp.MustCompile(
	`^\s*(\d+)\s+(\d+)\s+0x[0-9a-f]+\s+0x([0-9a-f]+)\s+0x[0-9a-f]+\s+([a-zA-Z0-9\.]+) \(([0-9\.]+)\)`,
)

// parseKextstat turns `kextstat -k -l` output into a Module map keyed
// by kext bundle ID. Mirrors Ohai's regex in kernel.rb's :darwin
// branch.
func parseKextstat(
	b []byte,
) map[string]Module {
	out := map[string]Module{}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		m := kextstatLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		mod := Module{Version: m[5]}
		if rc, err := strconv.Atoi(m[2]); err == nil {
			mod.RefCount = rc
		}
		if sz, err := strconv.ParseUint(m[3], 16, 64); err == nil {
			mod.Size = sz
		}
		out[m[4]] = mod
	}
	return out
}
