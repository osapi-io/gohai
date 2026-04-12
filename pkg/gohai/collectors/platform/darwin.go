//go:build darwin

// Copyright (c) 2024 John Dewey

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

package platform

import (
	"context"
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/host"
)

func collect(
	ctx context.Context,
) (any, error) {
	return collectWithHost(ctx, host.InfoWithContext)
}

// collectWithHost is the testable core: it accepts a function matching
// gopsutil's host.InfoWithContext so tests can stub the system call.
func collectWithHost(
	ctx context.Context,
	fn func(context.Context) (*host.InfoStat, error),
) (any, error) {
	info, err := fn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	return &Info{
		OS:           runtime.GOOS,
		Name:         info.Platform,
		Version:      info.PlatformVersion,
		Family:       "mac_os_x",
		Architecture: info.KernelArch,
		Build:        info.KernelVersion,
	}, nil
}
