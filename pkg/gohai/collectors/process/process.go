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

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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

	// CPUPercent is the CPU time the process used since it started, as a
	// share of the time since it started. 100 is one full CPU.
	CPUPercent float64 `json:"cpu_percent,omitempty"`
	// MemPercent is RSS as a share of the host's physical memory.
	MemPercent float64 `json:"mem_percent,omitempty"`
	// RSS is the resident set size, in bytes.
	RSS int64 `json:"rss,omitempty"`
	// CPUSeconds is the user and system CPU time the process used. The
	// difference between two samples, over the time between them, is the
	// CPU share in that time.
	CPUSeconds float64 `json:"cpu_seconds,omitempty"`
}

// Collector is the public interface every process variant satisfies.
type Collector interface {
	collector.Collector

	// Read is the one process pid, through the same sources as Collect,
	// without reading the others. It is an error when there is no such
	// process.
	Read(ctx context.Context, pid int32) (*Process, error)
}

type base struct{}

func (base) Name() string                 { return "process" }
func (base) Category() collector.Category { return collector.CategoryMisc }

// DefaultEnabled is false: process enumeration scales with process
// count and isn't useful on every invocation. Opt in via
// --collector.process or WithEnabled("process").
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the process variant for the host OS: Linux reads /proc,
// and macOS runs ps.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// states are the words for a process state letter, as gopsutil names
// them: R is running, S is sleep, and so on.
var states = map[byte]string{
	'R': "running", 'S': "sleep", 'I': "idle", 'D': "blocked", 'T': "stop",
	't': "stop", 'W': "wait", 'Z': "zombie", 'L': "lock",
}

// state is the word for the first letter of s, which is not empty, or ""
// for a letter that has none.
func state(
	s string,
) string {
	return states[s[0]]
}
