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
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin enumerates legacy kernel extensions via `kextstat -k -l`.
// System Extensions (macOS 11+) live under /Library/SystemExtensions/
// and require systemextensionsctl, which gohai does not query yet.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
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
	if collector.Absent(err) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read kexts: %w", err)
	}
	info.Modules = parseKextstat(out)
	return info, nil
}

// kextstatLine matches `kextstat -k -l` fixed-width rows. Captures:
//
//	1: index, 2: refcount, 3: size (hex), 4: name, 5: version.
//
// The name is any run of non-space characters, because a bundle ID can
// hold a hyphen or an underscore, such as com.paragon-software.filesystems.ntfs.
// The version is everything inside the parentheses, because a bundle
// version can carry a letter, such as 1.0.0d1 or 2.1.0b3.
var kextstatLine = regexp.MustCompile(
	`^\s*(\d+)\s+(\d+)\s+0x[0-9a-f]+\s+0x([0-9a-f]+)\s+0x[0-9a-f]+\s+(\S+) \(([^)]+)\)`,
)

// parseKextstat turns `kextstat -k -l` output into a Module map keyed
// by kext bundle ID.
func parseKextstat(
	b []byte,
) map[string]Module {
	out := map[string]Module{}
	for rawLine := range strings.Lines(string(b)) {
		m := kextstatLine.FindStringSubmatch(strings.TrimRight(rawLine, "\r\n"))
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
