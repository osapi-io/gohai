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

// Package collector defines the Collector interface gohai uses to plug
// fact collectors into the registry.
package collector

import "strings"

// DsclPath reads one line of `dscl . -list <path> <attr>` whose values are
// paths, such as NFSHomeDirectory or UserShell. It returns the record name
// and its first path. dscl puts a space between the name, its padding and
// each value, so a value is not split on every space: a path with a space,
// such as /Users/Jo Smith, is one value, and a second value starts at a
// space followed by a slash, as in "root /var/root /private/var/root". ok is
// false for a line with no value.
func DsclPath(
	line string,
) (string, string, bool) {
	line = strings.TrimSpace(line)
	name, rest, ok := strings.Cut(line, " ")
	if !ok {
		return "", "", false
	}
	// The line is trimmed, so the rest after the first space has a value.
	rest = strings.TrimSpace(rest)
	if i := strings.Index(rest, " /"); i >= 0 {
		rest = rest[:i]
	}

	return name, strings.TrimSpace(rest), true
}
