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

package packagemgr

import "context"

// RHEL covers rhel / redhat / centos / fedora / rocky / alma /
// amazon / oracle — dnf/yum-based hosts. Prefers dnf (RHEL 8+),
// falls back to yum (RHEL 7 and earlier, Amazon Linux 2).
type RHEL struct {
	base
}

// NewRHEL returns an RHEL variant.
func NewRHEL() *RHEL {
	return &RHEL{}
}

// Collect returns dnf if present, else yum.
func (r *RHEL) Collect(
	_ context.Context,
) (any, error) {
	name, path := firstFound("dnf", "yum")
	return &Info{Name: name, Path: path}, nil
}
