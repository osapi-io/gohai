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

// Package pci enumerates PCI devices attached to the host. On Linux
// we wrap ghw/pci which walks /sys/bus/pci/devices and resolves
// vendor/product/class names via ghw's bundled pci.ids database — no
// shell-out to `lspci` required. macOS has no sysfs PCI tree and
// returns an empty Info.
//
// Mirrors Ohai's linux/lspci.rb output shape (map keyed by PCI address)
// but sources the data from sysfs instead of shelling to `lspci`.
package pci

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the enumerated PCI devices keyed by address.
type Info struct {
	Devices map[string]Device `json:"devices"`
}

// Device is one PCI device entry. Matches the shape Ohai's
// linux/lspci.rb emits under node['pci'][<address>].
type Device struct {
	VendorID      string `json:"vendor_id,omitempty"`
	VendorName    string `json:"vendor_name,omitempty"`
	DeviceID      string `json:"device_id,omitempty"`
	DeviceName    string `json:"device_name,omitempty"`
	ClassID       string `json:"class_id,omitempty"`
	ClassName     string `json:"class_name,omitempty"`
	SubclassID    string `json:"subclass_id,omitempty"`
	SubclassName  string `json:"subclass_name,omitempty"`
	SubsystemID   string `json:"sdevice_id,omitempty"`
	SubsystemName string `json:"sdevice_name,omitempty"`
	Revision      string `json:"revision,omitempty"`
	Driver        string `json:"driver,omitempty"`
	IOMMUGroup    string `json:"iommu_group,omitempty"`
	ParentAddress string `json:"parent_address,omitempty"`
}

// Collector is the public interface every pci variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "pci" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the pci variant appropriate for the host OS.
func New() Collector {
	if platform.IsDarwin() {
		return NewDarwin()
	}
	return NewLinux()
}
