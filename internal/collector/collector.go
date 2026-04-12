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

import "context"

// Collector is the interface every fact collector must implement.
type Collector interface {
	// Name returns the unique identifier (e.g., "platform").
	Name() string

	// DefaultEnabled reports whether this collector should run when no
	// explicit enable/disable flags are provided. Most collectors return
	// true; heavy or privileged collectors (ssh host keys, full package
	// inventory, full service list) return false and require the caller
	// to opt in via --collector.<name>.
	DefaultEnabled() bool

	// Dependencies returns names of collectors that must run before this one.
	Dependencies() []string

	// Collect gathers facts and returns a typed struct (or nil if not supported).
	Collect(
		ctx context.Context,
	) (any, error)
}
