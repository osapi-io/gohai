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

package rootgroup_test

import (
	"context"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rootgroup"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*rootgroup.Linux)(nil)
	_ collector.Collector = (*rootgroup.Darwin)(nil)
)

type RootGroupPublicTestSuite struct {
	suite.Suite
}

func TestRootGroupPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RootGroupPublicTestSuite))
}

func (s *RootGroupPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(rootgroup.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c rootgroup.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := rootgroup.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = rootgroup.New()
			}
			s.Equal("root_group", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *RootGroupPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		want    string
		wantErr string
	}{
		{
			name:    "linux: root's gid 0 is root",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "# users\nroot:x:0:0:root:/root:/bin/bash\n",
				"/etc/group":  "bad\nroot:x:0:\n",
			},
			want: "root",
		},
		{
			name:    "darwin: root's gid 0 is wheel",
			variant: "darwin",
			files: map[string]string{
				"/etc/passwd": "root:*:0:0:System Administrator:/var/root:/bin/sh\n",
				"/etc/group":  "wheel:*:0:root\n",
			},
			want: "wheel",
		},
		{
			name:    "linux: no /etc/passwd",
			variant: "linux",
			files:   map[string]string{"/etc/group": "root:x:0:\n"},
			wantErr: "lookup user root",
		},
		{
			name:    "linux: no root in /etc/passwd",
			variant: "linux",
			files:   map[string]string{"/etc/passwd": "nobody:x:65534:65534::/:/bin/false\n"},
			wantErr: "lookup user root",
		},
		{
			name:    "linux: no group with root's gid",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/root:/bin/bash\n",
				"/etc/group":  "users:x:100:\n",
			},
			wantErr: "lookup gid 0",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			var c rootgroup.Collector = &rootgroup.Linux{FS: fs}
			if tt.variant == "darwin" {
				c = &rootgroup.Darwin{FS: fs}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(&rootgroup.Info{Name: tt.want}, got)
		})
	}
}
