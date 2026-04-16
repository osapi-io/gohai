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

package pci_test

import (
	"context"
	"errors"
	"testing"

	ghwpci "github.com/jaypipes/ghw/pkg/pci"
	"github.com/jaypipes/pcidb"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
)

var (
	_ collector.Collector = (*pci.Linux)(nil)
	_ collector.Collector = (*pci.Darwin)(nil)
)

type PCIPublicTestSuite struct {
	suite.Suite
}

func TestPCIPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PCIPublicTestSuite))
}

func (s *PCIPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := pci.New()
			s.Equal("pci", c.Name())
			s.Equal("hardware", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*pci.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*pci.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *PCIPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		pciFn    func(...any) (*ghwpci.Info, error)
		validate func(*pci.Info)
	}{
		{
			name:    "linux: happy path maps ghw devices",
			variant: "linux",
			pciFn: func(...any) (*ghwpci.Info, error) {
				return &ghwpci.Info{
					Devices: []*ghwpci.Device{
						{
							Address:       "0000:03:00.0",
							ParentAddress: "0000:00:1c.0",
							Revision:      "0x01",
							Driver:        "iwlwifi",
							IOMMUGroup:    "12",
							Vendor:        &pcidb.Vendor{ID: "8086", Name: "Intel Corporation"},
							Product:       &pcidb.Product{ID: "24fd", Name: "Wireless 8265 / 8275"},
							Class:         &pcidb.Class{ID: "02", Name: "Network controller"},
							Subclass:      &pcidb.Subclass{ID: "80", Name: "Network controller"},
							Subsystem: &pcidb.Product{
								ID:   "0130",
								Name: "Dual Band Wireless-AC 8265",
							},
						},
					},
				}, nil
			},
			validate: func(info *pci.Info) {
				s.Require().Len(info.Devices, 1)
				d := info.Devices["0000:03:00.0"]
				s.Equal("8086", d.VendorID)
				s.Equal("Intel Corporation", d.VendorName)
				s.Equal("24fd", d.DeviceID)
				s.Equal("Wireless 8265 / 8275", d.DeviceName)
				s.Equal("02", d.ClassID)
				s.Equal("Network controller", d.ClassName)
				s.Equal("80", d.SubclassID)
				s.Equal("0130", d.SubsystemID)
				s.Equal("Dual Band Wireless-AC 8265", d.SubsystemName)
				s.Equal("0x01", d.Revision)
				s.Equal("iwlwifi", d.Driver)
				s.Equal("12", d.IOMMUGroup)
				s.Equal("0000:00:1c.0", d.ParentAddress)
			},
		},
		{
			name:    "linux: ghw error yields empty info, no error",
			variant: "linux",
			pciFn: func(...any) (*ghwpci.Info, error) {
				return nil, errors.New("no /sys/bus/pci")
			},
			validate: func(info *pci.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "linux: ghw nil info yields empty",
			variant: "linux",
			pciFn: func(...any) (*ghwpci.Info, error) {
				return nil, nil
			},
			validate: func(info *pci.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "linux: devices with nil/empty fields tolerated",
			variant: "linux",
			pciFn: func(...any) (*ghwpci.Info, error) {
				return &ghwpci.Info{
					Devices: []*ghwpci.Device{
						nil,           // skipped
						{Address: ""}, // skipped
						{Address: "0000:00:00.0", Driver: "unknown"}, // driver normalized to empty
						{
							Address: "0000:01:00.0",
							Vendor:  &pcidb.Vendor{ID: "10de", Name: "NVIDIA Corporation"},
						},
					},
				}, nil
			},
			validate: func(info *pci.Info) {
				s.Require().Len(info.Devices, 2)
				s.Equal("", info.Devices["0000:00:00.0"].Driver)
				s.Equal("10de", info.Devices["0000:01:00.0"].VendorID)
				s.Empty(info.Devices["0000:01:00.0"].DeviceID)
				s.Empty(info.Devices["0000:01:00.0"].ClassID)
			},
		},
		{
			name:    "darwin: returns empty",
			variant: "darwin",
			validate: func(info *pci.Info) {
				s.Require().NotNil(info)
				s.Empty(info.Devices)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c pci.Collector
			switch tt.variant {
			case "linux":
				defer pci.SetGHWPCIFn(tt.pciFn)()
				c = &pci.Linux{}
			case "darwin":
				c = &pci.Darwin{}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*pci.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
