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

package pcidev_test

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/pcidev"
	"github.com/osapi-io/gohai/internal/sourcetest"
)

type PCIDevPublicTestSuite struct {
	suite.Suite
}

func TestPCIDevPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PCIDevPublicTestSuite))
}

// ids is a pci.ids with one vendor, product and subsystem, and one class.
const ids = `# pci.ids
10de  NVIDIA Corporation
	1c82  GP107 [GeForce GTX 1050 Ti]
		1458 3764  GV-N105TOC-4GD
no two spaces here
C 03  Display controller
	00  VGA compatible controller
		00  VGA controller
`

// gpuModalias is the modalias of the NVIDIA card of ids.
const gpuModalias = "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00\n"

func (s *PCIDevPublicTestSuite) TestList() {
	gz := func(b string) string {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(b))
		_ = zw.Close()
		return buf.String()
	}
	dev := "/sys/bus/pci/devices/"
	tests := []struct {
		name         string
		files        map[string]string
		links        map[string]string
		validateFunc func([]pcidev.Device)
	}{
		{
			name: "a named device with its driver, IOMMU group and parent",
			files: map[string]string{
				"/usr/share/hwdata/pci.ids":                              ids,
				"/devices/pci0000:00/0000:00:01.0/0000:01:00.0/modalias": gpuModalias,
				"/devices/pci0000:00/0000:00:01.0/0000:01:00.0/revision": "0xa1\n",
			},
			links: map[string]string{
				dev + "0000:01:00.0":             "/devices/pci0000:00/0000:00:01.0/0000:01:00.0",
				dev + "0000:01:00.0/driver":      "../../../bus/pci/drivers/nvidia",
				dev + "0000:01:00.0/iommu_group": "../../../kernel/iommu_groups/14",
			},
			validateFunc: func(d []pcidev.Device) {
				s.Require().Len(d, 1)
				s.Equal(pcidev.Device{
					Address: "0000:01:00.0", VendorID: "10de", VendorName: "NVIDIA Corporation",
					ProductID: "1c82", ProductName: "GP107 [GeForce GTX 1050 Ti]",
					ClassID: "03", ClassName: "Display controller",
					SubclassID: "00", SubclassName: "VGA compatible controller",
					SubsystemVendorID: "1458", SubsystemID: "3764", SubsystemName: "GV-N105TOC-4GD",
					Revision: "0xa1", Driver: "nvidia", IOMMUGroup: "14",
					ParentAddress: "0000:00:01.0",
				}, d[0])
			},
		},
		{
			name: "a gzip pci.ids, an unknown device, a root-bus parent, and entries left out",
			files: map[string]string{
				"/usr/share/misc/pci.ids.gz":    gz(ids),
				dev + "0000:00:02.0/modalias":   "pci:v00008086d00009A49sv00001028sd00000A38bc03sc80i00\n",
				dev + "0000:00:03.0/modalias":   "pci:short\n",
				dev + "0000:00:00.0/modalias":   gpuModalias,
				dev + "0000:00:04.0/.keep":      "",
				dev + "not-an-address/modalias": gpuModalias,
			},
			validateFunc: func(d []pcidev.Device) {
				s.Require().Len(d, 2)
				s.Equal("0000:00:00.0", d[0].Address)
				s.Equal("NVIDIA Corporation", d[0].VendorName)
				s.Equal("unknown", d[1].VendorName)
				s.Equal("unknown", d[1].ProductName)
				s.Equal("Display controller", d[1].ClassName)
				s.Equal("unknown", d[1].SubclassName)
				s.Equal("unknown", d[1].SubsystemName)
				s.Empty(d[1].Driver)
				s.Empty(d[1].ParentAddress)
			},
		},
		{
			name: "a pci.ids line longer than the read buffer is skipped, and the parse goes on",
			files: map[string]string{
				"/usr/share/hwdata/pci.ids": "8086  " + strings.Repeat(
					"x",
					2*1024*1024,
				) + "\n" + ids,
				dev + "0000:01:00.0/modalias": gpuModalias,
			},
			validateFunc: func(d []pcidev.Device) {
				s.Require().Len(d, 1)
				s.Equal("NVIDIA Corporation", d[0].VendorName)
				s.Equal("GV-N105TOC-4GD", d[0].SubsystemName)
			},
		},
		{
			name: "a pci.ids.gz that is not gzip, and no other: names unknown",
			files: map[string]string{
				"/usr/share/hwdata/pci.ids.gz": "plain text",
				dev + "0000:01:00.0/modalias":  gpuModalias,
			},
			validateFunc: func(d []pcidev.Device) {
				s.Require().Len(d, 1)
				s.Equal("unknown", d[0].VendorName)
			},
		},
		{
			name: "a pci.ids.gz that is cut short",
			files: map[string]string{
				"/usr/share/hwdata/pci.ids.gz": gz(ids)[:30],
				dev + "0000:01:00.0/modalias":  gpuModalias,
			},
			validateFunc: func(d []pcidev.Device) {
				s.Require().Len(d, 1)
				s.Equal("unknown", d[0].VendorName)
			},
		},
		{
			name:         "no /sys/bus/pci",
			validateFunc: func(d []pcidev.Device) { s.Empty(d) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.Links(s.T(), sourcetest.FS(s.T(), tt.files), tt.links)
			tt.validateFunc(pcidev.List(fs, pcidev.LoadDB(fs)))
		})
	}
}

func (s *PCIDevPublicTestSuite) TestRead() {
	tests := []struct {
		name         string
		files        map[string]string
		address      string
		validateFunc func(pcidev.Device, bool)
	}{
		{
			name:    "a device with no modalias is not read",
			address: "0000:09:00.0",
			validateFunc: func(_ pcidev.Device, ok bool) {
				s.False(ok)
			},
		},
		{
			name: "a device with a modalias is read, unnamed with no pci.ids",
			files: map[string]string{
				"/sys/bus/pci/devices/0000:01:00.0/modalias": gpuModalias,
			},
			address: "0000:01:00.0",
			validateFunc: func(d pcidev.Device, ok bool) {
				s.Require().True(ok)
				s.Equal("10de", d.VendorID)
				s.Equal("unknown", d.VendorName)
				s.Equal("1c82", d.ProductID)
				s.Equal("1458", d.SubsystemVendorID)
				s.Equal("3764", d.SubsystemID)
				s.Equal("03", d.ClassID)
				s.Equal("00", d.SubclassID)
			},
		},
		{
			name: "a modalias of PCI length that is not PCI is not read",
			files: map[string]string{
				"/sys/bus/pci/devices/0000:01:00.0/modalias": strings.Repeat("x", 60),
			},
			address: "0000:01:00.0",
			validateFunc: func(_ pcidev.Device, ok bool) {
				s.False(ok)
			},
		},
		{
			name: "a PCI modalias with a non-hex ID is not read",
			files: map[string]string{
				"/sys/bus/pci/devices/0000:01:00.0/modalias": "pci:v0000zzzzd00001C82sv00001458sd00003764bc03sc00i00\n",
			},
			address: "0000:01:00.0",
			validateFunc: func(_ pcidev.Device, ok bool) {
				s.False(ok)
			},
		},
		{
			name: "a modalias with text after the PCI fields is not read",
			files: map[string]string{
				"/sys/bus/pci/devices/0000:01:00.0/modalias": "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00junk\n",
			},
			address: "0000:01:00.0",
			validateFunc: func(_ pcidev.Device, ok bool) {
				s.False(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			tt.validateFunc(pcidev.Read(fs, pcidev.ParseIDs(nil), tt.address))
		})
	}
}
