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
func (*Linux) Collect(
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

		info.Devices[d.Address] = pciDevice(d)
	}

	return info, nil
}

// pciDevice reads one device. Each of the five lookups is independent —
// a device the PCI database does not fully describe still reports what
// it does know.
func pciDevice(
	d *ghwpci.Device,
) Device {
	entry := Device{
		Revision:      d.Revision,
		Driver:        d.Driver,
		IOMMUGroup:    d.IOMMUGroup,
		ParentAddress: d.ParentAddress,
	}

	// ghw says "unknown" where it could not resolve the driver.
	if strings.EqualFold(entry.Driver, "unknown") {
		entry.Driver = ""
	}

	if v := d.Vendor; v != nil {
		entry.VendorID, entry.VendorName = v.ID, v.Name
	}

	if p := d.Product; p != nil {
		entry.DeviceID, entry.DeviceName = p.ID, p.Name
	}

	if c := d.Class; c != nil {
		entry.ClassID, entry.ClassName = c.ID, c.Name
	}

	if s := d.Subclass; s != nil {
		entry.SubclassID, entry.SubclassName = s.ID, s.Name
	}

	if s := d.Subsystem; s != nil {
		entry.SubsystemID, entry.SubsystemName = s.ID, s.Name
	}

	return entry
}
