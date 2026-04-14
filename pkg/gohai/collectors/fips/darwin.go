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

package fips

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin reports no FIPS data on macOS. Apple's CoreCrypto module is
// FIPS 140-validated by Apple but there's no kernel-level runtime
// toggle equivalent to Linux's /proc/sys/crypto/fips_enabled.
// Collect() returns nil for the facts.Fips field to match Ohai
// (which only provides fips on :linux and :windows).
type Darwin struct {
	base
}

// NewDarwin returns a Darwin variant.
func NewDarwin() *Darwin {
	return &Darwin{}
}

// Collect returns nil — no FIPS facts on darwin.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return nil, nil
}
