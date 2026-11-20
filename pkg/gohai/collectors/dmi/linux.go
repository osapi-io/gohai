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
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

const (
	// dmiDir has a file for each DMI string the kernel exports.
	dmiDir = "/sys/class/dmi/id"
	// deviceTreeDir is the device tree of a board with no DMI, such as an
	// ARM single-board computer.
	deviceTreeDir = "/sys/firmware/devicetree/base"
	// unknown is the value of a field that the host does not have.
	unknown = "unknown"
)

// Linux reads the DMI strings from /sys/class/dmi/id through FS. A board
// with no DMI but a device tree gets what the tree has. Each missing
// field is "unknown".
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's sysfs.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the DMI Info.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	if !l.isDir(dmiDir) && l.isDir(deviceTreeDir) {
		return l.deviceTree(), nil
	}
	item := func(name string) string { return l.read(dmiDir+"/"+name, unknown) }
	chassisType := item("chassis_type")

	return &Info{
		BIOS: &BIOS{
			Manufacturer: item("bios_vendor"),
			Ver:          item("bios_version"),
			Date:         item("bios_date"),
		},
		Baseboard: &Baseboard{
			Vendor:       item("board_vendor"),
			Product:      item("board_name"),
			Version:      item("board_version"),
			SerialNumber: item("board_serial"),
			AssetTag:     item("board_asset_tag"),
		},
		Chassis: &Chassis{
			Vendor:          item("chassis_vendor"),
			Type:            chassisType,
			TypeDescription: chassisTypeDescription(chassisType),
			Version:         item("chassis_version"),
			SerialNumber:    item("chassis_serial"),
			AssetTag:        item("chassis_asset_tag"),
		},
		Product: &Product{
			VendorName:   item("sys_vendor"),
			Name:         item("product_name"),
			Family:       item("product_family"),
			Version:      item("product_version"),
			SerialNumber: item("product_serial"),
			UUID:         item("product_uuid"),
			SKU:          item("product_sku"),
		},
	}, nil
}

// deviceTree is the Info of a board with a device tree: the model, the
// serial number, the vendor of the first "compatible" entry, the chassis
// type, and U-Boot as the firmware when it booted the board.
func (l *Linux) deviceTree() *Info {
	prop := func(name string) string {
		return strings.TrimSpace(strings.Trim(l.read(deviceTreeDir+"/"+name, unknown), "\x00"))
	}
	model, serial := prop("model"), prop("serial-number")
	vendor := unknown
	compatible := strings.Split(l.read(deviceTreeDir+"/compatible", ""), "\x00")
	if v, _, _ := strings.Cut(strings.TrimSpace(compatible[0]), ","); v != "" {
		vendor = v
	}
	chassisType := unknown
	if code, ok := deviceTreeChassisTypes[prop("chassis-type")]; ok {
		chassisType = code
	}
	bios := &BIOS{Manufacturer: unknown, Ver: unknown, Date: unknown}
	if v := prop("chosen/u-boot,version"); v != unknown {
		bios.Manufacturer, bios.Ver = "U-Boot", v
	}

	return &Info{
		BIOS: bios,
		Baseboard: &Baseboard{
			Vendor: vendor, Product: model, Version: unknown,
			SerialNumber: serial, AssetTag: unknown,
		},
		Chassis: &Chassis{
			Vendor: vendor, Type: chassisType,
			TypeDescription: chassisTypeDescription(chassisType),
			Version:         model, SerialNumber: serial, AssetTag: unknown,
		},
		Product: &Product{
			VendorName: vendor, Name: model, Family: unknown, Version: unknown,
			SerialNumber: serial, UUID: unknown, SKU: unknown,
		},
	}
}

// read is the trimmed content of path, or def when it cannot be read.
func (l *Linux) read(
	path string,
	def string,
) string {
	b, err := l.FS.ReadFile(path)
	if err != nil {
		return def
	}

	return strings.TrimSpace(string(b))
}

// isDir reports whether path is a directory.
func (l *Linux) isDir(
	path string,
) bool {
	fi, err := l.FS.Stat(path)

	return err == nil && fi.IsDir()
}

// chassisTypeDescription is the SMBIOS name of a chassis type code.
func chassisTypeDescription(
	code string,
) string {
	if d, ok := chassisTypes[code]; ok {
		return d
	}

	return unknown
}

// chassisTypes are the SMBIOS chassis types (DMTF DSP0134, 7.4.1).
var chassisTypes = map[string]string{
	"1": "Other", "2": "Unknown", "3": "Desktop", "4": "Low profile desktop",
	"5": "Pizza box", "6": "Mini tower", "7": "Tower", "8": "Portable",
	"9": "Laptop", "10": "Notebook", "11": "Hand held", "12": "Docking station",
	"13": "All in one", "14": "Sub notebook", "15": "Space-saving",
	"16": "Lunch box", "17": "Main server chassis", "18": "Expansion chassis",
	"19": "SubChassis", "20": "Bus Expansion chassis", "21": "Peripheral chassis",
	"22": "RAID chassis", "23": "Rack mount chassis", "24": "Sealed-case PC",
	"25": "Multi-system chassis", "26": "Compact PCI", "27": "Advanced TCA",
	"28": "Blade", "29": "Blade enclosure", "30": "Tablet", "31": "Convertible",
	"32": "Detachable", "33": "IoT gateway", "34": "Embedded PC", "35": "Mini PC",
	"36": "Stick PC",
}

// deviceTreeChassisTypes are the SMBIOS codes of the device tree's
// chassis-type values.
var deviceTreeChassisTypes = map[string]string{
	"desktop": "3", "laptop": "9", "convertible": "31", "server": "17",
	"tablet": "30", "handset": "11", "watch": "1", "embedded": "34",
	"all-in-one": "13",
}
