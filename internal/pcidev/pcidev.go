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

// Package pcidev reads the PCI devices of a Linux host from sysfs, and
// names them from the pci.ids database, both through a filesystem. The
// pci and gpu collectors share it.
package pcidev

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/avfs/avfs"
)

const (
	// devicesDir has a link for each PCI address of the host.
	devicesDir = "/sys/bus/pci/devices"
	// unknown is the name of an ID that pci.ids does not have.
	unknown = "unknown"
)

// idsPaths are where distributions put pci.ids, in the order pcidb
// searched them.
var idsPaths = []string{
	"/usr/share/hwdata/pci.ids",
	"/usr/share/misc/pci.ids",
	"/usr/share/hwdata/pci.ids.gz",
	"/usr/share/misc/pci.ids.gz",
}

// addressRE is a PCI address: domain, bus, device and function.
var addressRE = regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]$`)

// Device is one PCI device. Each ID is lower-case hex; each name is from
// pci.ids, or "unknown".
type Device struct {
	Address       string
	VendorID      string
	VendorName    string
	ProductID     string
	ProductName   string
	ClassID       string
	ClassName     string
	SubclassID    string
	SubclassName  string
	SubsystemID   string
	SubsystemName string
	Revision      string
	Driver        string
	IOMMUGroup    string
	ParentAddress string
}

// DB is the names of pci.ids.
type DB struct {
	vendors    map[string]string // vendor
	products   map[string]string // vendor+product
	subsystems map[string]string // vendor+product+subvendor+subdevice
	classes    map[string]string // class
	subclasses map[string]string // class+subclass
}

// LoadDB reads the first pci.ids of the host. With none, every name is
// "unknown".
func LoadDB(
	fs avfs.VFS,
) *DB {
	for _, p := range idsPaths {
		b, err := fs.ReadFile(p)
		if err != nil {
			continue
		}
		if strings.HasSuffix(p, ".gz") {
			zr, err := gzip.NewReader(bytes.NewReader(b))
			if err != nil {
				continue
			}
			if b, err = io.ReadAll(zr); err != nil {
				continue
			}
		}

		return ParseIDs(b)
	}

	return ParseIDs(nil)
}

// ParseIDs parses the pci.ids format: a vendor line "10de  NVIDIA",
// a product under it "\t1c82  GP107", a subsystem under that
// "\t\t1458 3764  Name"; and a class line "C 03  Display controller" with
// its subclasses "\t00  VGA compatible controller".
func ParseIDs(
	b []byte,
) *DB {
	db := &DB{
		vendors: map[string]string{}, products: map[string]string{},
		subsystems: map[string]string{}, classes: map[string]string{},
		subclasses: map[string]string{},
	}
	var vendor, product, class string
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		depth := len(line) - len(strings.TrimLeft(line, "\t"))
		id, name, ok := strings.Cut(strings.TrimLeft(line, "\t"), "  ")
		if !ok {
			continue
		}
		id = strings.ToLower(id)
		switch {
		case depth == 0 && strings.HasPrefix(id, "c "):
			class, vendor = strings.TrimPrefix(id, "c "), ""
			db.classes[class] = name
		case depth == 0:
			vendor, product, class = id, "", ""
			db.vendors[vendor] = name
		case depth == 1 && class != "":
			db.subclasses[class+id] = name
		case depth == 1 && vendor != "":
			product = id
			db.products[vendor+product] = name
		case depth == 2 && vendor != "" && product != "":
			db.subsystems[vendor+product+strings.ReplaceAll(id, " ", "")] = name
		}
	}

	return db
}

// name is m[key], or "unknown".
func name(
	m map[string]string,
	key string,
) string {
	if n, ok := m[key]; ok {
		return n
	}

	return unknown
}

// List is each device of /sys/bus/pci/devices, in address order. An entry
// that is not a PCI address or has no modalias is left out.
func List(
	fs avfs.VFS,
	db *DB,
) []Device {
	entries, err := fs.ReadDir(devicesDir)
	if err != nil {
		return nil
	}
	var out []Device
	for _, e := range entries {
		if d, ok := Read(fs, db, e.Name()); ok {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })

	return out
}

// Read is the device at a PCI address. Its IDs come from the modalias,
// "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00".
func Read(
	fs avfs.VFS,
	db *DB,
	address string,
) (Device, bool) {
	if !addressRE.MatchString(address) {
		return Device{}, false
	}
	dir := devicesDir + "/" + address
	b, err := fs.ReadFile(dir + "/modalias")
	m := strings.ToLower(strings.TrimSpace(string(b)))
	if err != nil || len(m) < 53 {
		return Device{}, false
	}
	vendor, product := m[9:13], m[18:22]
	subvendor, subdevice := m[28:32], m[38:42]
	class, subclass := m[44:46], m[48:50]
	d := Device{
		Address:       address,
		VendorID:      vendor,
		VendorName:    name(db.vendors, vendor),
		ProductID:     product,
		ProductName:   name(db.products, vendor+product),
		ClassID:       class,
		ClassName:     name(db.classes, class),
		SubclassID:    subclass,
		SubclassName:  name(db.subclasses, class+subclass),
		SubsystemID:   subdevice,
		SubsystemName: name(db.subsystems, vendor+product+subvendor+subdevice),
	}
	if b, err := fs.ReadFile(dir + "/revision"); err == nil {
		d.Revision = strings.TrimSpace(string(b))
	}
	if t, err := fs.Readlink(dir + "/driver"); err == nil {
		d.Driver = path.Base(t)
	}
	if t, err := fs.Readlink(dir + "/iommu_group"); err == nil {
		d.IOMMUGroup = path.Base(t)
	}
	if t, err := fs.Readlink(dir); err == nil {
		if p := path.Base(path.Dir(t)); addressRE.MatchString(p) {
			d.ParentAddress = p
		}
	}

	return d, true
}
