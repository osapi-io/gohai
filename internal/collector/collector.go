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

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
)

// Absent reports whether err says that the source of a read is not on
// the host: the file does not exist, or the command is not on the PATH.
// A collector gives an empty result, or no fact, for an absent source,
// and returns any other error (specs components/gohai/collectors.md,
// "When a read fails").
func Absent(
	err error,
) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, exec.ErrNotFound)
}

// ErrUnsupported is the error of a collector variant that has nothing to
// read on its platform, such as the macOS variant of a Linux-only collector.
// The registry gives it the status StatusUnsupported, not StatusFailed, and
// it is not an error of the run.
var ErrUnsupported = errors.New("not supported on this platform")

// Status is why a collector has its fact, or has none.
type Status string

// The statuses of a collector after a run.
const (
	// StatusOK is a collector that returned its fact.
	StatusOK Status = "ok"
	// StatusAbsent is a collector that found nothing on the host to report:
	// it returned no fact and no error.
	StatusAbsent Status = "absent"
	// StatusUnsupported is a collector with nothing to read on the platform:
	// it returned ErrUnsupported.
	StatusUnsupported Status = "unsupported"
	// StatusFailed is a collector that returned an error and no fact, or that
	// did not run because a collector it depends on failed.
	StatusFailed Status = "failed"
	// StatusPartial is a collector that returned a fact and an error: the
	// fact holds what it read before the error.
	StatusPartial Status = "partial"
)

// StatusOf is the status of a collector that returned out and err.
func StatusOf(
	out any,
	err error,
) Status {
	switch {
	case errors.Is(err, ErrUnsupported):
		return StatusUnsupported
	case err != nil && out != nil:
		return StatusPartial
	case err != nil:
		return StatusFailed
	case out == nil:
		return StatusAbsent
	default:
		return StatusOK
	}
}

// Result returns v as a Collect result. A nil v becomes an untyped nil, so
// the registry records no fact. A typed nil pointer in an any is not nil,
// and would put a null fact in the output.
func Result[T any](
	v *T,
	err error,
) (any, error) {
	if v == nil {
		return nil, err
	}

	return v, err
}

// PriorResults is the map of already-completed collector outputs
// passed to each collector's Collect call. Access typed values via
// GetDep, which does the runtime type assertion for you.
type PriorResults map[string]any

// GetDep returns the prior collector's result cast to T, such as
// GetDep[*dmi.Info](prior, "dmi"). ok is false when the collector wasn't in
// prior (disabled or missing) or when the stored value's type doesn't match
// T. The type check is at run time: a T that is not what the dependency
// returns compiles, and gives ok false.
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
// be added without code changes, because the registry and SDK treat them as
// opaque tags. Keep this list in sync with the docs when the roster
// changes.
const (
	CategorySystem         Category = "system"
	CategoryHardware       Category = "hardware"
	CategoryNetwork        Category = "network"
	CategoryCloud          Category = "cloud"
	CategoryVirtualization Category = "virtualization"
	CategorySecurity       Category = "security"
	CategorySoftware       Category = "software"
	CategoryUsers          Category = "users"
	CategoryLinux          Category = "linux"
	CategoryMisc           Category = "misc"
)

// Category is the group a collector belongs to. The constants above are
// the set: a collector returns one of them.
type Category string

// Categories are the categories of the roster above.
func Categories() []Category {
	return []Category{
		CategorySystem, CategoryHardware, CategoryNetwork, CategoryCloud,
		CategoryVirtualization, CategorySecurity, CategorySoftware, CategoryUsers,
		CategoryLinux, CategoryMisc,
	}
}

// Collector is the interface every fact collector must implement.
type Collector interface {
	// Name returns the unique identifier (e.g., "platform").
	Name() string

	// Category returns the group this collector belongs to: "system",
	// "hardware", "cloud", "network", etc. Used by the SDK's
	// WithCategory option and the CLI's --category flag to enable
	// sets of related collectors at once.
	Category() Category

	// DefaultEnabled reports whether this collector should run when no
	// explicit enable/disable flags are provided. Most collectors return
	// true; heavy or privileged collectors (ssh host keys, full package
	// inventory, full service list) return false and require the caller
	// to opt in via --collector.<name>.
	DefaultEnabled() bool

	// Dependencies returns names of collectors that must run before this one.
	Dependencies() []string

	// Collect gathers facts and returns a typed struct, nil when the host
	// does not have them, or ErrUnsupported on a platform with nothing to
	// read. prior contains the typed outputs of collectors that finished
	// before this one. It holds each dependency declared in Dependencies
	// that gave a fact, and may hold other upstream collectors. When a
	// dependency fails with no fact, Collect is not called, and the
	// collector's error wraps ErrDependencyFailed. Access typed values via
	// GetDep[T].
	Collect(
		ctx context.Context,
		prior PriorResults,
	) (any, error)
}
