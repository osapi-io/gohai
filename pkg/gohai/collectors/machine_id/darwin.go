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

package machineid

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin resolves the machine ID on macOS. Wraps readHostID
// (gopsutil.host.Info internally, which on darwin reads
// `IOPlatformUUID` from IOKit — the correct, stable hardware
// identifier Apple intends for this purpose).
type Darwin struct {
	base
}

// NewDarwin returns a Darwin variant.
func NewDarwin() *Darwin {
	return &Darwin{}
}

// Collect returns the machine ID. gopsutil's darwin path is
// correct — no extension needed.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	id, err := readHostID(ctx)
	if err != nil {
		return nil, err
	}
	return &Info{ID: id}, nil
}
