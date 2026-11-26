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

package gpu_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gpu"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*gpu.Linux)(nil)
	_ collector.Collector = (*gpu.Darwin)(nil)
)

type GPUPublicTestSuite struct {
	suite.Suite
}

func TestGPUPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GPUPublicTestSuite))
}

// displayExec returns a MockRunner that returns (out, err) when
// system_profiler SPDisplaysDataType -json is invoked.
func displayExec(
	t *testing.T,
	out []byte,
	err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "system_profiler", "SPDisplaysDataType", "-json").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *GPUPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(gpu.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c gpu.Collector) {
				_, ok := c.(*gpu.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c gpu.Collector) {
				_, ok := c.(*gpu.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c gpu.Collector) {
				_, ok := c.(*gpu.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := gpu.New()
			s.Equal("gpu", c.Name())
			s.Equal("hardware", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

const darwinJSON = `{
  "SPDisplaysDataType": [
    {
      "_name": "Apple M1 Pro",
      "spdisplays_vendor": "sppci_vendor_Apple",
      "sppci_bus": "spdisplays_builtin",
      "sppci_cores": "16",
      "sppci_device_type": "spdisplays_gpu",
      "sppci_model": "Apple M1 Pro"
    },
    {
      "_name": "NVIDIA Card",
      "spdisplays_vendor": "sppci_vendor_NVIDIA",
      "sppci_bus": "spdisplays_pcie",
      "sppci_cores": "not-a-number",
      "sppci_model": "GeForce RTX 4090"
    }
  ]
}`

func (s *GPUPublicTestSuite) TestCollect() {
	const ids = "10de  NVIDIA Corporation\n\t1c82  GP107 [GeForce GTX 1050 Ti]\n"
	drm := "/sys/class/drm/"
	linuxFiles := map[string]string{
		"/usr/share/hwdata/pci.ids":                  ids,
		"/sys/bus/pci/devices/0000:03:00.0/modalias": "pci:v000010DEd00001C82sv00001458sd00003764bc03sc00i00\n",
		drm + "version":                              "drm 1.1.0\n",
	}
	linuxLinks := map[string]string{
		drm + "card0":          "../../devices/pci0000:00/0000:00:01.0/0000:03:00.0/drm/card0",
		drm + "card0-HDMI-A-1": "../../devices/pci0000:00/0000:00:01.0/0000:03:00.0/drm/card0/card0-HDMI-A-1",
		drm + "card1":          "../../devices/pci0000:00/0000:00:02.0/0000:04:00.0/drm/card1",
		drm + "card2":          "../../devices/platform/vc4/drm/card2",
		drm + "renderD128":     "../../devices/pci0000:00/0000:00:01.0/0000:03:00.0/drm/renderD128",
	}

	tests := []struct {
		name         string
		variant      string
		files        map[string]string
		links        map[string]string
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: each cardN with its PCI device; a card PCI has no record of is bare",
			variant: "linux",
			files:   linuxFiles,
			links:   linuxLinks,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Equal([]gpu.Card{
					{
						Vendor: "NVIDIA Corporation", Model: "GP107 [GeForce GTX 1050 Ti]",
						Address: "0000:03:00.0", VendorID: "10de", DeviceID: "1c82",
					},
					{Address: "0000:04:00.0"},
				}, i.Cards)
			},
		},
		{
			name:    "linux: a cardN that is not a link is skipped",
			variant: "linux",
			files:   map[string]string{drm + "card0/dev": "226:0\n"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Empty(i.Cards)
			},
		},
		{
			name:    "linux: no /sys/class/drm yields empty Info",
			variant: "linux",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Empty(i.Cards)
			},
		},
		{
			name:    "darwin: system_profiler JSON parsed for Apple GPU + discrete",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return displayExec(t, []byte(darwinJSON), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Require().Len(i.Cards, 2)
				s.Equal("Apple", i.Cards[0].Vendor)
				s.Equal("Apple M1 Pro", i.Cards[0].Model)
				s.Equal("builtin", i.Cards[0].Bus)
				s.Equal(16, i.Cards[0].Cores)
				s.Equal("NVIDIA", i.Cards[1].Vendor)
				s.Equal("GeForce RTX 4090", i.Cards[1].Model)
				s.Equal("pcie", i.Cards[1].Bus)
				s.Zero(i.Cards[1].Cores) // non-numeric Cores ignored
			},
		},
		{
			name:    "darwin: missing model falls back to _name",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return displayExec(
					t,
					[]byte(
						`{"SPDisplaysDataType":[{"_name":"Fallback","spdisplays_vendor":"Intel"}]}`,
					),
					nil,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Require().Len(i.Cards, 1)
				s.Equal("Fallback", i.Cards[0].Model)
				s.Equal("Intel", i.Cards[0].Vendor)
			},
		},
		{
			name:    "darwin: exec error yields empty Info",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return displayExec(t, nil, errors.New("not found"))
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Empty(i.Cards)
			},
		},
		{
			name:    "darwin: malformed JSON yields empty Info",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return displayExec(t, []byte("not json"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Empty(i.Cards)
			},
		},
		{
			name:    "darwin: nil Exec yields empty",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*gpu.Info)
				s.Require().True(ok)
				s.Empty(i.Cards)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c gpu.Collector
			switch tt.variant {
			case "linux":
				c = &gpu.Linux{
					FS: sourcetest.Links(s.T(), sourcetest.FS(s.T(), tt.files), tt.links),
				}
			case "darwin":
				d := &gpu.Darwin{}
				if tt.exec != nil {
					d.Exec = tt.exec(s.T())
				}
				c = d
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
