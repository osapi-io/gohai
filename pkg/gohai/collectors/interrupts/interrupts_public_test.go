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

package interrupts_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
)

var (
	_ collector.Collector = (*interrupts.Linux)(nil)
	_ collector.Collector = (*interrupts.Darwin)(nil)
)

// errorFS forces a non-ErrNotExist read error.
type errorFS struct {
	avfs.VFS
}

func (errorFS) ReadFile(
	string,
) ([]byte, error) {
	return nil, errors.New("permission denied")
}

// twoCPUInterrupts is a typical 2-CPU /proc/interrupts excerpt containing
// numeric IRQs with full type/vector/device fields and non-numeric labels.
var twoCPUInterrupts = []byte(
	"           CPU0       CPU1\n" +
		"  0:         46          0   IO-APIC    2-edge      timer\n" +
		"  9:          0          0   ACPI   9-fasteoi   acpi\n" +
		"NMI:          0          0   Non-maskable interrupts\n" +
		"ERR:          0\n",
)

type InterruptsPublicTestSuite struct {
	suite.Suite
}

func TestInterruptsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InterruptsPublicTestSuite))
}

func (s *InterruptsPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := interrupts.New()
			s.Equal("interrupts", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*interrupts.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*interrupts.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *InterruptsPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() avfs.VFS
		wantErr bool
		wantNil bool
		want    []interrupts.IRQ
	}{
		{
			name:    "linux: two-cpu with numeric and non-numeric IRQs",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/interrupts", twoCPUInterrupts, fs.FileMode(0o444))
				return f
			},
			want: []interrupts.IRQ{
				{Number: "0", Type: "IO-APIC", Device: "timer", CountsPerCPU: []int64{46, 0}},
				{Number: "9", Type: "ACPI", Device: "acpi", CountsPerCPU: []int64{0, 0}},
				{Number: "NMI", Type: "Non-maskable interrupts", CountsPerCPU: []int64{0, 0}},
				{Number: "ERR", CountsPerCPU: []int64{0, 0}},
			},
		},
		{
			name:    "linux: empty /proc/interrupts yields empty list",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/interrupts", []byte{}, fs.FileMode(0o444))
				return f
			},
			want: []interrupts.IRQ{},
		},
		{
			name:    "linux: /proc/interrupts absent returns empty list",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			want:    []interrupts.IRQ{},
		},
		{
			name:    "linux: permission denied propagates error",
			variant: "linux",
			setupFS: func() avfs.VFS { return errorFS{memfs.New()} },
			wantErr: true,
		},
		{
			name:    "linux: invalid count field returns error",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/interrupts",
					[]byte("           CPU0\n  0:      BAD   IO-APIC   2-edge   timer\n"),
					fs.FileMode(0o444))
				return f
			},
			wantErr: true,
		},
		{
			name:    "linux: line without colon is skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile(
					"/proc/interrupts",
					[]byte(
						"           CPU0\n  0:        10   IO-APIC   2-edge   timer\nno colon here\n",
					),
					fs.FileMode(0o444),
				)
				return f
			},
			want: []interrupts.IRQ{
				{Number: "0", Type: "IO-APIC", Device: "timer", CountsPerCPU: []int64{10}},
			},
		},
		{
			name:    "linux: numeric IRQ with only type (no vector or device)",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/interrupts",
					[]byte("           CPU0\n  7:          0   IO-APIC\n"),
					fs.FileMode(0o444))
				return f
			},
			want: []interrupts.IRQ{
				{Number: "7", Type: "IO-APIC", CountsPerCPU: []int64{0}},
			},
		},
		{
			name:    "linux: numeric IRQ with type and vector but no device",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/interrupts",
					[]byte("           CPU0\n  8:          3   IO-APIC   8-edge\n"),
					fs.FileMode(0o444))
				return f
			},
			want: []interrupts.IRQ{
				{Number: "8", Type: "IO-APIC", CountsPerCPU: []int64{3}},
			},
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c interrupts.Collector
			switch tt.variant {
			case "linux":
				c = &interrupts.Linux{FS: tt.setupFS()}
			case "darwin":
				c = interrupts.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*interrupts.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.IRQs)
		})
	}
}
