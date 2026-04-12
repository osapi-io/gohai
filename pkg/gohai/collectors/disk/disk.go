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

// Package disk collects per-device disk I/O counters.
package disk

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds per-device disk I/O counters.
type Info struct {
	Devices []Device `json:"devices"`
}

// Device represents I/O counters for a single block device.
type Device struct {
	Name       string `json:"name"`
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadTime   uint64 `json:"read_time,omitempty"`
	WriteTime  uint64 `json:"write_time,omitempty"`
	IoTime     uint64 `json:"io_time,omitempty"`
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new disk Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "disk".
func (c *Collector) Name() string {
	return "disk"
}

// Tier returns TierExtended.
func (c *Collector) Tier() collector.Tier {
	return collector.TierExtended
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers disk I/O facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
