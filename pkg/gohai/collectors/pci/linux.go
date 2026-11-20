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

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pcidev"
)

// Linux lists the PCI devices of /sys/bus/pci/devices through FS, and
// names them from the host's pci.ids, which it also reads through FS.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's sysfs.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the PCI Info, keyed by address.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{Devices: map[string]Device{}}
	for _, d := range pcidev.List(l.FS, pcidev.LoadDB(l.FS)) {
		entry := Device{
			VendorID:      d.VendorID,
			VendorName:    d.VendorName,
			DeviceID:      d.ProductID,
			DeviceName:    d.ProductName,
			ClassID:       d.ClassID,
			ClassName:     d.ClassName,
			SubclassID:    d.SubclassID,
			SubclassName:  d.SubclassName,
			SubsystemID:   d.SubsystemID,
			SubsystemName: d.SubsystemName,
			Revision:      d.Revision,
			Driver:        d.Driver,
			IOMMUGroup:    d.IOMMUGroup,
			ParentAddress: d.ParentAddress,
		}
		if strings.EqualFold(entry.Driver, "unknown") {
			entry.Driver = ""
		}
		info.Devices[d.Address] = entry
	}

	return info, nil
}
