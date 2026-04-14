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

package dmi

import (
	"github.com/jaypipes/ghw/pkg/baseboard"
	"github.com/jaypipes/ghw/pkg/bios"
	"github.com/jaypipes/ghw/pkg/chassis"
	"github.com/jaypipes/ghw/pkg/product"
)

// SetBIOSFn swaps the bios.New seam for tests and returns a restore
// func the caller defers. Same pattern as osapi's test helpers.
func SetBIOSFn(
	fn func(...any) (*bios.Info, error),
) func() {
	orig := biosFn
	biosFn = fn
	return func() { biosFn = orig }
}

// SetBaseboardFn swaps the baseboard.New seam for tests.
func SetBaseboardFn(
	fn func(...any) (*baseboard.Info, error),
) func() {
	orig := baseboardFn
	baseboardFn = fn
	return func() { baseboardFn = orig }
}

// SetChassisFn swaps the chassis.New seam for tests.
func SetChassisFn(
	fn func(...any) (*chassis.Info, error),
) func() {
	orig := chassisFn
	chassisFn = fn
	return func() { chassisFn = orig }
}

// SetProductFn swaps the product.New seam for tests.
func SetProductFn(
	fn func(...any) (*product.Info, error),
) func() {
	orig := productFn
	productFn = fn
	return func() { productFn = orig }
}
