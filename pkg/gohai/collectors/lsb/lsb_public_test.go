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

package lsb_test

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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
)

var (
	_ collector.Collector = (*lsb.Linux)(nil)
	_ collector.Collector = (*lsb.Darwin)(nil)
)

type LSBPublicTestSuite struct {
	suite.Suite
}

func TestLSBPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LSBPublicTestSuite))
}

// lsbReleaseExec returns a MockExecutor that canned-answers
// `lsb_release -a`.
func lsbReleaseExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsb_release", "-a").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *LSBPublicTestSuite) TestNew() {
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
			c := lsb.New()
			s.Equal("lsb", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*lsb.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*lsb.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *LSBPublicTestSuite) TestCollect() {
	cliFull := []byte(`Distributor ID:	Ubuntu
Description:	Ubuntu 24.04 LTS
Release:	24.04
Codename:	noble
`)

	tests := []struct {
		name    string
		variant string
		exec    func(*testing.T) executor.Executor
		wantNil bool
		want    lsb.Info
	}{
		{
			name:    "linux: CLI succeeds, four fields populated",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return lsbReleaseExec(t, cliFull, nil) },
			want: lsb.Info{
				ID: "Ubuntu", Release: "24.04", Codename: "noble",
				Description: "Ubuntu 24.04 LTS",
			},
		},
		{
			name:    "linux: CLI returns subset, only matching fields populated",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsbReleaseExec(t, []byte("Distributor ID:\tDebian\nRelease:\t12\n"), nil)
			},
			want: lsb.Info{ID: "Debian", Release: "12"},
		},
		{
			name:    "linux: CLI output with unmatched / no-colon lines: skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsbReleaseExec(t,
					[]byte("no colon line here\nunrelated: value\nDistributor ID:\tArch\n"),
					nil)
			},
			want: lsb.Info{ID: "Arch"},
		},
		{
			name:    "linux: CLI missing, empty Info no error",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return lsbReleaseExec(t, nil, errors.New("not found"))
			},
			want: lsb.Info{},
		},
		{
			name:    "linux: nil Exec, empty Info no error",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    lsb.Info{},
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c lsb.Collector
			switch tt.variant {
			case "linux":
				c = &lsb.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = lsb.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*lsb.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
