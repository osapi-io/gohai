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

package cpu

import (
	"context"
	"fmt"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
)

var (
	cpuInfoFn   = gcpu.InfoWithContext
	cpuCountsFn = gcpu.CountsWithContext
)

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, cpuInfoFn, cpuCountsFn)
}

func collectFromGopsutil(
	ctx context.Context,
	infoFn func(context.Context) ([]gcpu.InfoStat, error),
	countsFn func(context.Context, bool) (int, error),
) (any, error) {
	infos, err := infoFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("cpu.Info: %w", err)
	}
	total, err := countsFn(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("cpu.Counts(logical): %w", err)
	}
	cores, err := countsFn(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("cpu.Counts(physical): %w", err)
	}
	out := &Info{Total: total, Cores: cores}
	if len(infos) > 0 {
		first := infos[0]
		out.ModelName = first.ModelName
		out.VendorID = first.VendorID
		out.Family = first.Family
		out.Model = first.Model
		out.Stepping = first.Stepping
		out.Mhz = first.Mhz
		out.CacheSize = first.CacheSize
		out.Flags = first.Flags
	}
	return out, nil
}
