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

package cloudmetadata

import "regexp"

// safeName matches a name that a metadata listing may hold and that is safe
// to join into a request path: letters, digits and the characters _ . : -.
var safeName = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

// SafeName reports whether name, a child name from a metadata directory
// listing, is safe to join into a request path. A trailing slash, which a
// listing gives a subdirectory, is ignored. A name with "..", "/", "?", "#",
// a space or any other character is not safe, because it changes the path
// or the query of the request.
func SafeName(
	name string,
) bool {
	name = trimDirSlash(name)
	return name != ".." && name != "." && safeName.MatchString(name)
}

// trimDirSlash removes one trailing slash, which marks a subdirectory in a
// metadata listing.
func trimDirSlash(
	name string,
) string {
	if len(name) > 0 && name[len(name)-1] == '/' {
		return name[:len(name)-1]
	}
	return name
}
