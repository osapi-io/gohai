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
)

// Info holds a snapshot of running processes.
type Info struct {
	Count     int       `json:"count"`
	Processes []Process `json:"processes,omitempty"`
}

// Process is a single process snapshot entry. Fields that can't be read
// (e.g., permission-denied for another user's process) are left empty.
type Process struct {
	PID      int32  `json:"pid"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	Cmdline  string `json:"cmdline,omitempty"`
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new process Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "process".
func (c *Collector) Name() string {
	return "process"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers process facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
