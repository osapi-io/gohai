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
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type RootPublicTestSuite struct {
	suite.Suite
}

func TestRootPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RootPublicTestSuite))
}

// failWriter fails each write, as a closed pipe does.
type failWriter struct{}

func (failWriter) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("broken pipe")
}

func (s *RootPublicTestSuite) TestExecute() {
	dir := s.T().TempDir()
	valid := filepath.Join(dir, "valid.json")
	invalid := filepath.Join(dir, "invalid.json")
	var facts bytes.Buffer
	s.Require().Zero(cmd.Execute(context.Background(),
		[]string{"collect", "--no-defaults", "--collector.uptime"},
		strings.NewReader(""), &facts, io.Discard))
	s.Require().NoError(os.WriteFile(valid, facts.Bytes(), 0o600))
	s.Require().NoError(os.WriteFile(invalid, []byte(`{"cpu": 7}`), 0o600))
	devNull, err := os.Open(os.DevNull)
	s.Require().NoError(err)
	defer func() { _ = devNull.Close() }()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name        string
		ctx         context.Context
		args        []string
		in          io.Reader
		out         io.Writer
		newRegistry func() (*gohai.PublicRegistry, error)
		// setup, when set, runs before the command and returns its restore.
		setup        func() func()
		validateFunc func(code int, out, errOut string)
	}{
		{
			name: "collect reports a registry that cannot be built",
			args: []string{"collect"},
			newRegistry: func() (*gohai.PublicRegistry, error) {
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
			newRegistry: func() (*gohai.PublicRegistry, error) {
				return nil, errors.New("duplicate")
			},
			validateFunc: func(code int, out, errOut string) {
				s.Equal(1, code)
				s.Empty(out)
				s.Equal("error: duplicate\n", errOut)
			},
		},
		{
			name: "no command prints the help, with the banner",
			args: nil,
			validateFunc: func(code int, out, _ string) {
				s.Zero(code)
				s.Contains(out, "Collect system facts")
				s.Contains(out, "collector")
			},
		},
		{
			name: "help of a subcommand has no banner",
			args: []string{"collect", "--help"},
			validateFunc: func(code int, out, _ string) {
				s.Zero(code)
				s.Contains(out, "--format ohai")
			},
		},
		{
			name: "collect prints the facts of the chosen collectors as JSON",
			args: []string{"collect", "--no-defaults", "--collector.uptime", "--pretty"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
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
				s.Zero(code, errOut)
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
				s.Zero(code, errOut)
				s.Contains(out, "uptime.")
				s.Contains(out, "=")
			},
		},
		{
			name: "collect --format ocsf prints an inventory event",
			args: []string{"collect", "--no-defaults", "--collector.uptime", "--format", "ocsf"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
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
		{
			name: "validate passes a valid file",
			args: []string{"validate", "--file", valid},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "schema valid")
			},
		},
		{
			name: "validate reads stdin",
			args: []string{"validate"},
			in:   bytes.NewReader(facts.Bytes()),
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "schema valid")
			},
		},
		{
			name: "validate fails a document that breaks the schema",
			args: []string{"validate", "--file", invalid},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "schema validation failed")
			},
		},
		{
			name: "validate fails a file that is missing",
			args: []string{"validate", "--file", filepath.Join(dir, "missing.json")},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "read file")
			},
		},
		{
			name: "validate refuses a terminal on stdin",
			args: []string{"validate"},
			in:   devNull,
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "no input")
			},
		},
		{
			name: "validate refuses empty stdin",
			args: []string{"validate"},
			in:   strings.NewReader(""),
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "no input")
			},
		},
		{
			name: "validate fails when stdin fails",
			args: []string{"validate"},
			in:   iotest.ErrReader(errors.New("stdin closed")),
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "read stdin: stdin closed")
			},
		},
		{
			name: "version prints the build as JSON",
			args: []string{"version"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
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
			name: "collector list prints a table",
			args: []string{"collector", "list"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "NAME")
				s.Contains(out, "DEPENDS ON")
				s.Contains(out, "none")
			},
		},
		{
			name: "collector list --json prints each collector",
			args: []string{"collector", "list", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				var specs []gohai.CollectorSpec
				s.Require().NoError(json.Unmarshal([]byte(out), &specs))
				s.NotEmpty(specs)
			},
		},
		{
			name: "collector get shows the fields and the reads",
			args: []string{"collector", "get", "--name", "uptime"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "Collector:    uptime")
				s.Contains(out, "Reads on")
			},
		},
		{
			name: "collector get of a cloud collector names its network",
			args: []string{"collector", "get", "--name", "ec2"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "Network:")
				s.Contains(out, "nothing")
			},
		},
		{
			name: "collector get --json",
			args: []string{"collector", "get", "--name", "uptime", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
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
			name: "collector spec get prints each part",
			args: []string{"collector", "spec", "get"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "Use:")
			},
		},
		{
			name: "collector spec get --json",
			args: []string{"collector", "spec", "get", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				var terms []gohai.Term
				s.Require().NoError(json.Unmarshal([]byte(out), &terms))
				s.NotEmpty(terms)
			},
		},
		{
			name: "collector validate passes every collector",
			args: []string{"collector", "validate"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Contains(out, "Every collector passes each rule.")
			},
		},
		{
			name: "collector validate --json prints no problems",
			args: []string{"collector", "validate", "--json"},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code, errOut)
				s.Equal("[]", strings.TrimSpace(out))
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
		{
			name: "collector get fails when its output fails",
			args: []string{"collector", "get", "--name", "uptime"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
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
		{
			name: "collector validate --json fails when its output fails",
			args: []string{"collector", "validate", "--json"},
			out:  failWriter{},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "broken pipe")
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

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.newRegistry != nil {
				defer cmd.SetNewRegistryFn(tt.newRegistry)()
			}
			if tt.setup != nil {
				defer tt.setup()()
			}
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			in := tt.in
			if in == nil {
				in = strings.NewReader("")
			}
			var out, errOut bytes.Buffer
			w := tt.out
			if w == nil {
				w = &out
			}
			code := cmd.Execute(ctx, tt.args, in, w, &errOut)
			tt.validateFunc(code, out.String(), errOut.String())
		})
	}
}
