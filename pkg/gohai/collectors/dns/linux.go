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

package dns

import (
	"context"
	"regexp"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux reads /etc/resolv.conf through FS, and, when systemd-resolved
// runs, each link's servers and domains from `resolvectl dns` and
// `resolvectl domain`.
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant that reads the host.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// resolvectlLinkRE is a link line of resolvectl: "Link 2 (eth0): 10.0.0.1".
var resolvectlLinkRE = regexp.MustCompile(`^Link \d+ \(([^)]+)\):(.*)$`)

// Collect returns the dns Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	if b, err := l.FS.ReadFile(resolvConf); err == nil {
		parseResolvConf(info, b)
	}
	if l.Exec == nil {
		return info, nil
	}
	for _, kind := range []string{"dns", "domain"} {
		out, err := l.Exec.Execute(ctx, "resolvectl", kind)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			m := resolvectlLinkRE.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			values := strings.Fields(m[2])
			if kind == "dns" {
				for i, v := range values {
					values[i], _, _ = strings.Cut(v, "%")
				}
				setResolver(info, m[1], values, nil)
			} else {
				setResolver(info, m[1], nil, values)
			}
		}
	}

	return info, nil
}
