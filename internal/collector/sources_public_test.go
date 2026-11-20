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

package collector_test

import (
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

type SourcesPublicTestSuite struct {
	suite.Suite
}

// variant is a collector variant with each source field.
type variant struct {
	FS   avfs.VFS
	Exec executor.Executor
	Now  func() time.Time
	Name string
}

// bare is a variant with no source field.
type bare struct{ Name string }

func (s *SourcesPublicTestSuite) TestIsDarwinAndFamily() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()
	platform.Detect = func() string { return family.Debian }

	tests := []struct {
		name       string
		src        source.Sources
		wantFamily string
		wantDarwin bool
	}{
		{
			name: "when src names darwin", src: source.Sources{Family: family.Darwin},
			wantFamily: family.Darwin, wantDarwin: true,
		},
		{
			name: "when src names a Linux family", src: source.Sources{Family: family.RHEL},
			wantFamily: family.RHEL, wantDarwin: false,
		},
		{
			name: "when src names none, the host's", src: source.Sources{},
			wantFamily: family.Debian, wantDarwin: false,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Equal(tc.wantFamily, collector.Family(tc.src))
			s.Equal(tc.wantDarwin, collector.IsDarwin(tc.src))
		})
	}
}

func (s *SourcesPublicTestSuite) TestWire() {
	fs := memfs.New()
	now := func() time.Time { return time.Unix(42, 0) }
	exec := source.Commands()

	v := collector.Wire(&variant{Name: "kept"}, source.Sources{
		FS: fs, Exec: exec, Now: now, Family: family.Debian,
	})
	s.Same(fs, v.FS)
	s.Equal(exec, v.Exec)
	s.Equal(time.Unix(42, 0), v.Now())
	s.Equal("kept", v.Name, "a field that is not a source is not changed")

	d := collector.Wire(&variant{}, source.Sources{Family: family.Debian})
	s.NotNil(d.FS, "a zero source is the real host's")
	s.NotNil(d.Exec)
	s.NotNil(d.Now)

	b := collector.Wire(&bare{Name: "b"}, source.Sources{FS: fs})
	s.Equal("b", b.Name, "a variant with no source field is unchanged")

	s.Equal(bare{Name: "v"}, collector.Wire(bare{Name: "v"}, source.Sources{}),
		"a value that is not a pointer to a struct is returned as it is")
}

func TestSourcesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SourcesPublicTestSuite))
}
