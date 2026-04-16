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

package pci

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin returns an empty Info. macOS exposes PCI topology through
// IOKit (reachable via `ioreg -p IODeviceTree -c IOPCIDevice -l`) but
// Ohai's pci plugin is Linux-only and the shape it emits is built
// around Linux's sysfs/lspci model; we match that scope by not
// implementing Darwin.
type Darwin struct {
	base
}

// NewDarwin returns a Darwin variant.
func NewDarwin() *Darwin {
	return &Darwin{}
}

// Collect returns an empty Info on macOS.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return &Info{Devices: map[string]Device{}}, nil
}
