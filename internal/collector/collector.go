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

// PriorResults is the map of already-completed collector outputs
// passed to each collector's Collect call. Access typed values via
// GetDep — that handles the runtime type assertion for you.
type PriorResults map[string]any

// GetDep returns the prior collector's result cast to T. ok is false
// when the collector wasn't in prior (disabled or missing) or when the
// stored value's type doesn't match T. Compile-time type safety: the
// caller names T explicitly (GetDep[*dmi.Info](prior, "dmi")), so a
// type mismatch between what the dependency returns and what the
// caller expects surfaces at compile time in the generic instantiation.
func GetDep[T any](
	prior PriorResults,
	name string,
) (T, bool) {
	var zero T
	raw, ok := prior[name]
	if !ok {
		return zero, false
	}
	v, ok := raw.(T)
	return v, ok
}

// Category groups related collectors for bulk enable/disable. Values
// are lowercase free-form strings; the set below is the canonical
// roster documented in docs/collectors/README.md. New categories can
// be added without code changes — the registry and SDK treat them as
// opaque tags. Keep this list in sync with the docs when the roster
// changes.
const (
	CategorySystem         = "system"
	CategoryHardware       = "hardware"
	CategoryNetwork        = "network"
	CategoryCloud          = "cloud"
	CategoryVirtualization = "virtualization"
	CategorySecurity       = "security"
	CategorySoftware       = "software"
	CategoryUsers          = "users"
	CategoryLinux          = "linux"
	CategoryMisc           = "misc"
)

// Collector is the interface every fact collector must implement.
type Collector interface {
	// Name returns the unique identifier (e.g., "platform").
	Name() string

	// Category returns the group this collector belongs to — "system",
	// "hardware", "cloud", "network", etc. Used by the SDK's
	// WithCategory option and the CLI's --category flag to enable
	// sets of related collectors at once.
	Category() string

	// DefaultEnabled reports whether this collector should run when no
	// explicit enable/disable flags are provided. Most collectors return
	// true; heavy or privileged collectors (ssh host keys, full package
	// inventory, full service list) return false and require the caller
	// to opt in via --collector.<name>.
	DefaultEnabled() bool

	// Dependencies returns names of collectors that must run before this one.
	Dependencies() []string

	// Collect gathers facts and returns a typed struct (or nil if not
	// supported). prior contains the typed outputs of collectors that
	// finished before this one (always includes everything declared in
	// Dependencies; may include additional upstream siblings). Access
	// typed values via GetDep[T].
	Collect(
		ctx context.Context,
		prior PriorResults,
	) (any, error)
}
