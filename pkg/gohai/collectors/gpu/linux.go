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

package gpu

import (
	"context"

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"

	"github.com/osapi-io/gohai/internal/collector"
)

// ghwGPUFn is the seam for ghw/gpu.New. Kept private so tests don't
// transitively need ghw; swapped via SetGHWGPUFn (export_test.go).
var ghwGPUFn = ghwgpu.New

// Linux reads /sys/class/drm via ghw/gpu and resolves each card's
// PCI vendor/product via ghw's bundled pci.ids database. No shell-out.
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect enumerates graphics cards. A ghw load error yields an empty
// Info with no error — containers and minimal VMs routinely lack
// /sys/class/drm and we shouldn't noisily fail for that.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	gi, err := ghwGPUFn()
	if err != nil || gi == nil {
		return info, nil
	}
	for _, c := range gi.GraphicsCards {
		card := Card{Address: c.Address}
		if c.DeviceInfo != nil {
			if c.DeviceInfo.Vendor != nil {
				card.Vendor = c.DeviceInfo.Vendor.Name
				card.VendorID = c.DeviceInfo.Vendor.ID
			}
			if c.DeviceInfo.Product != nil {
				card.Model = c.DeviceInfo.Product.Name
				card.DeviceID = c.DeviceInfo.Product.ID
			}
		}
		info.Cards = append(info.Cards, card)
	}
	return info, nil
}
