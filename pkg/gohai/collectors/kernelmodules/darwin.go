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

package kernelmodules

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin enumerates legacy kernel extensions via `kextstat -k -l`.
// System Extensions (macOS 11+) live under /Library/SystemExtensions/
// and require systemextensionsctl — not yet queried.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns the loaded kext map. A failed kextstat yields an
// empty Info with no error, matching the silent-on-miss convention.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	if d.Exec == nil {
		return info, nil
	}
	out, err := d.Exec.Execute(ctx, "kextstat", "-k", "-l")
	if err != nil {
		return info, nil
	}
	info.Modules = parseKextstat(out)
	return info, nil
}

// kextstatLine matches `kextstat -k -l` fixed-width rows. Captures:
//
//	1: index, 2: refcount, 3: size (hex), 4: name, 5: version.
//
// Character classes match Ohai's regex exactly (kernel.rb :darwin
// branch) — field-tested on real macOS output.
var kextstatLine = regexp.MustCompile(
	`^\s*(\d+)\s+(\d+)\s+0x[0-9a-f]+\s+0x([0-9a-f]+)\s+0x[0-9a-f]+\s+([a-zA-Z0-9\.]+) \(([0-9\.]+)\)`,
)

// parseKextstat turns `kextstat -k -l` output into a Module map keyed
// by kext bundle ID.
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
		if idx, err := strconv.Atoi(m[1]); err == nil {
			mod.Index = idx
		}
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
