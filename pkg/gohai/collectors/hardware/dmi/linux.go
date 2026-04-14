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

	"github.com/jaypipes/ghw/pkg/baseboard"
	"github.com/jaypipes/ghw/pkg/bios"
	"github.com/jaypipes/ghw/pkg/chassis"
	"github.com/jaypipes/ghw/pkg/product"

	"github.com/osapi-io/gohai/internal/collector"
)

// ghw upstream seams — tests swap these to inject canned SMBIOS data
// without requiring a host with real DMI. Each is the raw ghw New()
// call that reads /sys/class/dmi/id/* under the hood.
var (
	biosFn      = bios.New
	baseboardFn = baseboard.New
	chassisFn   = chassis.New
	productFn   = product.New
)

// Linux is the Linux dmi collector. Reads DMI data via ghw, which
// parses /sys/class/dmi/id/* (no root required — the sysfs entries
// are world-readable by default, except for product_uuid and
// product_serial which are 0400). When sysfs doesn't expose a field
// ghw returns an empty string, so distroless/minimal containers
// return mostly-empty sections rather than failing.
type Linux struct {
	base
}

// NewLinux returns a new Linux dmi collector.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect reads all four DMI sections. Any individual section that
// ghw fails to read is omitted (nil in the result); a completely
// empty Info is still returned as a non-nil pointer so consumers can
// check `facts.DMI != nil` reliably.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}

	if b, err := biosFn(); err == nil && b != nil {
		info.BIOS = &BIOS{
			Vendor:  b.Vendor,
			Version: b.Version,
			Date:    b.Date,
		}
	}
	if bb, err := baseboardFn(); err == nil && bb != nil {
		info.Baseboard = &Baseboard{
			Vendor:       bb.Vendor,
			Product:      bb.Product,
			Version:      bb.Version,
			SerialNumber: bb.SerialNumber,
			AssetTag:     bb.AssetTag,
		}
	}
	if ch, err := chassisFn(); err == nil && ch != nil {
		info.Chassis = &Chassis{
			Vendor:          ch.Vendor,
			Type:            ch.Type,
			TypeDescription: ch.TypeDescription,
			Version:         ch.Version,
			SerialNumber:    ch.SerialNumber,
			AssetTag:        ch.AssetTag,
		}
	}
	if p, err := productFn(); err == nil && p != nil {
		info.Product = &Product{
			Vendor:       p.Vendor,
			Name:         p.Name,
			Family:       p.Family,
			Version:      p.Version,
			SerialNumber: p.SerialNumber,
			UUID:         p.UUID,
			SKU:          p.SKU,
		}
	}
	return info, nil
}
