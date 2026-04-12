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

// Package rootgroup reports the primary group name for root (GID 0).
package rootgroup

import (
	"context"
)

// Info holds the root group data.
type Info struct {
	Name string `json:"name"` // typically "root" on Linux, "wheel" on macOS
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new root_group Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "root_group".
func (c *Collector) Name() string {
	return "root_group"
}

// DefaultEnabled returns true — collector is on by default.
func (c *Collector) DefaultEnabled() bool {
	return true
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers the root group name.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
