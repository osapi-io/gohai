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

package kernel

import "golang.org/x/sys/unix"

// BytesToString exposes the private bytesToString helper to the
// external kernel_test package.
var BytesToString = bytesToString

// DefaultUname exposes the private defaultUname helper.
var DefaultUname = defaultUname

// SetUnameSyscall swaps the private unix.Uname binding used by
// defaultUname for the duration of a test. Returns a restore func the
// caller must defer.
func SetUnameSyscall(
	fn func(*unix.Utsname) error,
) (restore func()) {
	orig := unameSyscall
	unameSyscall = fn
	return func() { unameSyscall = orig }
}
