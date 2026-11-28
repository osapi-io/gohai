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

package gohai_test

import (
	"context"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

type OptionsPublicTestSuite struct {
	suite.Suite
}

func TestOptionsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsPublicTestSuite))
}

func (s *OptionsPublicTestSuite) TestWithSources() {
	type row struct {
		name         string
		src          func() source.Sources
		collector    string
		validateFunc func(*gohai.Facts, error)
	}
	tests := []row{
		{
			name: "a collector reads the filesystem it is given",
			src: func() source.Sources {
				fs := memfs.New()
				s.Require().NoError(fs.MkdirAll("/etc", 0o755))
				s.Require().NoError(fs.WriteFile("/etc/os-release",
					[]byte("ID=gohaitest\nNAME=\"Gohai Test\"\nVERSION_ID=\"1.0\"\n"), 0o644))

				return source.Sources{FS: fs, Family: family.Debian}
			},
			collector: "os_release",
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts.OSRelease)
				s.Equal("gohaitest", facts.OSRelease.ID)
				s.Equal("1.0", facts.OSRelease.VersionID)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(
				gohai.WithSources(tt.src()),
				gohai.WithCollectors(tt.collector),
			)
			s.Require().NoError(err)
			tt.validateFunc(g.Collect(context.Background()))
		})
	}
}
