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

package vmware

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects VMware Tools data on macOS. macOS guests running under
// VMware Fusion have vmware-toolbox-cmd at /usr/bin/vmware-toolbox-cmd
// (same path as Linux). Detection probes the binary with `-v`; if the
// command is absent or fails we're not a VMware guest and return nil.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect gathers VMware Tools statistics on macOS. It returns no fact
// when the host is not a VMware guest or VMware Tools is absent. When a
// call to vmware-toolbox-cmd does not finish, it returns the values read
// before that call with the error.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	guest, err := probeToolbox(ctx, d.Exec)
	if err != nil || !guest {
		return nil, err
	}

	return collectVMwareTools(ctx, d.Exec)
}
