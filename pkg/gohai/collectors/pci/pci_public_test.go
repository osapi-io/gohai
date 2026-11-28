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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
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
		name         string
		detect       string
		validateFunc func(pci.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c pci.Collector) {
				_, ok := c.(*pci.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c pci.Collector) {
				_, ok := c.(*pci.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c pci.Collector) {
				_, ok := c.(*pci.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := pci.New()
			s.Equal("pci", c.Name())
			s.Equal("hardware", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PCIPublicTestSuite) TestCollect() {
	const ids = "10de  NVIDIA Corporation\n\t1c82  GP107\nC 03  Display controller\n\t00  VGA compatible controller\n"
	dev := "/sys/bus/pci/devices/"
	tests := []struct {
		name         string
		variant      string
		files        map[string]string
		links        map[string]string
		validateFunc func(any, error)
	}{
		{
			name:    "linux: each device by address; a driver named unknown is empty",
			variant: "linux",
			files: map[string]string{
				"/usr/share/hwdata/pci.ids":   ids,
				dev + "0000:01:00.0/modalias": "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00\n",
				dev + "0000:01:00.0/revision": "0xa1\n",
				dev + "0000:02:00.0/modalias": "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00\n",
			},
			links: map[string]string{
				dev + "0000:01:00.0/driver": "../../../bus/pci/drivers/nvidia",
				dev + "0000:02:00.0/driver": "../../../bus/pci/drivers/unknown",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&pci.Info{Devices: map[string]pci.Device{
					"0000:01:00.0": {
						VendorID: "10de", VendorName: "NVIDIA Corporation",
						DeviceID: "1c82", DeviceName: "GP107",
						ClassID: "03", ClassName: "Display controller",
						SubclassID: "00", SubclassName: "VGA compatible controller",
						SubsystemID: "3764", SubsystemName: "unknown",
						Revision: "0xa1", Driver: "nvidia",
					},
					"0000:02:00.0": {
						VendorID: "10de", VendorName: "NVIDIA Corporation",
						DeviceID: "1c82", DeviceName: "GP107",
						ClassID: "03", ClassName: "Display controller",
						SubclassID: "00", SubclassName: "VGA compatible controller",
						SubsystemID: "3764", SubsystemName: "unknown",
					},
				}}, got)
			},
		},
		{
			name:    "linux: no /sys/bus/pci",
			variant: "linux",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&pci.Info{Devices: map[string]pci.Device{}}, got)
			},
		},
		{
			name:    "darwin: no fact",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c pci.Collector = &pci.Darwin{}
			if tt.variant == "linux" {
				c = &pci.Linux{
					FS: sourcetest.Links(s.T(), sourcetest.FS(s.T(), tt.files), tt.links),
				}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
