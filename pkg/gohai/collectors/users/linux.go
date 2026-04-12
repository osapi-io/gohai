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

package users

import "context"

// Linux collects logged-in sessions on Linux via gopsutil (utmp).
type Linux struct {
	base

	SessionsFn func(context.Context) ([]Session, error)
}

// NewLinux returns a Linux variant wired to gopsutil.
func NewLinux() *Linux {
	return &Linux{SessionsFn: listSessions}
}

// Collect returns logged-in session Info.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	ss, err := l.SessionsFn(ctx)
	if err != nil {
		return nil, err
	}
	return &Info{LoggedIn: ss}, nil
}
