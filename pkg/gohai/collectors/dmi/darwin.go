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

package dmi

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin is the macOS dmi collector. macOS doesn't expose SMBIOS via
// the same sysfs mechanism Linux uses; hardware identity on macOS
// comes from IOKit (`ioreg`) and `system_profiler` instead, covered
// by the `hardware` collector. Returning an empty Info here keeps
// the shape consistent across platforms so consumers can safely check
// `facts.DMI.Product` without a per-OS branch — they just get empty
// strings on macOS.
type Darwin struct {
	base
}

// NewDarwin returns a new Darwin dmi collector.
func NewDarwin() *Darwin {
	return &Darwin{}
}

// Collect returns an empty Info — macOS has no SMBIOS/DMI surface.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return &Info{}, nil
}
