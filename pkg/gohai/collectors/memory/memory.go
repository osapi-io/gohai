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

// Package memory collects virtual and swap memory facts.
package memory

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds memory usage data.
type Info struct {
	Total       uint64  `json:"total"` // bytes
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Free        uint64  `json:"free"`
	Buffers     uint64  `json:"buffers,omitempty"`
	Cached      uint64  `json:"cached,omitempty"`
	Swap        *Swap   `json:"swap,omitempty"`
}

// Swap holds swap memory usage.
type Swap struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// Collector implements the collector.Collector interface for memory facts.
type Collector struct{}

// New returns a new memory Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "memory".
func (c *Collector) Name() string {
	return "memory"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers memory facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
