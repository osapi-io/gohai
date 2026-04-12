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

type HostnameLinuxPublicTestSuite struct {
	suite.Suite
}

func TestHostnameLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HostnameLinuxPublicTestSuite))
}

func (s *HostnameLinuxPublicTestSuite) TestCollect() {
	okShort := func(context.Context) (string, error) { return "web01", nil }

	tests := []struct {
		name         string
		shortFn      func(context.Context) (string, error)
		osHostnameFn func() (string, error)
		lookupHost   func(string) ([]string, error)
		lookupAddr   func(string) ([]string, error)
		wantErr      bool
		want         hostname.Info
	}{
		{
			name:         "canonical case: short name + reverse DNS",
			shortFn:      okShort,
			osHostnameFn: func() (string, error) { return "web01", nil },
			lookupHost:   func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr:   func(string) ([]string, error) { return []string{"web01.example.com."}, nil },
			want: hostname.Info{
				Hostname:    "web01",
				MachineName: "web01",
				FQDN:        "web01.example.com",
				Domain:      "example.com",
			},
		},
		{
			name:         "no DNS resolution falls back to short name",
			shortFn:      okShort,
			osHostnameFn: func() (string, error) { return "laptop", nil },
			lookupHost:   func(string) ([]string, error) { return nil, errors.New("no such host") },
			lookupAddr:   func(string) ([]string, error) { return nil, errors.New("unused") },
			want:         hostname.Info{Hostname: "web01", MachineName: "laptop", FQDN: "web01"},
		},
		{
			name:         "reverse lookup fails falls back to short name",
			shortFn:      okShort,
			osHostnameFn: func() (string, error) { return "web01", nil },
			lookupHost:   func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr:   func(string) ([]string, error) { return nil, errors.New("no PTR") },
			want:         hostname.Info{Hostname: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:         "reverse lookup empty falls back to short name",
			shortFn:      okShort,
			osHostnameFn: func() (string, error) { return "web01", nil },
			lookupHost:   func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr:   func(string) ([]string, error) { return nil, nil },
			want:         hostname.Info{Hostname: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:         "empty short name skips FQDN resolution",
			shortFn:      func(context.Context) (string, error) { return "", nil },
			osHostnameFn: func() (string, error) { return "", errors.New("no hostname") },
			lookupHost:   func(string) ([]string, error) { return nil, errors.New("unused") },
			lookupAddr:   func(string) ([]string, error) { return nil, errors.New("unused") },
			want:         hostname.Info{},
		},
		{
			name:         "short hostname error propagated",
			shortFn:      func(context.Context) (string, error) { return "", errors.New("boom") },
			osHostnameFn: func() (string, error) { return "", nil },
			lookupHost:   func(string) ([]string, error) { return nil, nil },
			lookupAddr:   func(string) ([]string, error) { return nil, nil },
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &hostname.Linux{
				ShortHostnameFn: tt.shortFn,
				OSHostnameFn:    tt.osHostnameFn,
				LookupHostFn:    tt.lookupHost,
				LookupAddrFn:    tt.lookupAddr,
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

func (s *HostnameLinuxPublicTestSuite) TestReadShortHostname() {
	tests := []struct {
		name    string
		hostFn  func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    string
	}{
		{
			name:   "success returns Hostname field",
			hostFn: func(context.Context) (*host.InfoStat, error) { return &host.InfoStat{Hostname: "web01"}, nil },
			want:   "web01",
		},
		{
			name:   "nil info returns empty string",
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
			restore := hostname.SetHostInfoFn(tt.hostFn)
			defer restore()
			got, err := hostname.ReadShortHostname(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
