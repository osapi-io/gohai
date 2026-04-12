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

package process

import (
	"context"

	"github.com/shirou/gopsutil/v4/process"
)

// Linux collects a process snapshot on Linux hosts via gopsutil.
type Linux struct {
	base

	ProcessesFn func(context.Context) ([]ProcSnapshot, error)
}

// NewLinux returns a Linux variant wired to gopsutil.
func NewLinux() *Linux {
	return &Linux{ProcessesFn: gopsutilProcesses}
}

// gopsutilProcesses adapts gopsutil's Processes return to our
// ProcSnapshot interface. Named helper so the factory assigns a
// function reference (no closure body to cover).
func gopsutilProcesses(
	ctx context.Context,
) ([]ProcSnapshot, error) {
	ps, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProcSnapshot, 0, len(ps))
	for _, p := range ps {
		out = append(out, procAdapter{P: p})
	}
	return out, nil
}

// procAdapter adapts *gopsutil/process.Process to our ProcSnapshot
// interface. gopsutil's Process has context-accepting methods; we call
// the non-context variants here to keep the ProcSnapshot interface
// simple.
type procAdapter struct {
	P *process.Process
}

func (a procAdapter) Pid() int32                 { return a.P.Pid }
func (a procAdapter) Ppid() (int32, error)       { return a.P.Ppid() }
func (a procAdapter) Name() (string, error)      { return a.P.Name() }
func (a procAdapter) Username() (string, error)  { return a.P.Username() }
func (a procAdapter) Cmdline() (string, error)   { return a.P.Cmdline() }
func (a procAdapter) Status() ([]string, error)  { return a.P.Status() }
func (a procAdapter) CreateTime() (int64, error) { return a.P.CreateTime() }

// Collect returns a process snapshot. Per-process read errors (access
// denied, zombie parent, etc.) fall through to empty fields rather
// than erroring — we'd rather list a process with partial data than
// drop it.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	procs, err := l.ProcessesFn(ctx)
	if err != nil {
		return nil, err
	}
	out := &Info{Count: len(procs), Processes: make([]Process, 0, len(procs))}
	for _, p := range procs {
		out.Processes = append(out.Processes, snapshotOf(p))
	}
	return out, nil
}
