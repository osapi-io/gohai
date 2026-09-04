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
	"net"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
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
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", "debian", osLinux},
		{"rhel dispatches to Linux", "rhel", osLinux},
		{"arch dispatches to Linux", "arch", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
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
			case osDarwin:
				_, ok := c.(*hostname.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *HostnamePublicTestSuite) TestCollect() {
	restore0 := hostname.SetResolverBackoff(0)
	defer restore0()

	okHostInfo := func(context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Hostname: hostGopsutil}, nil
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
	failLookupAddr := func(string) ([]string, error) { return nil, errors.New(valueUnused) }

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
			variant:    osLinux,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: okLookupHost,
			lookupAddr: okLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostWeb01WithNewline),
					nil,
					[]byte(hostWeb01WithNewline),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostWeb01,
				MachineName: hostWeb01,
				FQDN:        "web01.example.com",
				Domain:      "example.com",
			},
		},
		{
			name:       "linux: empty forward lookup short-circuits",
			variant:    osLinux,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostWeb01WithNewline),
					nil,
					[]byte(hostWeb01WithNewline),
					nil,
				)
			},
			want: hostname.Info{Name: hostWeb01, MachineName: hostWeb01, FQDN: hostWeb01},
		},
		{
			name:       "linux: empty reverse lookup treated as miss",
			variant:    osLinux,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: okLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostWeb01WithNewline),
					nil,
					[]byte(hostWeb01WithNewline),
					nil,
				)
			},
			want: hostname.Info{Name: hostWeb01, MachineName: hostWeb01, FQDN: hostWeb01},
		},
		{
			name:       "linux: FQDN without domain component",
			variant:    osLinux,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: okLookupHost,
			lookupAddr: func(string) ([]string, error) { return []string{"web01."}, nil },
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostWeb01WithNewline),
					nil,
					[]byte(hostWeb01WithNewline),
					nil,
				)
			},
			want: hostname.Info{Name: hostWeb01, MachineName: hostWeb01, FQDN: hostWeb01},
		},
		{
			name:       "darwin: exec succeeds, hostname -s + hostname beat gopsutil",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: darwinLookupHost,
			lookupAddr: darwinLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostMBPWithNewline),
					nil,
					[]byte("Johns MacBook Pro\n"),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostMBP,
				MachineName: "Johns MacBook Pro",
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "darwin: hostname -s fails, short falls back to gopsutil",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
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
				Name:        hostGopsutil,
				MachineName: "Friendly Name",
				FQDN:        hostGopsutil,
			},
		},
		{
			name:       "darwin: both exec fail, fall back to gopsutil + os.Hostname",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return hostFromOS, nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New(entryOne), nil, errors.New(entryTwo))
			},
			want: hostname.Info{
				Name:        hostGopsutil,
				MachineName: hostFromOS,
				FQDN:        hostGopsutil,
			},
		},
		{
			name:       "darwin: both exec fail and os.Hostname errors, machine_name mirrors short",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "", errors.New("nope") },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New(entryOne), nil, errors.New(entryTwo))
			},
			want: hostname.Info{
				Name:        hostGopsutil,
				MachineName: hostGopsutil,
				FQDN:        hostGopsutil,
			},
		},
		{
			name:       "darwin: empty exec output treated as fallback",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return hostFromOS, nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, []byte("\n"), nil, []byte(""), nil)
			},
			want: hostname.Info{
				Name:        hostGopsutil,
				MachineName: hostFromOS,
				FQDN:        hostGopsutil,
			},
		},
		{
			name:       "darwin: nil Exec path",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return hostFromOS, nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       func(*testing.T) executor.Executor { return nil },
			want: hostname.Info{
				Name:        hostGopsutil,
				MachineName: hostFromOS,
				FQDN:        hostGopsutil,
			},
		},
		{
			name:       "darwin: short hostname error propagated",
			variant:    osDarwin,
			hostInfo:   errHostInfo,
			osHostname: func() (string, error) { return "", nil },
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New(entryOne), nil, errors.New(entryTwo))
			},
			wantErr: true,
		},
		{
			name:       "darwin: transient DNS failure recovers on retry",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
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
				return hostnameExec(
					t,
					[]byte(hostMBPWithNewline),
					nil,
					[]byte(nameJohnsWithNewline),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostMBP,
				MachineName: nameJohns,
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "darwin: transient LookupAddr failure recovers on retry",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: darwinLookupHost,
			lookupAddr: func() func(string) ([]string, error) {
				n := 0
				return func(string) ([]string, error) {
					n++
					if n < 2 {
						return nil, errors.New("transient PTR")
					}
					return []string{"johns-mbp.local."}, nil
				}
			}(),
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostMBPWithNewline),
					nil,
					[]byte(nameJohnsWithNewline),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostMBP,
				MachineName: nameJohns,
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "darwin: IsNotFound on LookupHost short-circuits (no retry sleep)",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: func() func(string) ([]string, error) {
				calls := 0
				return func(string) ([]string, error) {
					calls++
					s.Equal(1, calls)
					return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
				}
			}(),
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostMBPWithNewline),
					nil,
					[]byte(nameJohnsWithNewline),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostMBP,
				MachineName: nameJohns,
				FQDN:        hostMBP,
			},
		},
		{
			name:       "darwin: IsNotFound on LookupAddr short-circuits",
			variant:    osDarwin,
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return valueUnused, nil },
			lookupHost: darwinLookupHost,
			lookupAddr: func() func(string) ([]string, error) {
				calls := 0
				return func(string) ([]string, error) {
					calls++
					s.Equal(1, calls)
					return nil, &net.DNSError{Err: "no PTR", IsNotFound: true}
				}
			}(),
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(
					t,
					[]byte(hostMBPWithNewline),
					nil,
					[]byte(nameJohnsWithNewline),
					nil,
				)
			},
			want: hostname.Info{
				Name:        hostMBP,
				MachineName: nameJohns,
				FQDN:        hostMBP,
			},
		},
		{
			name:       "darwin: empty short name skips FQDN",
			variant:    osDarwin,
			hostInfo:   func(context.Context) (*host.InfoStat, error) { return nil, nil },
			osHostname: func() (string, error) { return "", errors.New("no hostname") },
			lookupHost: func(string) ([]string, error) { return nil, errors.New(valueUnused) },
			lookupAddr: func(string) ([]string, error) { return nil, errors.New(valueUnused) },
			exec: func(t *testing.T) executor.Executor {
				return hostnameExec(t, nil, errors.New(entryOne), nil, errors.New(entryTwo))
			},
			want: hostname.Info{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore1 := hostname.SetHostInfoFn(tt.hostInfo)
			defer restore1()
			restore2 := hostname.SetOSHostnameFn(tt.osHostname)
			defer restore2()
			restore3 := hostname.SetLookupHostFn(tt.lookupHost)
			defer restore3()
			restore4 := hostname.SetLookupAddrFn(tt.lookupAddr)
			defer restore4()

			var c hostname.Collector
			switch tt.variant {
			case osLinux:
				c = &hostname.Linux{Exec: tt.exec(s.T())}
			case osDarwin:
				c = &hostname.Darwin{Exec: tt.exec(s.T())}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
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
