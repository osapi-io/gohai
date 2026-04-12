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

package machineid_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
)

type MachineIDLinuxPublicTestSuite struct {
	suite.Suite
}

func TestMachineIDLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MachineIDLinuxPublicTestSuite))
}

func (s *MachineIDLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		hostIDFn   func(context.Context) (string, error)
		readFileFn func(string) ([]byte, error)
		wantErr    bool
		wantID     string
	}{
		{
			name:       "gopsutil returns /etc/machine-id → use it",
			hostIDFn:   func(context.Context) (string, error) { return "gopsutil-id", nil },
			readFileFn: func(string) ([]byte, error) { return nil, errors.New("should not be called") },
			wantID:     "gopsutil-id",
		},
		{
			name:       "gopsutil empty, dbus fallback wins",
			hostIDFn:   func(context.Context) (string, error) { return "", nil },
			readFileFn: func(string) ([]byte, error) { return []byte("dbus-id\n"), nil },
			wantID:     "dbus-id",
		},
		{
			name:       "gopsutil empty, dbus missing → empty ID (no error)",
			hostIDFn:   func(context.Context) (string, error) { return "", nil },
			readFileFn: func(string) ([]byte, error) { return nil, errors.New("not found") },
			wantID:     "",
		},
		{
			name:       "gopsutil empty, dbus whitespace-only → empty ID",
			hostIDFn:   func(context.Context) (string, error) { return "", nil },
			readFileFn: func(string) ([]byte, error) { return []byte("   \n"), nil },
			wantID:     "",
		},
		{
			name:       "gopsutil error wrapped and returned",
			hostIDFn:   func(context.Context) (string, error) { return "", errors.New("boom") },
			readFileFn: func(string) ([]byte, error) { return nil, errors.New("ignored") },
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &machineid.Linux{HostIDFn: tt.hostIDFn, ReadFileFn: tt.readFileFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*machineid.Info)
			s.Require().True(ok)
			s.Equal(tt.wantID, info.ID)
		})
	}
}

func (s *MachineIDLinuxPublicTestSuite) TestReadHostID() {
	tests := []struct {
		name    string
		hostFn  func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    string
	}{
		{
			name:   "success returns HostID",
			hostFn: func(context.Context) (*host.InfoStat, error) { return &host.InfoStat{HostID: "abc123"}, nil },
			want:   "abc123",
		},
		{
			name:   "nil info returns empty",
			hostFn: func(context.Context) (*host.InfoStat, error) { return nil, nil },
			want:   "",
		},
		{
			name:    "gopsutil error wrapped and returned",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := machineid.SetHostInfoFn(tt.hostFn)
			defer restore()
			got, err := machineid.ReadHostID(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
