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

package mdadm

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects software RAID array data on Linux. FS provides
// /proc/mdstat access; Exec runs `mdadm --detail` for per-array
// enrichment. Both are injected for testability.
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// the production executor.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: executor.New(),
	}
}

// Collect reads /proc/mdstat and enriches each array with `mdadm --detail`
// output. When mdadm is not installed, arrays are returned with only the
// fields extracted from /proc/mdstat. A missing /proc/mdstat returns an
// empty list without error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	readFn := func(path string) ([]byte, error) {
		return l.FS.ReadFile(path)
	}

	var execFn func(context.Context, string, ...string) ([]byte, error)
	if l.Exec != nil {
		execFn = l.Exec.Execute
	}

	arrays, err := collectArrays(ctx, readFn, execFn)
	if err != nil {
		return nil, err
	}
	return &Info{Arrays: arrays}, nil
}
