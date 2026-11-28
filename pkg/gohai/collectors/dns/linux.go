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
	"fmt"
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

// resolvedRuntimeDir is the runtime directory of systemd-resolved,
// which exists only while the service runs.
const resolvedRuntimeDir = "/run/systemd/resolve"

// resolvectlLinkRE is a link line of resolvectl: "Link 2 (eth0): 10.0.0.1".
var resolvectlLinkRE = regexp.MustCompile(`^Link \d+ \(([^)]+)\):(.*)$`)

// resolvectlGlobalRE is resolvectl's line for the resolver of no link:
// "Global: 1.1.1.1".
var resolvectlGlobalRE = regexp.MustCompile(`^Global:(.*)$`)

// Collect returns the dns Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info, err := readResolvConf(l.FS)
	if err != nil {
		return nil, err
	}
	if l.Exec == nil {
		return info, nil
	}
	// resolvectl answers only while systemd-resolved runs, and the
	// service's runtime directory exists only then. Without it, the host
	// has no per-link resolvers, which is not a failed read.
	if _, err := l.FS.Stat(resolvedRuntimeDir); err != nil {
		if collector.Absent(err) {
			return info, nil
		}

		return nil, fmt.Errorf("stat %s: %w", resolvedRuntimeDir, err)
	}
	for _, kind := range []string{"dns", "domain"} {
		out, err := l.Exec.Execute(ctx, "resolvectl", kind)
		if collector.Absent(err) {
			return info, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read resolvectl %s: %w", kind, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if m := resolvectlGlobalRE.FindStringSubmatch(line); m != nil {
				values := strings.Fields(m[1])
				if len(values) == 0 {
					continue
				}
				if info.Global == nil {
					info.Global = &Resolver{}
				}
				if kind == "dns" {
					info.Global.Nameservers = append(
						info.Global.Nameservers,
						serverAddrs(values)...)
				} else {
					info.Global.Domains = append(info.Global.Domains, values...)
				}
				continue
			}
			m := resolvectlLinkRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			values := strings.Fields(m[2])
			if kind == "dns" {
				setResolver(info, m[1], serverAddrs(values), nil)
			} else {
				setResolver(info, m[1], nil, values)
			}
		}
	}

	return info, nil
}

// serverAddrs gives the address of each resolvectl server: without its
// "%eth0" zone, its "#name" TLS server name or its port, as in
// "1.1.1.1:853#cloudflare-dns.com" or "[2606:4700::1111]:853".
func serverAddrs(
	values []string,
) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v, _, _ = strings.Cut(v, "#")
		if strings.HasPrefix(v, "[") {
			if end := strings.Index(v, "]"); end > 0 {
				v = v[1:end]
			}
		} else if strings.Count(v, ":") == 1 {
			v, _, _ = strings.Cut(v, ":")
		}
		v, _, _ = strings.Cut(v, "%")
		out = append(out, v)
	}

	return out
}
