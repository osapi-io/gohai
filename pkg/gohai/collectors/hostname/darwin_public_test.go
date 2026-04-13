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

	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

type HostnameDarwinPublicTestSuite struct {
	suite.Suite
}

func TestHostnameDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HostnameDarwinPublicTestSuite))
}

func (s *HostnameDarwinPublicTestSuite) TestCollect() {
	defer hostname.SetResolverBackoff(0)()

	okHostInfo := func(context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Hostname: "gopsutil-short"}, nil
	}
	errHostInfo := func(context.Context) (*host.InfoStat, error) {
		return nil, errors.New("boom")
	}
	okLookupHost := func(string) ([]string, error) { return []string{"192.168.1.42"}, nil }
	okLookupAddr := func(string) ([]string, error) { return []string{"johns-mbp.local."}, nil }
	failLookupHost := func(string) ([]string, error) { return nil, errors.New("no host") }
	failLookupAddr := func(string) ([]string, error) { return nil, errors.New("unused") }

	tests := []struct {
		name       string
		hostInfo   func(context.Context) (*host.InfoStat, error)
		osHostname func() (string, error)
		lookupHost func(string) ([]string, error)
		lookupAddr func(string) ([]string, error)
		exec       executor.Executor
		want       hostname.Info
		wantErr    bool
	}{
		{
			name:       "exec succeeds: hostname -s + hostname win over gopsutil",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: okLookupHost,
			lookupAddr: okLookupAddr,
			exec: hostnameExec(
				s.T(),
				[]byte("johns-mbp\n"),
				nil,
				[]byte("Johns MacBook Pro\n"),
				nil,
			),
			want: hostname.Info{
				Name:        "johns-mbp",
				MachineName: "Johns MacBook Pro",
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "hostname -s fails: short falls back to gopsutil; machine_name from bare hostname",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "unused", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: hostnameExec(
				s.T(),
				nil,
				errors.New("no hostname -s"),
				[]byte("Friendly Name\n"),
				nil,
			),
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "Friendly Name",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "both exec fail: fall back to gopsutil + os.Hostname",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       hostnameExec(s.T(), nil, errors.New("e1"), nil, errors.New("e2")),
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "both exec fail and os.Hostname errors: machine_name mirrors short",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "", errors.New("nope") },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       hostnameExec(s.T(), nil, errors.New("e1"), nil, errors.New("e2")),
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "gopsutil-short",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "empty exec output treated as fallback",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       hostnameExec(s.T(), []byte("\n"), nil, []byte(""), nil),
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "nil Exec path",
			hostInfo:   okHostInfo,
			osHostname: func() (string, error) { return "os-hostname", nil },
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       nil,
			want: hostname.Info{
				Name:        "gopsutil-short",
				MachineName: "os-hostname",
				FQDN:        "gopsutil-short",
			},
		},
		{
			name:       "short hostname error propagated",
			hostInfo:   errHostInfo,
			osHostname: func() (string, error) { return "", nil },
			lookupHost: func(string) ([]string, error) { return nil, nil },
			lookupAddr: func(string) ([]string, error) { return nil, nil },
			exec:       hostnameExec(s.T(), nil, errors.New("e1"), nil, errors.New("e2")),
			wantErr:    true,
		},
		{
			name:       "transient DNS failure recovers on retry",
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
			lookupAddr: okLookupAddr,
			exec:       hostnameExec(s.T(), []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil),
			want: hostname.Info{
				Name:        "johns-mbp",
				MachineName: "Johns",
				FQDN:        "johns-mbp.local",
				Domain:      "local",
			},
		},
		{
			name:       "empty short name skips FQDN",
			hostInfo:   func(context.Context) (*host.InfoStat, error) { return nil, nil },
			osHostname: func() (string, error) { return "", errors.New("no hostname") },
			lookupHost: func(string) ([]string, error) { return nil, errors.New("unused") },
			lookupAddr: func(string) ([]string, error) { return nil, errors.New("unused") },
			exec:       hostnameExec(s.T(), nil, errors.New("e1"), nil, errors.New("e2")),
			want:       hostname.Info{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer hostname.SetHostInfoFn(tt.hostInfo)()
			defer hostname.SetOSHostnameFn(tt.osHostname)()
			defer hostname.SetLookupHostFn(tt.lookupHost)()
			defer hostname.SetLookupAddrFn(tt.lookupAddr)()

			c := &hostname.Darwin{Exec: tt.exec}
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
