// Copyright (c) 2024 John Dewey

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

// Package filesystem collects mounted filesystem data.
package filesystem

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds mounted filesystem data.
type Info struct {
	Mounts []Mount `json:"mounts"`
}

// Mount represents a single mounted filesystem.
type Mount struct {
	Device      string   `json:"device"`
	Mountpoint  string   `json:"mountpoint"`
	Fstype      string   `json:"fstype"`
	Opts        []string `json:"opts,omitempty"`
	Total       uint64   `json:"total,omitempty"`
	Used        uint64   `json:"used,omitempty"`
	Free        uint64   `json:"free,omitempty"`
	UsedPercent float64  `json:"used_percent,omitempty"`
	InodesTotal uint64   `json:"inodes_total,omitempty"`
	InodesUsed  uint64   `json:"inodes_used,omitempty"`
	InodesFree  uint64   `json:"inodes_free,omitempty"`
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new filesystem Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "filesystem".
func (c *Collector) Name() string {
	return "filesystem"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers filesystem facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
