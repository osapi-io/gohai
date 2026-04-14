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

package memory

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects memory usage on macOS. gopsutil's mem.VirtualMemory
// (mach `host_statistics64`) provides Total / Available / Used / Free
// / Active / Inactive / Wired. We additionally run `vm_stat` through
// the shared Executor to populate Speculative and Compressed — the
// mach syscall doesn't expose either and they're essential for Apple
// Silicon performance diagnostics (aggressive compressor use).
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns the memory Info with vm_stat extensions layered on
// top of gopsutil's totals.
func (d *Darwin) Collect(
	ctx context.Context,
) (any, error) {
	info, err := readMemory(ctx)
	if err != nil {
		return nil, err
	}
	if d.Exec == nil {
		return info, nil
	}
	out, err := d.Exec.Execute(ctx, "vm_stat")
	if err != nil {
		return info, nil
	}
	applyVMStat(info, out)
	return info, nil
}

// vmStatPageSize matches the first line's page-size declaration.
// `Mach Virtual Memory Statistics: (page size of 16384 bytes)` on
// Apple Silicon; 4096 on Intel.
var vmStatPageSize = regexp.MustCompile(`page size of (\d+) bytes`)

// applyVMStat parses `vm_stat` output and populates Darwin-specific
// buckets (Speculative, Compressed) on the existing Info. Active /
// Inactive / Wired are left to gopsutil (they're already populated
// via the mach syscall and match vm_stat's numbers anyway).
func applyVMStat(
	info *Info,
	out []byte,
) {
	pageSize := uint64(4096)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if m := vmStatPageSize.FindStringSubmatch(line); m != nil {
			if ps, err := strconv.ParseUint(m[1], 10, 64); err == nil && ps > 0 {
				pageSize = ps
			}
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[idx+1:]), "."))
		pages, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}
		bytes := pages * pageSize
		switch key {
		case "Pages speculative":
			info.Speculative = bytes
		case "Pages stored in compressor":
			info.Compressed = bytes
		}
	}
}
