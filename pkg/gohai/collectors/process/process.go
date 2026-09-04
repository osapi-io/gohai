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

// Package process collects a snapshot of running processes.
package process

import (
	"context"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds a snapshot of running processes.
type Info struct {
	Count     int       `json:"count"`
	Processes []Process `json:"processes,omitempty"`
}

// Process is a single process snapshot entry. Fields that can't be
// read (permission-denied for another user's process, zombie parents,
// etc.) are left empty rather than erroring.
type Process struct {
	PID          int32  `json:"pid"`
	ParentPID    int32  `json:"parent_pid,omitempty"`
	Name         string `json:"name,omitempty"`
	Owner        string `json:"owner,omitempty"`
	CmdLine      string `json:"cmd_line,omitempty"`      // OCSF-style: process.cmd_line
	State        string `json:"state,omitempty"`         // R/S/D/Z/T/I (Linux proc/status)
	CreationTime int64  `json:"creation_time,omitempty"` // unix timestamp (seconds)
}

// Collector is the public interface every process variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

// gopsutil reports creation time in milliseconds.
const millisPerSecond = 1000

func (base) Name() string     { return "process" }
func (base) Category() string { return collector.CategoryMisc }

// DefaultEnabled is false: process enumeration scales with process
// count and isn't useful on every invocation. Opt in via
// --collector.process or WithEnabled("process").
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the process variant for the host OS. gopsutil's process
// package works cross-platform — both variants share listing logic.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// processesFn is the injection seam for gopsutil's
// process.ProcessesWithContext. Kept private so importers don't
// transitively need gopsutil. Swapped via SetProcessesFn.
var processesFn = process.ProcessesWithContext

// listProcesses is the production bridge to gopsutil. Factored as a
// named function so factories can assign it as a plain function
// reference (no closure body). Tests inject a stub and don't touch
// this directly.
func listProcesses(
	ctx context.Context,
) ([]Process, error) {
	ps, err := processesFn(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Process, 0, len(ps))
	for _, p := range ps {
		out = append(out, snapshotFromGopsutil(p))
	}
	return out, nil
}

// snapshotFromGopsutil maps a *gopsutil.Process onto our Process
// struct. Per-field read errors (access denied, zombie state, etc.)
// leave the corresponding field zero-valued.
func snapshotFromGopsutil(
	p *process.Process,
) Process {
	out := Process{PID: p.Pid}

	if ppid, err := p.Ppid(); err == nil {
		out.ParentPID = ppid
	}

	applyProcessStrings(p, &out)
	applyProcessState(p, &out)

	return out
}

// applyProcessStrings copies the descriptive fields. Each is read
// separately because a process can deny one and answer the rest.
func applyProcessStrings(
	p *process.Process,
	out *Process,
) {
	for _, f := range []struct {
		read func() (string, error)
		dst  *string
	}{
		{p.Name, &out.Name},
		{p.Username, &out.Owner},
		{p.Cmdline, &out.CmdLine},
	} {
		if v, err := f.read(); err == nil {
			*f.dst = v
		}
	}
}

// applyProcessState copies the run state and start time.
func applyProcessState(
	p *process.Process,
	out *Process,
) {
	if st, err := p.Status(); err == nil && len(st) > 0 {
		out.State = st[0]
	}

	// gopsutil reports creation time in milliseconds since the epoch.
	if ct, err := p.CreateTime(); err == nil && ct > 0 {
		out.CreationTime = ct / millisPerSecond
	}
}
