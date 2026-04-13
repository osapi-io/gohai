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

package filesystem

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
)

// ListMounts exposes the private listMounts bridge to the external
// filesystem_test package.
var ListMounts = listMounts

// SetPartitionsFn swaps the private gopsutil disk.PartitionsWithContext
// call backing listMounts. Returns a restore func the caller must
// defer.
func SetPartitionsFn(
	fn func(context.Context, bool) ([]disk.PartitionStat, error),
) (restore func()) {
	orig := partitionsFn
	partitionsFn = fn
	return func() { partitionsFn = orig }
}

// SetUsageFn swaps the private gopsutil disk.UsageWithContext call
// backing listMounts. Returns a restore func the caller must defer.
func SetUsageFn(
	fn func(context.Context, string) (*disk.UsageStat, error),
) (restore func()) {
	orig := usageFn
	usageFn = fn
	return func() { usageFn = orig }
}

// SetListMountsFn swaps the per-collector listMounts seam. Returns a
// restore func the caller must defer.
func SetListMountsFn(
	fn func(context.Context) ([]Mount, error),
) (restore func()) {
	orig := listMountsFn
	listMountsFn = fn
	return func() { listMountsFn = orig }
}
