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

package cli_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cli"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
)

type OutputPublicTestSuite struct {
	suite.Suite
}

func TestOutputPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OutputPublicTestSuite))
}

// facts holds one fact, so the writers have a value to show.
var facts = &gohai.Facts{Load: &load.Info{One: 0.5, Five: 0.25, Fifteen: 0.125}}

// compactJSON is facts as compact JSON, with the newline WriteJSON adds.
const compactJSON = `{"load":{"one":0.5,"five":0.25,"fifteen":0.125},` +
	`"collect_time":"0001-01-01T00:00:00Z","collect_duration_ns":0}` + "\n"

// flatFacts is facts as WriteFlat writes them.
const flatFacts = "collect_duration_ns=0\n" +
	"collect_time=0001-01-01T00:00:00Z\n" +
	"load.fifteen=0.125\n" +
	"load.five=0.25\n" +
	"load.one=0.5\n"

func (s *OutputPublicTestSuite) TestWriteOutput() {
	tests := []struct {
		name         string
		pretty       bool
		flat         bool
		validateFunc func(string, error)
	}{
		{
			name: "compact JSON",
			validateFunc: func(out string, err error) {
				s.Require().NoError(err)
				s.Equal(compactJSON, out)
			},
		},
		{
			name:   "pretty JSON",
			pretty: true,
			validateFunc: func(out string, err error) {
				s.Require().NoError(err)
				s.JSONEq(compactJSON, out)
				s.Contains(out, "\n  \"load\": {\n    \"one\": 0.5,")
			},
		},
		{
			name:   "flat wins over pretty",
			pretty: true,
			flat:   true,
			validateFunc: func(out string, err error) {
				s.Require().NoError(err)
				s.Equal(flatFacts, out)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			err := cli.WriteOutput(&buf, facts, tc.pretty, tc.flat)

			tc.validateFunc(buf.String(), err)
		})
	}
}

func (s *OutputPublicTestSuite) TestWriteJSON() {
	tests := []struct {
		name         string
		w            io.Writer
		facts        *gohai.Facts
		pretty       bool
		validateFunc func(io.Writer, error)
	}{
		{
			name:  "compact",
			w:     &bytes.Buffer{},
			facts: facts,
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				s.Equal(compactJSON, w.(*bytes.Buffer).String())
			},
		},
		{
			name:   "pretty",
			w:      &bytes.Buffer{},
			facts:  facts,
			pretty: true,
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				out := w.(*bytes.Buffer).String()
				s.JSONEq(compactJSON, out)
				s.Contains(out, "\n  \"collect_time\": ")
			},
		},
		{
			name:  "write error",
			w:     &errWriter{err: errors.New("disk full")},
			facts: facts,
			validateFunc: func(_ io.Writer, err error) {
				s.EqualError(err, "write output: disk full")
			},
		},
		{
			name:  "a fact JSON cannot hold is an encode error",
			w:     &bytes.Buffer{},
			facts: &gohai.Facts{Load: &load.Info{One: math.NaN()}},
			validateFunc: func(w io.Writer, err error) {
				s.ErrorContains(err, "encode output: ")
				s.ErrorContains(err, "unsupported value: NaN")
				s.Empty(w.(*bytes.Buffer).String())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(tc.w, cli.WriteJSON(tc.w, tc.facts, tc.pretty))
		})
	}
}

func (s *OutputPublicTestSuite) TestWriteFlat() {
	tests := []struct {
		name         string
		facts        *gohai.Facts
		w            io.Writer
		validateFunc func(io.Writer, error)
	}{
		{
			name: "sorted key=value lines",
			w:    &bytes.Buffer{},
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				s.Equal(flatFacts, w.(*bytes.Buffer).String())
			},
		},
		{
			name: "write error",
			w:    &errWriter{err: errors.New("disk full")},
			validateFunc: func(_ io.Writer, err error) {
				s.EqualError(err, "write flat output: disk full")
			},
		},
		{
			name:  "facts that do not encode",
			facts: &gohai.Facts{Load: &load.Info{One: math.NaN()}},
			w:     &bytes.Buffer{},
			validateFunc: func(w io.Writer, err error) {
				s.EqualError(err, "write flat output: flatten facts: json: unsupported value: NaN")
				s.Empty(w.(*bytes.Buffer).String())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			f := facts
			if tc.facts != nil {
				f = tc.facts
			}
			tc.validateFunc(tc.w, cli.WriteFlat(tc.w, f))
		})
	}
}

