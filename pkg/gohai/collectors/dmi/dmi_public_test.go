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

package dmi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

type DmiPublicTestSuite struct {
	suite.Suite
}

func TestDmiPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DmiPublicTestSuite))
}

func (s *DmiPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		validateFunc func(dmi.Collector)
	}{
		{
			name:   "linux",
			detect: "debian",
			validateFunc: func(c dmi.Collector) {
				_, ok := c.(*dmi.Linux)
				s.True(ok)
			},
		},
		{
			name:   "darwin",
			detect: "darwin",
			validateFunc: func(c dmi.Collector) {
				_, ok := c.(*dmi.Darwin)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			orig := platform.Detect
			platform.Detect = func() string { return tt.detect }
			defer func() { platform.Detect = orig }()

			c := dmi.New()
			s.Equal("dmi", c.Name())
			s.Equal("hardware", c.Category())
			s.False(c.DefaultEnabled())
			s.Nil(c.Dependencies())

			tt.validateFunc(c)
		})
	}
}

func (s *DmiPublicTestSuite) TestCollect() {
	dmiFiles := map[string]string{
		"/sys/class/dmi/id/bios_vendor":       "LENOVO\n",
		"/sys/class/dmi/id/bios_version":      "N2VET36W\n",
		"/sys/class/dmi/id/bios_date":         "01/01/2024\n",
		"/sys/class/dmi/id/board_vendor":      "LENOVO\n",
		"/sys/class/dmi/id/board_name":        "21DC\n",
		"/sys/class/dmi/id/board_version":     "SDK0\n",
		"/sys/class/dmi/id/board_serial":      "L1HF\n",
		"/sys/class/dmi/id/board_asset_tag":   "Not Available\n",
		"/sys/class/dmi/id/chassis_vendor":    "LENOVO\n",
		"/sys/class/dmi/id/chassis_type":      "10\n",
		"/sys/class/dmi/id/chassis_version":   "None\n",
		"/sys/class/dmi/id/chassis_serial":    "PF3X\n",
		"/sys/class/dmi/id/chassis_asset_tag": "No Asset\n",
		"/sys/class/dmi/id/sys_vendor":        "LENOVO\n",
		"/sys/class/dmi/id/product_name":      "21DC0040US\n",
		"/sys/class/dmi/id/product_family":    "ThinkPad X1\n",
		"/sys/class/dmi/id/product_version":   "ThinkPad X1 Carbon Gen 10\n",
		"/sys/class/dmi/id/product_uuid":      "4c4c4544-0042\n",
		"/sys/class/dmi/id/product_sku":       "LENOVO_MT_21DC\n",
	}
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		want    *dmi.Info
	}{
		{
			name:    "linux: each DMI string; an unreadable one is unknown",
			variant: "linux",
			files:   dmiFiles,
			want: &dmi.Info{
				BIOS: &dmi.BIOS{Manufacturer: "LENOVO", Ver: "N2VET36W", Date: "01/01/2024"},
				Baseboard: &dmi.Baseboard{
					Vendor: "LENOVO", Product: "21DC", Version: "SDK0",
					SerialNumber: "L1HF", AssetTag: "Not Available",
				},
				Chassis: &dmi.Chassis{
					Vendor: "LENOVO", Type: "10", TypeDescription: "Notebook",
					Version: "None", SerialNumber: "PF3X", AssetTag: "No Asset",
				},
				Product: &dmi.Product{
					VendorName: "LENOVO", Name: "21DC0040US", Family: "ThinkPad X1",
					Version: "ThinkPad X1 Carbon Gen 10", SerialNumber: "unknown",
					UUID: "4c4c4544-0042", SKU: "LENOVO_MT_21DC",
				},
			},
		},
		{
			name:    "linux: a device tree board with U-Boot",
			variant: "linux",
			files: map[string]string{
				"/sys/firmware/devicetree/base/model":                 "Raspberry Pi 4 Model B Rev 1.4\x00",
				"/sys/firmware/devicetree/base/serial-number":         "10000000abcd\x00",
				"/sys/firmware/devicetree/base/compatible":            "raspberrypi,4-model-b\x00brcm,bcm2711\x00",
				"/sys/firmware/devicetree/base/chassis-type":          "embedded\x00",
				"/sys/firmware/devicetree/base/chosen/u-boot,version": "U-Boot 2024.01\x00",
			},
			want: &dmi.Info{
				BIOS: &dmi.BIOS{Manufacturer: "U-Boot", Ver: "U-Boot 2024.01", Date: "unknown"},
				Baseboard: &dmi.Baseboard{
					Vendor: "raspberrypi", Product: "Raspberry Pi 4 Model B Rev 1.4",
					Version: "unknown", SerialNumber: "10000000abcd", AssetTag: "unknown",
				},
				Chassis: &dmi.Chassis{
					Vendor: "raspberrypi", Type: "34", TypeDescription: "Embedded PC",
					Version: "Raspberry Pi 4 Model B Rev 1.4", SerialNumber: "10000000abcd",
					AssetTag: "unknown",
				},
				Product: &dmi.Product{
					VendorName: "raspberrypi", Name: "Raspberry Pi 4 Model B Rev 1.4",
					Family: "unknown", Version: "unknown", SerialNumber: "10000000abcd",
					UUID: "unknown", SKU: "unknown",
				},
			},
		},
		{
			name:    "linux: a device tree with nothing but its directory",
			variant: "linux",
			files:   map[string]string{"/sys/firmware/devicetree/base/.keep": ""},
			want: &dmi.Info{
				BIOS: &dmi.BIOS{Manufacturer: "unknown", Ver: "unknown", Date: "unknown"},
				Baseboard: &dmi.Baseboard{
					Vendor: "unknown", Product: "unknown", Version: "unknown",
					SerialNumber: "unknown", AssetTag: "unknown",
				},
				Chassis: &dmi.Chassis{
					Vendor: "unknown", Type: "unknown", TypeDescription: "unknown",
					Version: "unknown", SerialNumber: "unknown", AssetTag: "unknown",
				},
				Product: &dmi.Product{
					VendorName: "unknown", Name: "unknown", Family: "unknown", Version: "unknown",
					SerialNumber: "unknown", UUID: "unknown", SKU: "unknown",
				},
			},
		},
		{
			name:    "darwin: no DMI",
			variant: "darwin",
			want:    &dmi.Info{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c dmi.Collector = &dmi.Darwin{}
			if tt.variant == "linux" {
				c = &dmi.Linux{FS: sourcetest.FS(s.T(), tt.files)}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
