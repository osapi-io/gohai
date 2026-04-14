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
	"errors"
	"testing"

	"github.com/jaypipes/ghw/pkg/baseboard"
	"github.com/jaypipes/ghw/pkg/bios"
	"github.com/jaypipes/ghw/pkg/chassis"
	"github.com/jaypipes/ghw/pkg/product"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware/dmi"
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
		name   string
		detect string
	}{
		{"linux", "debian"},
		{"darwin", "darwin"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			orig := platform.Detect
			platform.Detect = func() string { return tt.detect }
			defer func() { platform.Detect = orig }()

			c := dmi.New()
			s.Equal("dmi", c.Name())
			s.False(c.DefaultEnabled())
			s.Nil(c.Dependencies())
		})
	}
}

func (s *DmiPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		biosFn  func(...any) (*bios.Info, error)
		bbFn    func(...any) (*baseboard.Info, error)
		chFn    func(...any) (*chassis.Info, error)
		prodFn  func(...any) (*product.Info, error)
		verify  func(s *DmiPublicTestSuite, info *dmi.Info)
	}{
		{
			name:    "linux populates all sections when ghw succeeds",
			variant: "linux",
			biosFn: func(...any) (*bios.Info, error) {
				return &bios.Info{Vendor: "SeaBIOS", Version: "1.16.2", Date: "04/01/2014"}, nil
			},
			bbFn: func(...any) (*baseboard.Info, error) {
				return &baseboard.Info{
					Vendor: "Google", Product: "Google Compute Engine",
					Version: "", SerialNumber: "Board-GoogleCloud", AssetTag: "",
				}, nil
			},
			chFn: func(...any) (*chassis.Info, error) {
				return &chassis.Info{
					Vendor: "Google", Type: "1", TypeDescription: "Other",
					Version: "", SerialNumber: "GoogleCloud-1234", AssetTag: "",
				}, nil
			},
			prodFn: func(...any) (*product.Info, error) {
				return &product.Info{
					Vendor: "Google", Name: "Google Compute Engine",
					Family: "", Version: "",
					SerialNumber: "GoogleCloud-1234",
					UUID:         "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
					SKU:          "",
				}, nil
			},
			verify: func(s *DmiPublicTestSuite, info *dmi.Info) {
				s.Require().NotNil(info)
				s.Require().NotNil(info.BIOS)
				s.Equal("SeaBIOS", info.BIOS.Vendor)
				s.Equal("1.16.2", info.BIOS.Version)
				s.Equal("04/01/2014", info.BIOS.Date)

				s.Require().NotNil(info.Baseboard)
				s.Equal("Google", info.Baseboard.Vendor)
				s.Equal("Google Compute Engine", info.Baseboard.Product)

				s.Require().NotNil(info.Chassis)
				s.Equal("Google", info.Chassis.Vendor)
				s.Equal("Other", info.Chassis.TypeDescription)

				s.Require().NotNil(info.Product)
				s.Equal("Google", info.Product.Vendor)
				s.Equal("Google Compute Engine", info.Product.Name)
				s.Equal("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", info.Product.UUID)
			},
		},
		{
			name:    "linux drops sections whose ghw call errors",
			variant: "linux",
			biosFn: func(...any) (*bios.Info, error) {
				return nil, errors.New("permission denied")
			},
			bbFn: func(...any) (*baseboard.Info, error) {
				return &baseboard.Info{Vendor: "Dell Inc."}, nil
			},
			chFn: func(...any) (*chassis.Info, error) { return nil, errors.New("nope") },
			prodFn: func(...any) (*product.Info, error) {
				return &product.Info{Name: "OptiPlex 3070"}, nil
			},
			verify: func(s *DmiPublicTestSuite, info *dmi.Info) {
				s.Require().NotNil(info)
				s.Nil(info.BIOS)
				s.Nil(info.Chassis)
				s.Require().NotNil(info.Baseboard)
				s.Equal("Dell Inc.", info.Baseboard.Vendor)
				s.Require().NotNil(info.Product)
				s.Equal("OptiPlex 3070", info.Product.Name)
			},
		},
		{
			name:    "linux drops sections when ghw returns nil",
			variant: "linux",
			biosFn:  func(...any) (*bios.Info, error) { return nil, nil },
			bbFn:    func(...any) (*baseboard.Info, error) { return nil, nil },
			chFn:    func(...any) (*chassis.Info, error) { return nil, nil },
			prodFn:  func(...any) (*product.Info, error) { return nil, nil },
			verify: func(s *DmiPublicTestSuite, info *dmi.Info) {
				s.Require().NotNil(info)
				s.Nil(info.BIOS)
				s.Nil(info.Baseboard)
				s.Nil(info.Chassis)
				s.Nil(info.Product)
			},
		},
		{
			name:    "darwin returns empty info",
			variant: "darwin",
			verify: func(s *DmiPublicTestSuite, info *dmi.Info) {
				s.Require().NotNil(info)
				s.Nil(info.BIOS)
				s.Nil(info.Baseboard)
				s.Nil(info.Chassis)
				s.Nil(info.Product)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c dmi.Collector
			switch tt.variant {
			case "linux":
				defer dmi.SetBIOSFn(tt.biosFn)()
				defer dmi.SetBaseboardFn(tt.bbFn)()
				defer dmi.SetChassisFn(tt.chFn)()
				defer dmi.SetProductFn(tt.prodFn)()
				c = dmi.NewLinux()
			case "darwin":
				c = dmi.NewDarwin()
			}

			out, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := out.(*dmi.Info)
			s.Require().True(ok)
			tt.verify(s, info)
		})
	}
}
