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

package libvirt

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Darwin is a no-op libvirt collector for macOS. KVM requires Linux; libvirt
// on macOS targets QEMU/HVF but the CLI surface and daemon behaviour differ
// substantially from the Linux case. Rather than emit partial data, we return
// nil, as Ohai does: it only implements the Linux branch.
type Darwin struct {
	base
}

// NewDarwin returns a Darwin variant.
func NewDarwin() *Darwin {
	return &Darwin{}
}

// Collect always returns nil on macOS, because libvirt does not support Darwin.
func (d *Darwin) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return nil, collector.ErrUnsupported
}
