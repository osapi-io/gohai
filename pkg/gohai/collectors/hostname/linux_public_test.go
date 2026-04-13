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
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

type HostnameLinuxPublicTestSuite struct {
	suite.Suite
}

func TestHostnameLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HostnameLinuxPublicTestSuite))
}

// hostnameExec returns a MockExecutor that canned-answers
// `hostname -s` and bare `hostname` separately. Shared by linux and
// darwin tests because both OSes run the same commands.
func hostnameExec(
	t *testing.T,
	shortOut []byte, shortErr error,
	bareOut []byte, bareErr error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "hostname", "-s").
		Return(shortOut, shortErr).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "hostname").
		Return(bareOut, bareErr).
		AnyTimes()
	return m
}

func (s *HostnameLinuxPublicTestSuite) TestCollect() {
	defer hostname.SetResolverBackoff(0)()

	okHostInfo := func(context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Hostname: "gopsutil-short"}, nil
	}

	tests := []struct {
		name       string
		hostInfo   func(context.Context) (*host.InfoStat, error)
		osHostname func() (string, error)
		lookupHost func(string) ([]string, error)
		lookupAddr func(string) ([]string, error)
		execShort  []byte
		execBare   []byte
		want       hostname.Info
	}{
		{
			name:       "canonical success with reverse DNS",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr: func(string) ([]string, error) { return []string{"web01.example.com."}, nil },
			execShort:  []byte("web01\n"),
			execBare:   []byte("web01\n"),
			want: hostname.Info{
				Name:        "web01",
				MachineName: "web01",
				FQDN:        "web01.example.com",
				Domain:      "example.com",
			},
		},
		{
			name:       "empty forward lookup treated as miss; all retries exhausted",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: func(string) ([]string, error) { return nil, nil },
			lookupAddr: func(string) ([]string, error) { return nil, nil },
			execShort:  []byte("web01\n"),
			execBare:   []byte("web01\n"),
			want:       hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:       "empty reverse lookup treated as miss",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr: func(string) ([]string, error) { return nil, nil },
			execShort:  []byte("web01\n"),
			execBare:   []byte("web01\n"),
			want:       hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:       "FQDN without domain component",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: func(string) ([]string, error) { return []string{"10.0.0.5"}, nil },
			lookupAddr: func(string) ([]string, error) { return []string{"web01."}, nil },
			execShort:  []byte("web01\n"),
			execBare:   []byte("web01\n"),
			want:       hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer hostname.SetHostInfoFn(tt.hostInfo)()
			defer hostname.SetOSHostnameFn(tt.osHostname)()
			defer hostname.SetLookupHostFn(tt.lookupHost)()
			defer hostname.SetLookupAddrFn(tt.lookupAddr)()

			c := &hostname.Linux{
				Exec: hostnameExec(s.T(), tt.execShort, nil, tt.execBare, nil),
			}
			got, err := c.Collect(context.Background())
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
