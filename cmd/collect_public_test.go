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
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type CollectPublicTestSuite struct {
	suite.Suite
}

func TestCollectPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CollectPublicTestSuite))
}

func (s *CollectPublicTestSuite) TestRegisterCollectorFlags() {
	tests := []executeCase{
		{
			name: "collect reports a registry that cannot be built",
			args: []string{"collect"},
			newRegistry: func() (gohai.Registry, error) {
				return nil, errors.New("duplicate")
			},
			validateFunc: func(code int, _, errOut string) {
				s.NotZero(code)
				s.Contains(errOut, "duplicate")
			},
		},
		{
			name: "collect --list-collectors reports a registry that cannot be built",
			args: []string{"collect", "--list-collectors"},
			newRegistry: func() (gohai.Registry, error) {
				return nil, errors.New("duplicate")
			},
			validateFunc: func(code int, out, errOut string) {
				s.Equal(1, code)
				s.Empty(out)
				s.Equal("error: duplicate\n", errOut)
			},
		},
	}

	runExecute(&s.Suite, tests)
}

func (s *CollectPublicTestSuite) TestRunCollect() {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []executeCase{
		{
			name: "collect prints the facts of the chosen collectors as JSON",
			args: []string{"collect", "--no-defaults", "--collector.uptime", "--pretty"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				var facts map[string]any
				s.Require().NoError(json.Unmarshal([]byte(out), &facts))
				s.Contains(facts, "uptime")
			},
		},
		{
			name: "collect with defaults, a disabled collector, a category and timings",
			args: []string{
				"collect", "--no-collector.uptime", "--category", "misc", "--with-timings",
			},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				var facts map[string]any
				s.Require().NoError(json.Unmarshal([]byte(out), &facts))
				s.NotContains(facts, "uptime")
				s.Contains(facts, "_timings")
			},
		},
		{
			name: "collect --flat prints key=value lines",
			args: []string{"collect", "--no-defaults", "--collector.uptime", "--flat"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "uptime.")
				s.Contains(out, "=")
			},
		},
		{
			name: "collect --format ocsf prints an inventory event",
			args: []string{"collect", "--no-defaults", "--collector.uptime", "--format", "ocsf"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, `"class_uid":5001`)
			},
		},
		{
			name: "collect --list-collectors lists them",
			args: []string{"collect", "--list-collectors"},
			validateFunc: func(code int, out, _ string) {
				s.Zero(code)
				s.Contains(out, "uptime")
			},
		},
		{
			name: "collect refuses an unknown format",
			args: []string{"collect", "--format", "xml"},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, `unknown format "xml"`)
			},
		},
		{
			name: "collect refuses --flat with --format ocsf",
			args: []string{
				"collect",
				"--no-defaults",
				"--collector.uptime",
				"--format",
				"ocsf",
				"--flat",
			},
			validateFunc: func(code int, out, errOut string) {
				s.Equal(1, code)
				s.Empty(out)
				s.Contains(errOut, "--flat does not apply to --format ocsf")
			},
		},
		{
			name: "collect refuses --with-timings with --format ocsf",
			args: []string{
				"collect",
				"--no-defaults",
				"--collector.uptime",
				"--format",
				"ocsf",
				"--with-timings",
			},
			validateFunc: func(code int, out, errOut string) {
				s.Equal(1, code)
				s.Empty(out)
				s.Contains(errOut, "--with-timings does not apply to --format ocsf")
			},
		},
		{
			name: "collect refuses a collector both enabled and disabled",
			args: []string{
				"collect",
				"--no-defaults",
				"--collector.uptime",
				"--no-collector.uptime",
			},
			validateFunc: func(code int, out, errOut string) {
				s.Equal(1, code)
				s.Empty(out)
				s.Contains(
					errOut,
					"--collector.uptime and --no-collector.uptime cannot both be set",
				)
			},
		},
		{
			name: "collect refuses an unknown category",
			args: []string{"collect", "--no-defaults", "--category", "nope"},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "nope")
			},
		},
		{
			name: "collect fails when its context is cancelled",
			ctx:  cancelled,
			args: []string{"collect", "--no-defaults", "--collector.uptime"},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "context canceled")
			},
		},
	}

	runExecute(&s.Suite, tests)
}
