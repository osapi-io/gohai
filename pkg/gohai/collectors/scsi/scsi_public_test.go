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

package scsi_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scsi"
)

var (
	_ collector.Collector = (*scsi.Linux)(nil)
	_ collector.Collector = (*scsi.Darwin)(nil)
)

type SCSIPublicTestSuite struct {
	suite.Suite
}

func TestSCSIPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SCSIPublicTestSuite))
}

// lsscsiExec returns a MockExecutor that returns (out, err) when
// `lsscsi` is invoked.
func lsscsiExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsscsi").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *SCSIPublicTestSuite) TestNew() {
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
			c := scsi.New()
			s.Equal("scsi", c.Name())
			s.Equal("hardware", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*scsi.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*scsi.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *SCSIPublicTestSuite) TestCollect() {
	const lsscsiOut = `[0:0:0:0]    disk    ATA      ST500DM002-1BD14 KC48  /dev/sda
[5:0:0:0]    disk    LSI      MR9240-8i        2.13  /dev/sdb
[6:0:0:0]    cd/dvd  NECVMWar VMware IDE CDR10 1.00  /dev/sr0

malformed
[7:0:0:0]   only  three  fields
`

	tests := []struct {
		name     string
		variant  string
		exec     func(*testing.T) executor.Executor
		validate func(*scsi.Info)
	}{
		{
			name:    "linux: happy path parses lsscsi output",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsscsiExec(t, []byte(lsscsiOut), nil)
			},
			validate: func(info *scsi.Info) {
				s.Require().Len(info.Devices, 3)

				d := info.Devices["0:0:0:0"]
				s.Equal("0:0:0:0", d.SCSIAddr)
				s.Equal("disk", d.Type)
				s.Equal("ATA", d.Transport)
				s.Equal("ST500DM002-1BD14", d.Name)
				s.Equal("KC48", d.Revision)
				s.Equal("/dev/sda", d.Device)

				d = info.Devices["5:0:0:0"]
				s.Equal("LSI", d.Transport)
				s.Equal("MR9240-8i", d.Name)
				s.Equal("2.13", d.Revision)

				// Vendor + model with a space between them.
				d = info.Devices["6:0:0:0"]
				s.Equal("cd/dvd", d.Type)
				s.Equal("NECVMWar", d.Transport)
				s.Equal("VMware IDE CDR10", d.Name)
				s.Equal("1.00", d.Revision)
				s.Equal("/dev/sr0", d.Device)
			},
		},
		{
			name:    "linux: missing lsscsi binary yields empty info, no error",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsscsiExec(t, nil, errors.New("not found"))
			},
			validate: func(info *scsi.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "linux: nil Exec yields empty info",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			validate: func(info *scsi.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "linux: empty output yields empty info",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsscsiExec(t, []byte{}, nil)
			},
			validate: func(info *scsi.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "linux: minimum 5-field row (no name) still parses",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsscsiExec(t, []byte("[0:0:0:0] disk ATA KC48 /dev/sda\n"), nil)
			},
			validate: func(info *scsi.Info) {
				s.Require().Len(info.Devices, 1)
				d := info.Devices["0:0:0:0"]
				s.Equal("disk", d.Type)
				s.Equal("ATA", d.Transport)
				s.Empty(d.Name) // no tokens between transport and revision
				s.Equal("KC48", d.Revision)
				s.Equal("/dev/sda", d.Device)
			},
		},
		{
			name:    "linux: row with empty brackets is skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsscsiExec(t, []byte("[] disk ATA KC48 /dev/sda\n"), nil)
			},
			validate: func(info *scsi.Info) {
				s.Empty(info.Devices)
			},
		},
		{
			name:    "darwin: returns empty",
			variant: "darwin",
			validate: func(info *scsi.Info) {
				s.Require().NotNil(info)
				s.Empty(info.Devices)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c scsi.Collector
			switch tt.variant {
			case "linux":
				c = &scsi.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &scsi.Darwin{}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*scsi.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
