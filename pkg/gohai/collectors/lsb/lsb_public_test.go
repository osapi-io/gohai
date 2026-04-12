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
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
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

func TestLSBPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LSBPublicTestSuite))
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

func (s *LSBPublicTestSuite) TestCollectLinux() {
	tests := []struct {
		name    string
		content string
		err     error
		wantErr bool
		want    lsb.Info
	}{
		{
			name: "ubuntu noble",
			content: `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=24.04
DISTRIB_CODENAME=noble
DISTRIB_DESCRIPTION="Ubuntu 24.04 LTS"
`,
			want: lsb.Info{
				ID:          "Ubuntu",
				Release:     "24.04",
				Codename:    "noble",
				Description: "Ubuntu 24.04 LTS",
			},
		},
		{
			name: "missing file soft-misses",
			err:  os.ErrNotExist,
			want: lsb.Info{},
		},
		{
			name:    "read error propagated",
			err:     errors.New("perm"),
			wantErr: true,
		},
		{
			name:    "malformed lines skipped",
			content: "# comment\n\nbroken\nDISTRIB_ID=Debian\n",
			want:    lsb.Info{ID: "Debian"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &lsb.Linux{
				ReadFileFn: func(string) ([]byte, error) {
					if tt.err != nil {
						return nil, tt.err
					}
					return []byte(tt.content), nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*lsb.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *LSBPublicTestSuite) TestCollectDarwin() {
	c := lsb.NewDarwin()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	s.Nil(got)
}
