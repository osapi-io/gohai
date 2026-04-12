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

package kernel

import "context"

// Darwin collects kernel facts on macOS using syscall.Uname. Modules
// on darwin would require parsing `kextstat` output — Apple has
// deprecated kexts since Big Sur (system extensions replaced them),
// so we skip the module map on darwin for now.
type Darwin struct {
	base

	UnameFn func() (name, release, version, machine string, err error)
}

// NewDarwin returns a Darwin variant wired to the shared defaultUname
// helper.
func NewDarwin() *Darwin {
	return &Darwin{UnameFn: defaultUname}
}

// Collect returns kernel Info. No modules map on darwin.
func (d *Darwin) Collect(_ context.Context) (any, error) {
	name, release, version, machine, err := d.UnameFn()
	if err != nil {
		return nil, err
	}
	return &Info{
		Name:    name,
		Release: release,
		Version: version,
		Machine: machine,
	}, nil
}
