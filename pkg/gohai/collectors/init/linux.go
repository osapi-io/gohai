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

package initd

import (
	"context"
	"os"
)

// Linux detects the init system by reading /proc/1/comm.
type Linux struct {
	base

	ReadFileFn func(string) ([]byte, error)
}

// NewLinux returns a Linux variant wired to os.ReadFile.
func NewLinux() *Linux {
	return &Linux{ReadFileFn: os.ReadFile}
}

// Collect reads /proc/1/comm and classifies the result. Missing or
// unreadable /proc/1/comm (rare — possible in restricted containers)
// returns an empty Name rather than erroring.
func (l *Linux) Collect(_ context.Context) (any, error) {
	b, err := l.ReadFileFn(proc1CommPath)
	if err != nil {
		return &Info{}, nil
	}
	return &Info{Name: classify(string(b))}, nil
}
