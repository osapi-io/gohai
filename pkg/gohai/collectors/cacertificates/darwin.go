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

package cacertificates

import (
	"context"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/osapi-io/gohai/internal/collector"
)

// The keychains of macOS that hold CAs: the roots that Apple ships, and the
// System keychain, where an administrator adds one.
var keychains = []Dir{
	{Path: "/System/Library/Keychains/SystemRootCertificates.keychain", Source: SourceSystem},
	{Path: "/Library/Keychains/System.keychain", Source: SourceLocal},
}

// Darwin collects the CAs of a Mac from its keychains with `security`.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's commands.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect lists each certificate of each keychain as PEM. A keychain that
// `security` cannot read has no CAs. The System keychain can hold
// certificates that are not CAs, such as a device's own; a certificate
// that is not a CA is left out.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{Certificates: []Certificate{}}
	for _, k := range keychains {
		out, err := d.Exec.Execute(ctx, "security", "find-certificate", "-a", "-p", k.Path)
		if err != nil {
			continue
		}
		for _, c := range certs(out) {
			if !c.IsCA {
				continue
			}
			e := Certificate{Name: nameOf(c), Path: k.Path, Source: k.Source, Enabled: true}
			info.Certificates = append(info.Certificates, describe(e, c))
		}
	}

	return info, nil
}
