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

// Package pattern holds the regular expressions that gohai keeps in
// package variables. regexp.MustCompile panics on a bad expression, and code
// in gohai does not panic. New compiles without a panic: a bad expression
// gives a Pattern that matches nothing, and Errors reports it. A test that
// links every collector checks that Errors is empty, so a bad expression
// fails the build of the tests, not a host.
package pattern

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
)

var (
	mu     sync.Mutex
	failed []error
)

// Pattern is a compiled regular expression. A Pattern whose expression did
// not compile matches nothing.
type Pattern struct {
	re *regexp.Regexp
}

// New compiles expr. When expr does not compile, New records the error for
// Errors and returns a Pattern that matches nothing.
func New(
	expr string,
) *Pattern {
	re, err := regexp.Compile(expr)
	if err != nil {
		mu.Lock()
		failed = append(failed, fmt.Errorf("pattern %q: %w", expr, err))
		mu.Unlock()

		return &Pattern{}
	}

	return &Pattern{re: re}
}

// Errors returns the errors of each expression that New could not compile,
// joined, or nil when each one compiled.
func Errors() error {
	mu.Lock()
	defer mu.Unlock()

	return errors.Join(failed...)
}

// MatchString reports whether s contains a match of the pattern.
func (p *Pattern) MatchString(
	s string,
) bool {
	if p.re == nil {
		return false
	}

	return p.re.MatchString(s)
}

// FindStringSubmatch returns the leftmost match in s and its submatches, or
// nil when there is no match.
func (p *Pattern) FindStringSubmatch(
	s string,
) []string {
	if p.re == nil {
		return nil
	}

	return p.re.FindStringSubmatch(s)
}

// FindSubmatch returns the leftmost match in b and its submatches, or nil
// when there is no match.
func (p *Pattern) FindSubmatch(
	b []byte,
) [][]byte {
	if p.re == nil {
		return nil
	}

	return p.re.FindSubmatch(b)
}

// FindAllString returns up to n successive matches in s, or all of them when
// n is negative, or nil when there is no match.
func (p *Pattern) FindAllString(
	s string,
	n int,
) []string {
	if p.re == nil {
		return nil
	}

	return p.re.FindAllString(s, n)
}

// FindAllStringSubmatch returns up to n successive matches in s and their
// submatches, or all of them when n is negative, or nil when there is no
// match.
func (p *Pattern) FindAllStringSubmatch(
	s string,
	n int,
) [][]string {
	if p.re == nil {
		return nil
	}

	return p.re.FindAllStringSubmatch(s, n)
}

// ReplaceAllString returns s with each match replaced by repl, which can
// name submatches as in regexp.Regexp.Expand. A pattern that matches
// nothing returns s unchanged.
func (p *Pattern) ReplaceAllString(
	s string,
	repl string,
) string {
	if p.re == nil {
		return s
	}

	return p.re.ReplaceAllString(s, repl)
}
