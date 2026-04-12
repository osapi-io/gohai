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
	PID       int32  `json:"pid"`
	PPID      int32  `json:"ppid,omitempty"`
	Name      string `json:"name,omitempty"`
	Username  string `json:"username,omitempty"`
	CmdLine   string `json:"cmd_line,omitempty"`   // OCSF-style: process.cmd_line
	State     string `json:"state,omitempty"`      // R/S/D/Z/T/I (Linux proc/status)
	StartTime uint64 `json:"start_time,omitempty"` // unix timestamp (seconds)
}

// Collector is the public interface every process variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "process" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the process variant for the host OS. gopsutil's process
// package works cross-platform — both variants share listing logic.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// listProcesses is the production bridge to gopsutil. Factored as a
// named function so factories can assign it as a plain function
// reference (no closure body). Tests inject a stub and don't touch
// this directly.
func listProcesses(
	ctx context.Context,
) ([]Process, error) {
	ps, err := process.ProcessesWithContext(ctx)
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
		out.PPID = ppid
	}
	if n, err := p.Name(); err == nil {
		out.Name = n
	}
	if u, err := p.Username(); err == nil {
		out.Username = u
	}
	if cl, err := p.Cmdline(); err == nil {
		out.CmdLine = cl
	}
	if st, err := p.Status(); err == nil && len(st) > 0 {
		out.State = st[0]
	}
	if ct, err := p.CreateTime(); err == nil && ct > 0 {
		// gopsutil returns ms since epoch; convert to seconds.
		out.StartTime = uint64(ct / 1000)
	}
	return out
}
