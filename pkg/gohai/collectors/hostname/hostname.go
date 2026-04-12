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

// Package hostname collects hostname, FQDN, and domain identification.
package hostname

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds hostname identification data.
type Info struct {
	Hostname string `json:"hostname"`         // short hostname (e.g., "web01")
	FQDN     string `json:"fqdn,omitempty"`   // fully qualified (e.g., "web01.example.com")
	Domain   string `json:"domain,omitempty"` // domain portion (e.g., "example.com")
}

// Collector implements the collector.Collector interface for hostname facts.
type Collector struct{}

// New returns a new hostname Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "hostname".
func (c *Collector) Name() string {
	return "hostname"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers hostname facts. Implementation lives in linux.go / darwin.go.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
