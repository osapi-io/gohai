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

package cmd

import "github.com/osapi-io/gohai/pkg/gohai"

// WriteProblems is writeProblems, for a list of problems: the built-in
// collectors break no rule, so the CLI never reaches one.
var WriteProblems = writeProblems

// SetBuildInfo sets the build's ldflags variables, and returns a func that
// puts them back.
func SetBuildInfo(
	v, c, tree, d, by string,
) func() {
	old := []string{version, commit, treestate, date, builtBy}
	version, commit, treestate, date, builtBy = v, c, tree, d, by

	return func() {
		version, commit, treestate, date, builtBy = old[0], old[1], old[2], old[3], old[4]
	}
}

// SetNewRegistryFn replaces how the collect command builds the registry, and
// returns a func that puts it back.
func SetNewRegistryFn(
	fn func() (*gohai.PublicRegistry, error),
) func() {
	old := newRegistryFn
	newRegistryFn = fn

	return func() { newRegistryFn = old }
}
