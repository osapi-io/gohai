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

package memory

import "context"

// Darwin collects memory usage on macOS via gopsutil (which uses mach
// vm_stat APIs under the hood). Buffers/Cached fields stay zero on
// darwin — those are Linux-only concepts.
type Darwin struct {
	base

	ReadFn func(context.Context) (*Info, error)
}

// NewDarwin returns a Darwin variant wired to gopsutil.
func NewDarwin() *Darwin {
	return &Darwin{ReadFn: readMemory}
}

// Collect returns the memory Info.
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	return d.ReadFn(ctx)
}
