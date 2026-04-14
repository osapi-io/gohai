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

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

var (
	_ collector.Collector = (*hostname.Linux)(nil)
	_ collector.Collector = (*hostname.Darwin)(nil)
)

type HostnamePublicTestSuite struct {
	suite.Suite
}

func TestHostnamePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnamePublicTestSuite))
}

// hostnameExec returns a MockExecutor that canned-answers
// `hostname -s` and bare `hostname` separately.
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

func (s *HostnamePublicTestSuite) TestNew() {
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
			c := hostname.New()
			s.Equal("hostname", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*hostname.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *HostnamePublicTestSuite) TestCollect() {
	defer hostname.SetResolverBackoff(0)()

	okHostInfo := func(context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Hostname: "gopsutil-short"}, nil
	}
	errHostInfo := func(context.Context) (*host.InfoStat, error) {
		return nil, errors.New("boom")
	}
	okLookupHost := func(string) ([]string, error) { return []string{"10.0.0.5"}, nil }
	okLookupAddr := func(string) ([]string, error) { return []string{"web01.example.com."}, nil }
	darwinLookupHost := func(string) ([]string, error) { return []string{"192.168.1.42"}, nil }
	darwinLookupAddr := func(string) ([]string, error) { return []string{"johns-mbp.local."}, nil }
	emptyLookupHost := func(string) ([]string, error) { return nil, nil }
	emptyLookupAddr := func(string) ([]string, error) { return nil, nil }
	failLookupHost := func(string) ([]string, error) { return nil, errors.New("no host") }
	failLookupAddr := func(string) ([]string, error) { return nil, errors.New("unused") }

	tests := []struct {
		name       string
		variant    string
		hostInfo   func(context.Context) (*host.InfoStat, error)
		osHostname func() (string, error)
		lookupHost func(string) ([]string, error)
		lookupAddr func(string) ([]string, error)
		exec       func(*testing.T) executor.Executor
		want       hostname.Info
		wantErr    bool
	}{
		{
			name:       "linux: canonical success with reverse DNS",
			variant:    "linux",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: okLookupHost,
			lookupAddr: okLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			want: hostname.Info{
				Name:        "web01",
				MachineName: "web01",
				FQDN:        "web01.example.com",
				Domain:      "example.com",
			},
		},
		{
			name:       "linux: empty forward lookup exhausts retries",
			variant:    "linux",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			want: hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:       "linux: empty reverse lookup treated as miss",
			variant:    "linux",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: okLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			want: hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:       "linux: FQDN without domain component",
			variant:    "linux",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: okLookupHost,
			lookupAddr: func(string) ([]string, error) { return []string{"web01."}, nil },
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			want: hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"},
		},
		{
			name:       "darwin: exec succeeds, hostname -s + hostname beat gopsutil",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: darwinLookupHost,
			lookupAddr: darwinLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte("johns-mbp\n"),
					nil,
					[]byte("Johns MacBook Pro\n"),
					nil,
				)
			},
			want: hostname.Info{
				Name:        "johns-mbp",
				MachineName: "Johns MacBook Pro",
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "darwin: hostname -s fails, short falls back to gopsutil",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					nil,
					errors.New("no hostname -s"),
					[]byte("Friendly Name\n"),
					nil,
				)
			},
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "Friendly Name",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "darwin: both exec fail, fall back to gopsutil + os.Hostname",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New("e1"), nil, errors.New("e2"))
			},
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "darwin: both exec fail and os.Hostname errors, machine_name mirrors short",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "", errors.New("nope") },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New("e1"), nil, errors.New("e2"))
			},
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "gopsutil-short",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "darwin: empty exec output treated as fallback",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("\n"), nil, []byte(""), nil)
			},
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "darwin: nil Exec path",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       func(*testing.T) executor.Executor { return nil },
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "darwin: short hostname error propagated",
			variant:    "darwin",
			hostInfo:   errHostInfo,
			osHostname: func() (string, error) { return "", nil },
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New("e1"), nil, errors.New("e2"))
			},
			wantErr: true,
		},
		{
			name:       "darwin: transient DNS failure recovers on retry",
			variant:    "darwin",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: func() func(string) ([]string, error) {
				n := 0
				return func(string) ([]string, error) {
					n++
					if n < 2 {
						return nil, errors.New("transient")
					}
					return []string{"192.168.1.42"}, nil
				}
			}(),
			lookupAddr: darwinLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil)
			},
			want: hostname.Info{
				Name:        "johns-mbp",
				MachineName: "Johns",
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "darwin: empty short name skips FQDN",
			variant:    "darwin",
			hostInfo:   func(context.Context) (*host.InfoStat, error) { return nil, nil },
			osHostname: func() (string, error) { return "", errors.New("no hostname") },
			lookupHost: func(string) ([]string, error) { return nil, errors.New("unused") },
			lookupAddr: func(string) ([]string, error) { return nil, errors.New("unused") },
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New("e1"), nil, errors.New("e2"))
			},
			want: hostname.Info{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer hostname.SetHostInfoFn(tt.hostInfo)()
			defer hostname.SetOSHostnameFn(tt.osHostname)()
			defer hostname.SetLookupHostFn(tt.lookupHost)()
			defer hostname.SetLookupAddrFn(tt.lookupAddr)()

			var c hostname.Collector
			switch tt.variant {
			case "linux":
				c = &hostname.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &hostname.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
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
