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

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
	"github.com/jaypipes/ghw/pkg/pci"
	"github.com/jaypipes/pcidb"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gpu"
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

// displayExec returns a MockExecutor that returns (out, err) when
// system_profiler SPDisplaysDataType -json is invoked.
func displayExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
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
			c := gpu.New()
			s.Equal("gpu", c.Name())
			s.Equal("hardware", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*gpu.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*gpu.Linux)
				s.True(ok)
			}
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
	ghwPopulated := func(...any) (*ghwgpu.Info, error) {
		return &ghwgpu.Info{
			GraphicsCards: []*ghwgpu.GraphicsCard{
				{
					Address: "0000:03:00.0",
					DeviceInfo: &pci.Device{
						Vendor:  &pcidb.Vendor{ID: "10de", Name: "NVIDIA Corporation"},
						Product: &pcidb.Product{ID: "1c82", Name: "GP107 [GeForce GTX 1050 Ti]"},
					},
				},
				{Address: "0000:04:00.0"}, // no DeviceInfo
			},
		}, nil
	}
	ghwErr := func(...any) (*ghwgpu.Info, error) { return nil, errors.New("no drm") }
	ghwNil := func(...any) (*ghwgpu.Info, error) { return nil, nil }

	tests := []struct {
		name     string
		variant  string
		ghw      func(...any) (*ghwgpu.Info, error)
		exec     func(*testing.T) executor.Executor
		validate func(*gpu.Info)
	}{
		{
			name:    "linux: ghw returns cards with + without DeviceInfo",
			variant: "linux",
			ghw:     ghwPopulated,
			validate: func(i *gpu.Info) {
				s.Require().Len(i.Cards, 2)
				s.Equal("NVIDIA Corporation", i.Cards[0].Vendor)
				s.Equal("10de", i.Cards[0].VendorID)
				s.Equal("1c82", i.Cards[0].DeviceID)
				s.Equal("GP107 [GeForce GTX 1050 Ti]", i.Cards[0].Model)
				s.Equal("0000:03:00.0", i.Cards[0].Address)
				s.Empty(i.Cards[1].Vendor)
				s.Equal("0000:04:00.0", i.Cards[1].Address)
			},
		},
		{
			name:     "linux: ghw error yields empty Info",
			variant:  "linux",
			ghw:      ghwErr,
			validate: func(i *gpu.Info) { s.Empty(i.Cards) },
		},
		{
			name:     "linux: ghw nil Info yields empty",
			variant:  "linux",
			ghw:      ghwNil,
			validate: func(i *gpu.Info) { s.Empty(i.Cards) },
		},
		{
			name:    "darwin: system_profiler JSON parsed for Apple GPU + discrete",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return displayExec(t, []byte(darwinJSON), nil)
			},
			validate: func(i *gpu.Info) {
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
			exec: func(t *testing.T) executor.Executor {
				return displayExec(
					t,
					[]byte(
						`{"SPDisplaysDataType":[{"_name":"Fallback","spdisplays_vendor":"Intel"}]}`,
					),
					nil,
				)
			},
			validate: func(i *gpu.Info) {
				s.Require().Len(i.Cards, 1)
				s.Equal("Fallback", i.Cards[0].Model)
				s.Equal("Intel", i.Cards[0].Vendor)
			},
		},
		{
			name:    "darwin: exec error yields empty Info",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return displayExec(t, nil, errors.New("not found"))
			},
			validate: func(i *gpu.Info) { s.Empty(i.Cards) },
		},
		{
			name:    "darwin: malformed JSON yields empty Info",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return displayExec(t, []byte("not json"), nil)
			},
			validate: func(i *gpu.Info) { s.Empty(i.Cards) },
		},
		{
			name:     "darwin: nil Exec yields empty",
			variant:  "darwin",
			exec:     func(*testing.T) executor.Executor { return nil },
			validate: func(i *gpu.Info) { s.Empty(i.Cards) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c gpu.Collector
			switch tt.variant {
			case "linux":
				defer gpu.SetGHWGPUFn(tt.ghw)()
				c = &gpu.Linux{}
			case "darwin":
				d := &gpu.Darwin{}
				if tt.exec != nil {
					d.Exec = tt.exec(s.T())
				}
				c = d
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*gpu.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