func (s *OutputPublicTestSuite) TestWriteCollectorList() {
	reg, err := gohai.NewRegistry()
	s.Require().NoError(err)

	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(io.Writer, error)
	}{
		{
			name: "each category, then its collectors, in order",
			w:    &bytes.Buffer{},
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				out := w.(*bytes.Buffer).String()
				system := strings.Index(out, "[system]\n")
				s.Require().GreaterOrEqual(system, 0)
				s.Contains(out[system:], "\n  platform\n")
				s.True(strings.HasPrefix(out, "["))
			},
		},
		{
			name: "write error on category header",
			w:    &errWriter{err: errors.New("disk full")},
			validateFunc: func(_ io.Writer, err error) {
				s.EqualError(err, "write collector list: disk full")
			},
		},
		{
			name: "write error on collector name",
			w:    &errWriter{err: errors.New("disk full"), failAfter: 1},
			validateFunc: func(_ io.Writer, err error) {
				s.EqualError(err, "write collector list: disk full")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(tc.w, cli.WriteCollectorList(tc.w, reg))
		})
	}
}

// compactOCSF is the OCSF event of facts, which has no field for load.
const compactOCSF = `{"activity_id":2,"category_uid":5,"class_uid":5001,` +
	`"class_name":"Device Inventory Info","severity_id":1,` +
	`"time":-62135596800000,"type_uid":500102,` +
	`"metadata":{"version":"1.8.0","product":{"name":"gohai","vendor_name":"osapi-io"}},` +
	`"device":{"type_id":0}}` + "\n"

func (s *OutputPublicTestSuite) TestWriteOCSF() {
	tests := []struct {
		name         string
		w            io.Writer
		facts        *gohai.Facts
		pretty       bool
		validateFunc func(io.Writer, error)
	}{
		{
			name:  "compact",
			w:     &bytes.Buffer{},
			facts: facts,
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				s.Equal(compactOCSF, w.(*bytes.Buffer).String())
			},
		},
		{
			name:   "pretty",
			w:      &bytes.Buffer{},
			facts:  facts,
			pretty: true,
			validateFunc: func(w io.Writer, err error) {
				s.Require().NoError(err)
				out := w.(*bytes.Buffer).String()
				s.JSONEq(compactOCSF, out)
				s.Contains(out, "\n  \"class_uid\": 5001,")
			},
		},
		{
			name:  "write error",
			w:     &errWriter{err: errors.New("disk full")},
			facts: facts,
			validateFunc: func(_ io.Writer, err error) {
				s.EqualError(err, "write ocsf output: disk full")
			},
		},
		{
			name:  "a value JSON cannot hold is an encode error",
			w:     &bytes.Buffer{},
			facts: &gohai.Facts{CPU: &cpu.Info{Speed: math.NaN()}},
			validateFunc: func(w io.Writer, err error) {
				s.ErrorContains(err, "encode ocsf output: ")
				s.ErrorContains(err, "unsupported value: NaN")
				s.Empty(w.(*bytes.Buffer).String())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(tc.w, cli.WriteOCSF(tc.w, tc.facts, tc.pretty))
		})
	}
}

func (s *OutputPublicTestSuite) TestPrint() {
	tests := []struct {
		name         string
		s            string
		validateFunc func(string)
	}{
		{
			name:         "a line ends with a newline",
			s:            "hello",
			validateFunc: func(got string) { s.Equal("hello\n", got) },
		},
		{
			name:         "an empty line is a newline",
			validateFunc: func(got string) { s.Equal("\n", got) },
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			cli.Print(&buf, tc.s)
			tc.validateFunc(buf.String())
		})
	}
}

func (s *OutputPublicTestSuite) TestPrintf() {
	tests := []struct {
		name         string
		format       string
		args         []any
		validateFunc func(string)
	}{
		{
			name:         "formats with no newline",
			format:       "count: %d",
			args:         []any{42},
			validateFunc: func(got string) { s.Equal("count: 42", got) },
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			cli.Printf(&buf, tc.format, tc.args...)
			tc.validateFunc(buf.String())
		})
	}
}

type errWriter struct {
	err       error
	failAfter int
	count     int
}

func (w *errWriter) Write(
	p []byte,
) (int, error) {
	if w.count >= w.failAfter {
		return 0, w.err
	}

	w.count++

	return len(p), nil
}
