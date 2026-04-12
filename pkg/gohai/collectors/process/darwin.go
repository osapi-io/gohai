//go:build darwin

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
	"fmt"

	"github.com/shirou/gopsutil/v4/process"
)

// Snapshot abstracts what we need from a gopsutil process for testability.
type Snapshot interface {
	Pid() int32
	Name() (string, error)
	Username() (string, error)
	Cmdline() (string, error)
}

// processesFn lists live processes. Errors from gopsutil's Processes()
// are infrequent in practice; we treat them as an empty list rather than
// failing the whole collector.
var processesFn = func(
	ctx context.Context,
) ([]Snapshot, error) {
	ps, _ := process.ProcessesWithContext(ctx)
	out := make([]Snapshot, 0, len(ps))
	for _, p := range ps {
		out = append(out, realSnap{p: p})
	}
	return out, nil
}

type realSnap struct{ p *process.Process }

func (r realSnap) Pid() int32                { return r.p.Pid }
func (r realSnap) Name() (string, error)     { return r.p.Name() }
func (r realSnap) Username() (string, error) { return r.p.Username() }
func (r realSnap) Cmdline() (string, error)  { return r.p.Cmdline() }

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, processesFn)
}

func collectFromGopsutil(
	ctx context.Context,
	fn func(context.Context) ([]Snapshot, error),
) (any, error) {
	ps, err := fn(ctx)
	if err != nil {
		return nil, fmt.Errorf("process.Processes: %w", err)
	}
	procs := make([]Process, 0, len(ps))
	for _, p := range ps {
		entry := Process{PID: p.Pid()}
		if n, err := p.Name(); err == nil {
			entry.Name = n
		}
		if u, err := p.Username(); err == nil {
			entry.Username = u
		}
		if c, err := p.Cmdline(); err == nil {
			entry.Cmdline = c
		}
		procs = append(procs, entry)
	}
	return &Info{Count: len(procs), Processes: procs}, nil
}
