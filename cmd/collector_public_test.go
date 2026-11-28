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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
	"github.com/osapi-io/gohai/internal/spec"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type CollectorPublicTestSuite struct {
	suite.Suite
}

func TestCollectorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CollectorPublicTestSuite))
}

func (s *CollectorPublicTestSuite) TestWriteProblems() {
	problems := []spec.Problem{{
		Collector: "bad", Rule: "name", Message: "is not lowercase.", Fix: "rename it.",
	}}
	tests := []struct {
		name         string
		asJSON       bool
		validateFunc func(out string, err error)
	}{
		{
			name: "a problem prints its fix and fails",
			validateFunc: func(out string, err error) {
				s.Error(err)
				s.Equal("bad [name]: is not lowercase. Fix: rename it.\n", out)
			},
		},
		{
			name:   "a problem as JSON fails",
			asJSON: true,
			validateFunc: func(out string, err error) {
				s.Error(err)
				s.Contains(out, `"collector": "bad"`)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer
			err := cmd.WriteProblems(&out, problems, tt.asJSON)
			tt.validateFunc(out.String(), err)
		})
	}
}

func (s *CollectorPublicTestSuite) TestNewCollectorCommand() {
	tests := []executeCase{
		{
			name: "collector validate passes every collector",
			args: []string{"collector", "validate"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "Every collector passes each rule.")
			},
		},
		{
			name: "collector validate --json prints no problems",
			args: []string{"collector", "validate", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Equal("[]", strings.TrimSpace(out))
			},
		},
		{
			name: "collector validate --json fails when its output fails",
			args: []string{"collector", "validate", "--json"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
			},
		},
	}

	runExecute(&s.Suite, tests)
}

func (s *CollectorPublicTestSuite) TestWriteCollectorList() {
	tests := []executeCase{
		{
			name: "collector list prints a table",
			args: []string{"collector", "list"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "NAME")
				s.Contains(out, "DEPENDS ON")
				s.Contains(out, "none")
			},
		},
		{
			name: "collector list --json prints each collector",
			args: []string{"collector", "list", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				var specs []gohai.CollectorSpec
				s.Require().NoError(json.Unmarshal([]byte(out), &specs))
				s.NotEmpty(specs)
			},
		},
		{
			name: "collector list fails when its output fails",
			args: []string{"collector", "list"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
			},
		},
		{
			name: "collector list --json fails when its output fails",
			args: []string{"collector", "list", "--json"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
			},
		},
	}

	runExecute(&s.Suite, tests)
}

func (s *CollectorPublicTestSuite) TestWriteCollector() {
	tests := []executeCase{
		{
			name: "collector get shows the fields and the reads",
			args: []string{"collector", "get", "--name", "uptime"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "Collector:    uptime")
				s.Contains(out, "Reads on")
			},
		},
		{
			name: "collector get of a cloud collector names its network",
			args: []string{"collector", "get", "--name", "ec2"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "Network:")
				s.Contains(out, "nothing")
			},
		},
		{
			name: "collector get --json",
			args: []string{"collector", "get", "--name", "uptime", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, `"name": "uptime"`)
			},
		},
		{
			name: "collector get refuses an unknown name",
			args: []string{"collector", "get", "--name", "nope"},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, `no collector named "nope"`)
			},
		},
		{
			name: "collector get fails when its output fails",
			args: []string{"collector", "get", "--name", "uptime"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
			},
		},
	}

	runExecute(&s.Suite, tests)
}

func (s *CollectorPublicTestSuite) TestWriteTerms() {
	tests := []executeCase{
		{
			name: "collector spec get prints each part",
			args: []string{"collector", "spec", "get"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "Use:")
			},
		},
		{
			name: "collector spec get --json",
			args: []string{"collector", "spec", "get", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				var terms []gohai.Term
				s.Require().NoError(json.Unmarshal([]byte(out), &terms))
				s.NotEmpty(terms)
			},
		},
		{
			name: "collector spec get fails when its output fails",
			args: []string{"collector", "spec", "get"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
			},
		},
	}

	runExecute(&s.Suite, tests)
}
