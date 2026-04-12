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

// Package fips reports whether the kernel is running in FIPS mode
// (FIPS 140-2 / 140-3).
package fips

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds FIPS mode state.
type Info struct {
	Kernel Kernel  `json:"kernel"`
	Policy *Policy `json:"policy,omitempty"`
}

// Kernel holds the kernel-level FIPS flag from
// /proc/sys/crypto/fips_enabled. True means the kernel booted with
// fips=1. This is the 140-2-era signal Ohai reads indirectly via
// OpenSSL.fips_mode.
type Kernel struct {
	Enabled bool `json:"enabled"`
}

// Policy holds the user-space crypto policy state, present on hosts
// with /etc/crypto-policies/config (RHEL 8+, Fedora 30+, CentOS Stream,
// Amazon Linux 2023). FIPS 140-3 systems can toggle this post-boot
// without flipping the kernel flag, so kernel.enabled alone can
// misreport the effective crypto posture.
type Policy struct {
	Name          string `json:"name"`           // e.g. "FIPS", "FIPS:OSPP", "DEFAULT"
	FIPSEffective bool   `json:"fips_effective"` // true if Name starts with "FIPS"
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new fips Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "fips".
func (c *Collector) Name() string {
	return "fips"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect reports FIPS kernel mode. Implementation lives in linux.go /
// darwin.go / other.go.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
