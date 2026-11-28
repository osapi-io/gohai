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
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin reads /etc/resolv.conf through FS, and each interface's
// resolver from `scutil --dns`.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewDarwin returns a Darwin variant that reads the host.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

var (
	scutilNameserverRE = pattern.New(`^nameserver\[\d+\]\s*:\s*(\S+)`)
	scutilSearchRE     = pattern.New(`^search domain\[\d+\]\s*:\s*(\S+)`)
	scutilIfIndexRE    = pattern.New(`^if_index\s*:\s*\d+\s*\(([^)]+)\)`)
	scutilDomainRE     = pattern.New(`^domain\s*:`)
)

// Collect returns the dns Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info, err := readResolvConf(d.FS)
	if err != nil {
		return nil, err
	}
	if d.Exec == nil {
		return info, nil
	}
	out, err := d.Exec.Execute(ctx, "scutil", "--dns")
	if collector.Absent(err) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read scutil --dns: %w", err)
	}
	// Each "resolver #N" block has its servers, its search domains and,
	// when it belongs to one, its interface. The first block of an
	// interface wins: scutil prints the scoped resolvers again after.
	//
	// Global is the default resolver, from which macOS writes resolv.conf:
	// the first block of the unscoped "DNS configuration" section that has
	// servers, no "domain" (which makes it a resolver for that domain only)
	// and is not Supplemental (which a VPN adds for its own names).
	var servers, domains []string
	scoped, forDomain, supplemental := false, false, false
	flush := func(iface string) {
		if _, seen := info.Interfaces[iface]; iface != "" && !seen {
			setResolver(info, iface, servers, domains)
		}
		if !scoped && !forDomain && !supplemental && info.Global == nil && len(servers) > 0 {
			info.Global = &Resolver{Nameservers: servers, Domains: domains}
		}
		servers, domains = nil, nil
		forDomain, supplemental = false, false
	}
	iface := ""
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "DNS configuration (for scoped queries)"):
			flush(iface)
			iface = ""
			scoped = true
		case strings.HasPrefix(line, "resolver #"):
			flush(iface)
			iface = ""
		case scutilNameserverRE.MatchString(line):
			servers = append(servers, scutilNameserverRE.FindStringSubmatch(line)[1])
		case scutilSearchRE.MatchString(line):
			domains = append(domains, scutilSearchRE.FindStringSubmatch(line)[1])
		case scutilIfIndexRE.MatchString(line):
			iface = scutilIfIndexRE.FindStringSubmatch(line)[1]
		case scutilDomainRE.MatchString(line):
			forDomain = true
		case strings.HasPrefix(line, "flags") && strings.Contains(line, "Supplemental"):
			supplemental = true
		}
	}
	flush(iface)

	return info, nil
}
