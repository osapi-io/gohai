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
	"strings"

	ghwpci "github.com/jaypipes/ghw/pkg/pci"

	"github.com/osapi-io/gohai/internal/collector"
)

// ghwPCIFn is the seam for ghw/pci.New. Kept private so tests don't
// transitively need ghw; swapped via SetGHWPCIFn (export_test.go).
var ghwPCIFn = ghwpci.New

// Linux reads /sys/bus/pci/devices via ghw/pci and resolves each
// device's vendor/product/class via ghw's bundled pci.ids database.
// No shell-out to lspci required.
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect enumerates PCI devices. A ghw load error yields an empty
// Info with no error — containers and minimal VMs routinely lack
// /sys/bus/pci/devices and we shouldn't noisily fail for that.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{Devices: map[string]Device{}}
	pi, err := ghwPCIFn()
	if err != nil || pi == nil {
		return info, nil
	}
	for _, d := range pi.Devices {
		if d == nil || d.Address == "" {
			continue
		}
		entry := Device{
			Revision:      d.Revision,
			Driver:        d.Driver,
			IOMMUGroup:    d.IOMMUGroup,
			ParentAddress: d.ParentAddress,
		}
		if d.Vendor != nil {
			entry.VendorID = d.Vendor.ID
			entry.VendorName = d.Vendor.Name
		}
		if d.Product != nil {
			entry.DeviceID = d.Product.ID
			entry.DeviceName = d.Product.Name
		}
		if d.Class != nil {
			entry.ClassID = d.Class.ID
			entry.ClassName = d.Class.Name
		}
		if d.Subclass != nil {
			entry.SubclassID = d.Subclass.ID
			entry.SubclassName = d.Subclass.Name
		}
		if d.Subsystem != nil {
			entry.SubsystemID = d.Subsystem.ID
			entry.SubsystemName = d.Subsystem.Name
		}
		// ghw returns "unknown" for unresolved driver; normalize to empty.
		if strings.EqualFold(entry.Driver, "unknown") {
			entry.Driver = ""
		}
		info.Devices[d.Address] = entry
	}
	return info, nil
}
