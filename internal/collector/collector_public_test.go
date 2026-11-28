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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type CollectorPublicTestSuite struct {
	suite.Suite
}

func (s *CollectorPublicTestSuite) TestAbsent() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(bool)
	}{
		{
			name:         "a file that does not exist is absent",
			err:          fmt.Errorf("read: %w", fs.ErrNotExist),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a command that is not on the PATH is absent",
			err:          fmt.Errorf("exec ps: %w", exec.ErrNotFound),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a denied read is not absent",
			err:          fmt.Errorf("read: %w", fs.ErrPermission),
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "a command that failed is not absent",
			err:          errors.New("exit status 1"),
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "no error is not absent",
			validateFunc: func(got bool) { s.False(got) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.Absent(tt.err))
		})
	}
}

func (s *CollectorPublicTestSuite) TestCancelled() {
	done, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name         string
		ctx          context.Context
		err          error
		validateFunc func(bool)
	}{
		{
			name:         "a done context is cancelled",
			ctx:          done,
			err:          errors.New("exit status 1"),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "an error that wraps context.Canceled is cancelled",
			ctx:          context.Background(),
			err:          fmt.Errorf("lscpu: %w", context.Canceled),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "an error that wraps context.DeadlineExceeded is cancelled",
			ctx:          context.Background(),
			err:          fmt.Errorf("lscpu: %w", context.DeadlineExceeded),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a failed command with a live context is not cancelled",
			ctx:          context.Background(),
			err:          errors.New("exit status 1"),
			validateFunc: func(got bool) { s.False(got) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.Cancelled(tt.ctx, tt.err))
		})
	}
}

func (s *CollectorPublicTestSuite) TestStatusOf() {
	tests := []struct {
		name         string
		out          any
		err          error
		validateFunc func(collector.Status)
	}{
		{
			name: "a fact and no error is ok",
			out:  "x",
			validateFunc: func(got collector.Status) {
				s.Equal(collector.StatusOK, got)
			},
		},
		{
			name: "no fact and no error is absent",
			validateFunc: func(got collector.Status) {
				s.Equal(collector.StatusAbsent, got)
			},
		},
		{
			name: "a wrapped ErrUnsupported is unsupported",
			err:  fmt.Errorf("darwin: %w", collector.ErrUnsupported),
			validateFunc: func(got collector.Status) {
				s.Equal(collector.StatusUnsupported, got)
			},
		},
		{
			name: "an error and no fact is failed",
			err:  errors.New("boom"),
			validateFunc: func(got collector.Status) {
				s.Equal(collector.StatusFailed, got)
			},
		},
		{
			name: "a fact and an error is partial",
			out:  "x",
			err:  errors.New("one read failed"),
			validateFunc: func(got collector.Status) {
				s.Equal(collector.StatusPartial, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.StatusOf(tt.out, tt.err))
		})
	}
}

func (s *CollectorPublicTestSuite) TestResult() {
	type info struct{ Name string }
	failed := errors.New("exit status 1")
	tests := []struct {
		name         string
		v            *info
		err          error
		validateFunc func(any, error)
	}{
		{
			name: "a value is the result",
			v:    &info{Name: "eth0"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&info{Name: "eth0"}, got)
			},
		},
		{
			name: "a nil value is an untyped nil",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name: "a nil value keeps its error",
			err:  failed,
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, failed)
				s.Exactly(nil, got)
			},
		},
		{
			name: "a value keeps its error",
			v:    &info{Name: "eth0"},
			err:  failed,
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, failed)
				s.Equal(&info{Name: "eth0"}, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.Result(tt.v, tt.err))
		})
	}
}

func TestCollectorPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CollectorPublicTestSuite))
}

func (s *CollectorPublicTestSuite) TestCategories() {
	tests := []struct {
		name         string
		validateFunc func([]collector.Category)
	}{
		{
			name: "lists the ten categories, cloud and misc among them",
			validateFunc: func(cats []collector.Category) {
				s.Len(cats, 10)
				s.Contains(cats, collector.CategoryCloud)
				s.Contains(cats, collector.CategoryMisc)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.Categories())
		})
	}
}

func (s *CollectorPublicTestSuite) TestGetDep() {
	prior := collector.PriorResults{
		"typed":    "hello",
		"wrong":    42,
		"nil-slot": nil,
	}

	tests := []struct {
		name         string
		lookup       string
		validateFunc func(string, bool)
	}{
		{
			name:   "matching type returns the value",
			lookup: "typed",
			validateFunc: func(got string, ok bool) {
				s.True(ok)
				s.Equal("hello", got)
			},
		},
		{
			name:   "missing key returns ok=false",
			lookup: "missing",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:   "type mismatch returns ok=false",
			lookup: "wrong",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:   "nil-valued any does not type-assert",
			lookup: "nil-slot",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.GetDep[string](prior, tt.lookup))
		})
	}
}
