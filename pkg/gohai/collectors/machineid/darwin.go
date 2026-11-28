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
	"fmt"
	"regexp"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// platformUUIDRE is the IOPlatformUUID line of ioreg's output.
var platformUUIDRE = regexp.MustCompile(`"IOPlatformUUID" = "([^"]+)"`)

// Darwin reads the machine ID on macOS: IOPlatformUUID, the stable
// hardware identifier Apple gives for this, from ioreg.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's ioreg.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns the machine ID. With no Exec it gives no fact, as the
// ID comes from ioreg.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	out, err := d.Exec.Execute(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		return nil, fmt.Errorf("ioreg IOPlatformExpertDevice: %w", err)
	}
	m := platformUUIDRE.FindSubmatch(out)
	if m == nil {
		return &Info{}, nil
	}

	return &Info{ID: string(m[1]), Source: "IOPlatformUUID"}, nil
}
