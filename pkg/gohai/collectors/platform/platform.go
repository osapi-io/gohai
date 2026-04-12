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

// Package platform collects operating system platform identification facts.
package platform

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds platform identification data.
type Info struct {
	OS           string `json:"os"`              // runtime.GOOS: "linux", "darwin", "windows"
	Name         string `json:"name"`            // distro/product: "ubuntu", "rhel", "darwin"
	Version      string `json:"version"`         // "24.04", "14.4.1"
	Family       string `json:"family"`          // "debian", "rhel", "mac_os_x"
	Architecture string `json:"architecture"`    // "amd64", "arm64"
	Build        string `json:"build,omitempty"` // kernel build (macOS)
}

// Collector implements the collector.Collector interface for platform facts.
type Collector struct{}

// New returns a new platform Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "platform".
func (c *Collector) Name() string {
	return "platform"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers platform facts. Implementation lives in platform_<os>.go.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
