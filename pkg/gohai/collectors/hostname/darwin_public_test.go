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

package hostname_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

type HostnameDarwinPublicTestSuite struct {
	suite.Suite
}

func TestHostnameDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HostnameDarwinPublicTestSuite))
}

func (s *HostnameDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		hostFn       func(context.Context) (*host.InfoStat, error)
		osHostnameFn func() (string, error)
		lookupHost   func(string) ([]string, error)
		lookupAddr   func(string) ([]string, error)
		want         hostname.Info
		wantErr      bool
	}{
		{
			name:         "canonical macOS host with DNS",
			hostFn:       func(_ context.Context) (*host.InfoStat, error) { return &host.InfoStat{Hostname: "johns-mbp"}, nil },
			osHostnameFn: func() (string, error) { return "johns-mbp.local", nil },
			lookupHost:   func(string) ([]string, error) { return []string{"192.168.1.42"}, nil },
			lookupAddr:   func(string) ([]string, error) { return []string{"johns-mbp.local."}, nil },
			want: hostname.Info{
				Hostname: "johns-mbp", MachineName: "johns-mbp.local",
				FQDN: "johns-mbp.local", Domain: "local",
			},
		},
		{
			name:         "no DNS falls back to short name",
			hostFn:       func(_ context.Context) (*host.InfoStat, error) { return &host.InfoStat{Hostname: "laptop"}, nil },
			osHostnameFn: func() (string, error) { return "laptop", nil },
			lookupHost:   func(string) ([]string, error) { return nil, errors.New("no host") },
			lookupAddr:   func(string) ([]string, error) { return nil, errors.New("unused") },
			want:         hostname.Info{Hostname: "laptop", MachineName: "laptop", FQDN: "laptop"},
		},
		{
			name:         "host.Info error propagated",
			hostFn:       func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			osHostnameFn: func() (string, error) { return "", nil },
			lookupHost:   func(string) ([]string, error) { return nil, nil },
			lookupAddr:   func(string) ([]string, error) { return nil, nil },
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &hostname.Darwin{
				HostInfoFn:   tt.hostFn,
				OSHostnameFn: tt.osHostnameFn,
				LookupHostFn: tt.lookupHost,
				LookupAddrFn: tt.lookupAddr,
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*hostname.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
