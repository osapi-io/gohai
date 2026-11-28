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

package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
)

type VersionPublicTestSuite struct {
	suite.Suite
}

func TestVersionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VersionPublicTestSuite))
}

func (s *VersionPublicTestSuite) TestBuildVersion() {
	tests := []executeCase{
		{
			name: "version prints the build as JSON",
			args: []string{"version"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				var v map[string]any
				s.Require().NoError(json.Unmarshal([]byte(out), &v))
				s.Contains(v, "gitVersion")
				s.Contains(v, "goVersion")
			},
		},
		{
			name: "version fails when its output fails",
			args: []string{"version"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "write version")
			},
		},
		{
			name: "version prints the build info as JSON",
			args: []string{"version"},
			setup: func() func() {
				return cmd.SetBuildInfo("v1.2.3", "abc123", "clean", "2026-11-26", "goreleaser")
			},
			validateFunc: func(code int, out, _ string) {
				s.Require().Zero(code)
				var v map[string]any
				s.Require().NoError(json.Unmarshal([]byte(out), &v))
				s.Equal("v1.2.3", v["gitVersion"])
				s.Equal("abc123", v["gitCommit"])
				s.Equal("clean", v["gitTreeState"])
				s.Equal("2026-11-26", v["buildDate"])
				s.Equal("goreleaser", v["builtBy"])
			},
		},
	}

	runExecute(&s.Suite, tests)
}
