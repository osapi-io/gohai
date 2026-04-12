//go:build linux

// Copyright (c) 2024 John Dewey

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

func (s *HostnameLinuxPublicTestSuite) TestBuildInfo() {
	tests := []struct {
		name       string
		short      string
		resolveFn  func(string) string
		wantFQDN   string
		wantDomain string
	}{
		{
			name:       "fqdn returns full name with domain",
			short:      "web01",
			resolveFn:  func(_ string) string { return "web01.example.com" },
			wantFQDN:   "web01.example.com",
			wantDomain: "example.com",
		},
		{
			name:       "short-only hostname when resolver returns input",
			short:      "standalone",
			resolveFn:  func(s string) string { return s },
			wantFQDN:   "standalone",
			wantDomain: "",
		},
		{
			name:       "fqdn ending in dot is not treated as domain",
			short:      "foo",
			resolveFn:  func(_ string) string { return "foo." },
			wantFQDN:   "foo.",
			wantDomain: "",
		},
		{
			name:       "multi-label domain",
			short:      "db",
			resolveFn:  func(_ string) string { return "db.prod.internal.corp" },
			wantFQDN:   "db.prod.internal.corp",
			wantDomain: "prod.internal.corp",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := hostname.BuildInfo(tt.short, tt.resolveFn)
			s.Equal(tt.short, got.Hostname)
			s.Equal(tt.wantFQDN, got.FQDN)
			s.Equal(tt.wantDomain, got.Domain)
		})
	}
}

func (s *HostnameLinuxPublicTestSuite) TestDefaultFqdnLookup() {
	tests := []struct {
		name   string
		lookup func(string) (string, error)
		input  string
		want   string
	}{
		{
			"cname trimmed of trailing dot",
			func(_ string) (string, error) { return "web01.example.com.", nil },
			"web01",
			"web01.example.com",
		},
		{
			"cname without trailing dot",
			func(_ string) (string, error) { return "web01.example.com", nil },
			"web01",
			"web01.example.com",
		},
		{
			"empty cname falls back to input",
			func(_ string) (string, error) { return "", nil },
			"solo",
			"solo",
		},
		{
			"lookup error falls back to input",
			func(_ string) (string, error) { return "", errors.New("dns") },
			"bad",
			"bad",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := hostname.SetLookupCNAMEFn(tt.lookup)
			defer hostname.SetLookupCNAMEFn(restore)
			s.Equal(tt.want, hostname.DefaultFqdnLookup(tt.input))
		})
	}
}

func (s *HostnameLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		stub    func(context.Context) (*host.InfoStat, error)
		resolve func(string) string
		wantErr bool
		want    *hostname.Info
	}{
		{
			name:    "success with fqdn",
			stub:    func(_ context.Context) (*host.InfoStat, error) { return &host.InfoStat{Hostname: "web01"}, nil },
			resolve: func(_ string) string { return "web01.example.com" },
			want: &hostname.Info{
				Hostname: "web01",
				FQDN:     "web01.example.com",
				Domain:   "example.com",
			},
		},
		{
			name:    "success without domain",
			stub:    func(_ context.Context) (*host.InfoStat, error) { return &host.InfoStat{Hostname: "solo"}, nil },
			resolve: func(s string) string { return s },
			want:    &hostname.Info{Hostname: "solo", FQDN: "solo"},
		},
		{
			name:    "host.Info error",
			stub:    func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			resolve: func(s string) string { return s },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreHost := hostname.SetHostInfoFn(tt.stub)
			restoreFqdn := hostname.SetFqdnLookupFn(tt.resolve)
			defer hostname.SetHostInfoFn(restoreHost)
			defer hostname.SetFqdnLookupFn(restoreFqdn)

			got, err := hostname.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*hostname.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info)
		})
	}
}
