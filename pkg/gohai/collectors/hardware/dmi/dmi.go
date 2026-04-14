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

// Package dmi reports SMBIOS / DMI data — BIOS, baseboard, chassis,
// and product identity. Consumers use this to detect virtualization
// vendors (cloud collectors check product.name for "Google Compute
// Engine", "Amazon EC2", etc.), inventory hardware, and drive
// compliance tooling. On Linux the data comes from
// /sys/class/dmi/id/* via ghw (no root needed). macOS has no SMBIOS
// equivalent; the collector returns an empty Info there — the macOS
// `hardware` collector (planned) covers that ground.
package dmi

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info is the DMI view consumers want. Each section is nil when that
// part of SMBIOS isn't available (virtual machines often omit chassis
// data; minimal containers may have no DMI at all).
type Info struct {
	BIOS      *BIOS      `json:"bios,omitempty"`
	Baseboard *Baseboard `json:"baseboard,omitempty"`
	Chassis   *Chassis   `json:"chassis,omitempty"`
	Product   *Product   `json:"product,omitempty"`
}

// BIOS is the firmware identity (DMI type 0).
type BIOS struct {
	Vendor  string `json:"vendor,omitempty"`
	Version string `json:"version,omitempty"`
	Date    string `json:"date,omitempty"`
}

// Baseboard is the motherboard identity (DMI type 2).
type Baseboard struct {
	Vendor       string `json:"vendor,omitempty"`
	Product      string `json:"product,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	AssetTag     string `json:"asset_tag,omitempty"`
}

// Chassis is the enclosure identity (DMI type 3).
type Chassis struct {
	Vendor          string `json:"vendor,omitempty"`
	Type            string `json:"type,omitempty"`
	TypeDescription string `json:"type_description,omitempty"`
	Version         string `json:"version,omitempty"`
	SerialNumber    string `json:"serial_number,omitempty"`
	AssetTag        string `json:"asset_tag,omitempty"`
}

// Product is the system identity (DMI type 1). Product.Name is the
// primary signal cloud collectors key off — "Google Compute Engine",
// "Amazon EC2", "DigitalOcean Droplet", etc.
type Product struct {
	Vendor       string `json:"vendor,omitempty"`
	Name         string `json:"name,omitempty"`
	Family       string `json:"family,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	SKU          string `json:"sku,omitempty"`
}

// Collector is the public interface every dmi variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the shared identity for every OS variant.
type base struct{}

// Name returns "dmi".
func (base) Name() string { return "dmi" }

// Category returns "hardware".
func (base) Category() string { return collector.CategoryHardware }

// DefaultEnabled returns false — dmi is opt-in. Cloud collectors
// declare it as a dependency so enabling any cloud collector pulls
// dmi in automatically; users who explicitly want hardware identity
// can enable it directly.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the dmi collector variant appropriate to the detected host OS.
func New() Collector {
	if platform.IsDarwin() {
		return NewDarwin()
	}
	return NewLinux()
}
