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
	"fmt"
	"net"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*hostname.Linux)(nil)
	_ collector.Collector = (*hostname.Darwin)(nil)
)

type HostnamePublicTestSuite struct {
	suite.Suite
}

func TestHostnamePublicTestSuite(t *testing.T) {
	suite.Run(t, new(HostnamePublicTestSuite))
}

// hostnameExec returns a MockRunner that canned-answers
// `hostname -s` and bare `hostname` separately.
// errNotOnPath is the error of a hostname that is not on the PATH.
var errNotOnPath = fmt.Errorf("hostname: %w", exec.ErrNotFound)

func hostnameExec(
	t *testing.T,
	shortOut []byte, shortErr error,
	bareOut []byte, bareErr error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
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
		name         string
		detect       string
		validateFunc func(hostname.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c hostname.Collector) {
				_, ok := c.(*hostname.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c hostname.Collector) {
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c hostname.Collector) {
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c hostname.Collector) {
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c hostname.Collector) {
				_, ok := c.(*hostname.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := hostname.New()
			s.Equal("hostname", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *HostnamePublicTestSuite) TestCollect() {
	// cancelledCalls counts the lookups of the cancelled row. A retry that
	// sleeps through the cancel calls LookupHost again.
	cancelledCalls := 0

	kernelName := map[string]string{"/proc/sys/kernel/hostname": "kernel-short.example.com\n"}
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
		files      map[string]string
		lookupHost func(string) ([]string, error)
		lookupAddr func(string) ([]string, error)
		exec       func(*testing.T) source.Runner
		// cancelled runs Collect with a ctx that has already ended.
		cancelled    bool
		validateFunc func(any, error)
	}{
		{
			name:       "linux: canonical success with reverse DNS",
			variant:    "linux",
			files:      kernelName,
			lookupHost: okLookupHost,
			lookupAddr: okLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "web01",
					MachineName: "web01",
					FQDN:        "web01.example.com",
					Domain:      "example.com",
				}, *info)
			},
		},
		{
			name:       "linux: empty forward lookup short-circuits",
			variant:    "linux",
			files:      kernelName,
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"}, *info)
			},
		},
		{
			name:       "linux: empty reverse lookup treated as miss",
			variant:    "linux",
			files:      kernelName,
			lookupHost: okLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"}, *info)
			},
		},
		{
			name:       "linux: FQDN without domain component",
			variant:    "linux",
			files:      kernelName,
			lookupHost: okLookupHost,
			lookupAddr: func(string) ([]string, error) { return []string{"web01."}, nil },
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("web01\n"), nil, []byte("web01\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{Name: "web01", MachineName: "web01", FQDN: "web01"}, *info)
			},
		},
		{
			name:       "darwin: exec succeeds, hostname -s + hostname beat the kernel name",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: darwinLookupHost,
			lookupAddr: darwinLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(
					t,
					[]byte("johns-mbp\n"),
					nil,
					[]byte("Johns MacBook Pro\n"),
					nil,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "johns-mbp",
					MachineName: "Johns MacBook Pro",
					FQDN:        "johns-mbp.local",
					Domain:      "local",
				}, *info)
			},
		},
		{
			name:       "darwin: no hostname -s, short falls back to the kernel name",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(
					t,
					nil,
					errNotOnPath,
					[]byte("Friendly Name\n"),
					nil,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "kernel-short",
					MachineName: "Friendly Name",
					FQDN:        "kernel-short",
				}, *info)
			},
		},
		{
			name:       "darwin: no hostname on the PATH, fall back to the kernel's hostname",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, errNotOnPath, nil, errNotOnPath)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "kernel-short",
					MachineName: "kernel-short",
					FQDN:        "kernel-short",
				}, *info)
			},
		},
		{
			name:       "linux: no hostname on the PATH, fall back to the kernel's hostname",
			variant:    "linux",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, errNotOnPath, nil, errNotOnPath)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "kernel-short",
					MachineName: "kernel-short",
					FQDN:        "kernel-short",
				}, *info)
			},
		},
		{
			name:       "linux: a failed hostname -s is an error",
			variant:    "linux",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, errors.New("exit status 1"), []byte("box\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.EqualError(err, "hostname -s: exit status 1")
				s.Exactly(nil, got)
			},
		},
		{
			name:       "linux: a failed hostname is an error",
			variant:    "linux",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("box\n"), nil, nil, errors.New("exit status 1"))
			},
			validateFunc: func(got any, err error) {
				s.EqualError(err, "hostname : exit status 1")
				s.Exactly(nil, got)
			},
		},
		{
			name:       "darwin: a cancelled context is an error",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, context.Canceled, nil, context.Canceled)
			},
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:       "darwin: empty exec output treated as fallback",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("\n"), nil, []byte(""), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "kernel-short",
					MachineName: "kernel-short",
					FQDN:        "kernel-short",
				}, *info)
			},
		},
		{
			name:       "darwin: nil Exec path",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: failLookupHost,
			lookupAddr: failLookupAddr,
			exec:       func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "kernel-short",
					MachineName: "kernel-short",
					FQDN:        "kernel-short",
				}, *info)
			},
		},
		{
			name:       "darwin: short hostname error propagated",
			variant:    "darwin",
			files:      nil,
			lookupHost: emptyLookupHost,
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, errNotOnPath, nil, errNotOnPath)
			},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: transient DNS failure recovers on retry",
			variant: "darwin",
			files:   kernelName,
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
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "johns-mbp",
					MachineName: "Johns",
					FQDN:        "johns-mbp.local",
					Domain:      "local",
				}, *info)
			},
		},
		{
			name:       "darwin: transient LookupAddr failure recovers on retry",
			variant:    "darwin",
			files:      kernelName,
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
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "johns-mbp",
					MachineName: "Johns",
					FQDN:        "johns-mbp.local",
					Domain:      "local",
				}, *info)
			},
		},
		{
			name:    "darwin: IsNotFound on LookupHost short-circuits (no retry sleep)",
			variant: "darwin",
			files:   kernelName,
			lookupHost: func() func(string) ([]string, error) {
				calls := 0
				return func(string) ([]string, error) {
					calls++
					s.Equal(1, calls)
					return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
				}
			}(),
			lookupAddr: failLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "johns-mbp",
					MachineName: "Johns",
					FQDN:        "johns-mbp",
				}, *info)
			},
		},
		{
			name:       "darwin: IsNotFound on LookupAddr short-circuits",
			variant:    "darwin",
			files:      kernelName,
			lookupHost: darwinLookupHost,
			lookupAddr: func() func(string) ([]string, error) {
				calls := 0
				return func(string) ([]string, error) {
					calls++
					s.Equal(1, calls)
					return nil, &net.DNSError{Err: "no PTR", IsNotFound: true}
				}
			}(),
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, []byte("johns-mbp\n"), nil, []byte("Johns\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{
					Name:        "johns-mbp",
					MachineName: "Johns",
					FQDN:        "johns-mbp",
				}, *info)
			},
		},
		{
			name:       "darwin: empty short name skips FQDN",
			variant:    "darwin",
			files:      map[string]string{"/proc/sys/kernel/hostname": "\n"},
			lookupHost: func(string) ([]string, error) { return nil, errors.New("unused") },
			lookupAddr: func(string) ([]string, error) { return nil, errors.New("unused") },
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(t, nil, errNotOnPath, nil, errNotOnPath)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal(hostname.Info{}, *info)
			},
		},
		{
			// With an hour between retries, a retry that sleeps through the
			// cancel would hang the test.
			name:    "darwin: a cancelled ctx stops the resolver retries",
			variant: "darwin",
			files:   map[string]string{"/proc/sys/kernel/hostname": "kernel-short\n"},
			lookupHost: func(string) ([]string, error) {
				cancelledCalls++

				return nil, context.Canceled
			},
			lookupAddr: emptyLookupAddr,
			exec: func(t *testing.T) source.Runner {
				return hostnameExec(
					t,
					nil,
					errNotOnPath,
					[]byte("Friendly Name\n"),
					nil,
				)
			},
			cancelled: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostname.Info)
				s.Require().True(ok)
				s.Equal("kernel-short", info.FQDN)
				s.Equal(1, cancelledCalls)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			// The generated resolver answers with the row's lookups.
			resolver := mocks.NewMockResolver(gomock.NewController(s.T()))
			resolver.EXPECT().
				LookupHost(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, host string) ([]string, error) { return tt.lookupHost(host) }).
				AnyTimes()
			resolver.EXPECT().
				LookupAddr(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, addr string) ([]string, error) { return tt.lookupAddr(addr) }).
				AnyTimes()

			var c hostname.Collector
			switch tt.variant {
			case "linux":
				c = &hostname.Linux{Exec: tt.exec(s.T()), FS: fs, Resolver: resolver}
			case "darwin":
				c = &hostname.Darwin{Exec: tt.exec(s.T()), FS: fs, Resolver: resolver}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancelled {
				cancel()
			}
			tt.validateFunc(c.Collect(ctx, nil))
		})
	}
}
