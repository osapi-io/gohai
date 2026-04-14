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

// Package machineid reports the host's stable machine identifier.
package machineid

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// via SetHostInfoFn (export_test.go).
var hostInfoFn = host.InfoWithContext

// readHostID is the production bridge. Wraps the private gopsutil call
// and returns just the HostID string — callers never see gopsutil
// types.
func readHostID(
	ctx context.Context,
) (string, error) {
	info, err := hostInfoFn(ctx)
	if err != nil {
		return "", fmt.Errorf("host.Info: %w", err)
	}
	if info == nil {
		return "", nil
	}
	return info.HostID, nil
}

// Info holds the machine ID.
type Info struct {
	ID string `json:"id"` // stable host identifier — survives reboots
}

// Collector is the public interface every machine_id variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "machine_id" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the machine_id variant appropriate for the detected host.
// Linux and Darwin have substantially different sources of truth, so
// each gets its own struct.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}
