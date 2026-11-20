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
	"path"
	"regexp"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pcidev"
)

// drmDir has a link for each DRM device; cardN links to its PCI device.
const drmDir = "/sys/class/drm"

// pciAddressRE is a PCI address in the path of a card's link.
var pciAddressRE = regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]$`)

// Linux lists the graphics cards of /sys/class/drm through FS: each
// cardN, its PCI address from where its link goes, and the vendor and
// model of that PCI device.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's sysfs.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the GPU Info.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	entries, err := l.FS.ReadDir(drmDir)
	if err != nil {
		return info, nil
	}
	var db *pcidev.DB
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, "card") || strings.Contains(n, "-") {
			continue
		}
		target, err := l.FS.Readlink(drmDir + "/" + n)
		if err != nil {
			continue
		}
		address := ""
		for _, part := range strings.Split(path.Clean(target), "/") {
			if pciAddressRE.MatchString(part) {
				address = part
			}
		}
		if address == "" {
			continue
		}
		if db == nil {
			db = pcidev.LoadDB(l.FS)
		}
		card := Card{Address: address}
		if d, ok := pcidev.Read(l.FS, db, address); ok {
			card.Vendor, card.VendorID = d.VendorName, d.VendorID
			card.Model, card.DeviceID = d.ProductName, d.ProductID
		}
		info.Cards = append(info.Cards, card)
	}

	return info, nil
}
