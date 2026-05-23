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
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cli"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type OutputPublicTestSuite struct {
	suite.Suite
}

func TestOutputPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OutputPublicTestSuite))
}

func (s *OutputPublicTestSuite) TestWriteOutput() {
	tests := []struct {
		name   string
		pretty bool
		flat   bool
		verify func(string)
	}{
		{
			name:   "json compact",
			pretty: false,
			flat:   false,
			verify: func(out string) {
				s.Contains(out, "collect_time")
			},
		},
		{
			name:   "json pretty",
			pretty: true,
			flat:   false,
			verify: func(out string) {
				s.Contains(out, "  ")
			},
		},
		{
			name:   "flat format",
			pretty: false,
			flat:   true,
			verify: func(out string) {
				s.Contains(out, "collect_time=")
			},
		},
	}

	for _, tc := range tests {
		var buf bytes.Buffer
		err := cli.WriteOutput(&buf, &gohai.Facts{}, tc.pretty, tc.flat)

		s.NoError(err)
		tc.verify(buf.String())
	}
}

func (s *OutputPublicTestSuite) TestWriteJSON() {
	tests := []struct {
		name     string
		w        io.Writer
		pretty   bool
		wantErr  string
		validate func(string)
	}{
		{
			name:   "compact",
			w:      &bytes.Buffer{},
			pretty: false,
			validate: func(out string) {
				var m map[string]any
				s.NoError(json.Unmarshal([]byte(out), &m))
			},
		},
		{
			name:   "pretty",
			w:      &bytes.Buffer{},
			pretty: true,
			validate: func(out string) {
				s.Contains(out, "\n  ")
			},
		},
		{
			name:    "write error",
			w:       &errWriter{err: errors.New("disk full")},
			pretty:  false,
			wantErr: "write output",
		},
	}

	for _, tc := range tests {
		err := cli.WriteJSON(tc.w, &gohai.Facts{}, tc.pretty)

		if tc.wantErr != "" {
			s.ErrorContains(err, tc.wantErr)
		} else {
			s.NoError(err)
			tc.validate(tc.w.(*bytes.Buffer).String())
		}
	}
}

func (s *OutputPublicTestSuite) TestWriteFlat() {
	tests := []struct {
		name    string
		w       io.Writer
		wantErr string
	}{
		{
			name: "success",
			w:    &bytes.Buffer{},
		},
		{
			name:    "write error",
			w:       &errWriter{err: errors.New("disk full")},
			wantErr: "write flat output",
		},
	}

	for _, tc := range tests {
		err := cli.WriteFlat(tc.w, &gohai.Facts{})

		if tc.wantErr != "" {
			s.ErrorContains(err, tc.wantErr)
		} else {
			s.NoError(err)
			s.Contains(tc.w.(*bytes.Buffer).String(), "collect_time=")
		}
	}
}

func (s *OutputPublicTestSuite) TestWriteCollectorList() {
	tests := []struct {
		name    string
		w       io.Writer
		wantErr string
	}{
		{
			name: "success",
			w:    &bytes.Buffer{},
		},
		{
			name:    "write error on category header",
			w:       &errWriter{err: errors.New("disk full")},
			wantErr: "write collector list",
		},
		{
			name:    "write error on collector name",
			w:       &errWriter{err: errors.New("disk full"), failAfter: 1},
			wantErr: "write collector list",
		},
	}

	for _, tc := range tests {
		err := cli.WriteCollectorList(tc.w)

		if tc.wantErr != "" {
			s.ErrorContains(err, tc.wantErr)
		} else {
			s.NoError(err)
			out := tc.w.(*bytes.Buffer).String()
			s.Contains(out, "[")
			s.Contains(out, "platform")
		}
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
