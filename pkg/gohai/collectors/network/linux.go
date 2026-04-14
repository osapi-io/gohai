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

package network

import (
	"context"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects network facts on Linux. gopsutil enumerates
// interfaces and I/O counters; we additionally:
//
//   - Read /sys/class/net/<iface>/type to derive the canonical
//     encapsulation name (Ethernet / Loopback / PPP / SLIP / IPIP /
//     6to4) — Ohai's linux_encaps_lookup table.
//   - Run `ip -o -f inet route show table main` and `ip -o -f inet6
//     route show table main` to populate the routing table, default
//     interface, and default gateway (v4 + v6).
//   - On OpenVZ guests (/proc/vz present, /proc/bc/0 absent) merge
//     `venet0:N` alias addresses under the primary venet0 interface
//     so consumers querying interfaces[venet0] find the IPs.
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the real OS filesystem
// and the production Executor.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect returns network Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	ifs, err := readInterfaces(ctx)
	if err != nil {
		return nil, err
	}
	if l.FS != nil {
		applyEncapsulation(l.FS, ifs)
		ifs = applyOpenVZAliasMerge(l.FS, ifs)
	}
	applyNICStats(l.FS, ifs)
	info := &Info{Interfaces: ifs}
	if l.Exec != nil {
		applyRoutes(ctx, l.Exec, info)
	}
	if entries, err := neighListFn(); err == nil {
		info.Neighbours = entries
	}
	return info, nil
}

// applyNICStats merges per-interface link-layer detail. Speed +
// Duplex come from ghw via the nicFn seam; Driver comes from a
// sysfs read of `/sys/class/net/<iface>/device/driver` symlink
// target via the avfs.VFS so tests don't need ghw.
func applyNICStats(
	fs avfs.VFS,
	ifs []Interface,
) {
	stats, err := nicFn()
	if err != nil {
		stats = nil
	}
	for i := range ifs {
		if s, ok := stats[ifs[i].Name]; ok {
			ifs[i].Speed = s.Speed
			ifs[i].Duplex = s.Duplex
		}
		if fs != nil {
			ifs[i].Driver = readSysfsDriver(fs, ifs[i].Name)
		}
	}
}

// readSysfsDriver resolves /sys/class/net/<iface>/device/driver as a
// symlink and returns the basename. Empty when the symlink can't be
// read (virtual / loopback interfaces have no driver).
func readSysfsDriver(
	fs avfs.VFS,
	name string,
) string {
	target, err := fs.Readlink("/sys/class/net/" + name + "/device/driver")
	if err != nil {
		return ""
	}
	// target looks like "../../../../bus/pci/drivers/e1000e"; basename.
	if i := strings.LastIndex(target, "/"); i >= 0 {
		return target[i+1:]
	}
	return target
}

// applyEncapsulation reads /sys/class/net/<iface>/type for each
// interface and assigns the canonical encapsulation string.
func applyEncapsulation(
	fs avfs.VFS,
	ifs []Interface,
) {
	for i := range ifs {
		b, err := fs.ReadFile("/sys/class/net/" + ifs[i].Name + "/type")
		if err != nil {
			continue
		}
		t, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			continue
		}
		if name, ok := arphrdEncapsulation[t]; ok {
			ifs[i].Encapsulation = name
		}
	}
}

// applyOpenVZAliasMerge collapses `venet0:N` aliases under their
// primary interface when running inside an OpenVZ guest. Detection:
// /proc/vz exists AND /proc/bc/0 does not.
func applyOpenVZAliasMerge(
	fs avfs.VFS,
	ifs []Interface,
) []Interface {
	if !openVZGuest(fs) {
		return ifs
	}
	byName := map[string]int{}
	for i, iface := range ifs {
		byName[iface.Name] = i
	}
	out := ifs[:0]
	for _, iface := range ifs {
		if base, alias := isOpenVZAlias(iface.Name); alias {
			if idx, ok := byName[base]; ok {
				ifs[idx].Addresses = append(ifs[idx].Addresses, iface.Addresses...)
				continue
			}
		}
		out = append(out, iface)
	}
	return out
}

// openVZGuest returns true when the host is an OpenVZ guest:
// /proc/vz present, /proc/bc/0 absent.
func openVZGuest(
	fs avfs.VFS,
) bool {
	if _, err := fs.Stat("/proc/vz"); err != nil {
		return false
	}
	if _, err := fs.Stat("/proc/bc/0"); err == nil {
		return false
	}
	return true
}

// applyRoutes runs `ip route show` for v4 and v6, parses the
// output, populates Info.Routes + per-interface Routes + the
// top-level default_* fields.
func applyRoutes(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	for _, fam := range []struct{ flag, family string }{
		{"-4", "inet"},
		{"-6", "inet6"},
	} {
		out, err := exec.Execute(ctx, "ip", "-o", fam.flag, "route", "show", "table", "main")
		if err != nil {
			continue
		}
		for _, logical := range joinContinuationLines(string(out)) {
			for _, r := range expandRouteLine(logical, fam.family) {
				info.Routes = append(info.Routes, r)
				if !isDefaultDestination(r.Destination) {
					continue
				}
				// First default-route hit wins (matches Ohai). On
				// multipath defaults the first nexthop becomes the
				// reported default; later nexthops still appear in
				// Routes.
				if fam.family == "inet" && info.DefaultInterface == "" {
					info.DefaultInterface = r.Interface
					info.DefaultGateway = r.Gateway
				}
				if fam.family == "inet6" && info.DefaultInet6Interface == "" {
					info.DefaultInet6Interface = r.Interface
					info.DefaultInet6Gateway = r.Gateway
				}
			}
		}
	}
	resolveRouteInterfacesBySource(info)
	for _, r := range info.Routes {
		attachToInterface(info, r)
	}
}

// joinContinuationLines folds `\`-continued lines into a single
// logical line. `ip route` emits multipath routes as one prefix line
// followed by indented `nexthop` lines, each ending with `\` for
// continuation. We rejoin them so expandRouteLine sees the whole
// route as one string. Empty / whitespace-only lines are dropped.
func joinContinuationLines(
	out string,
) []string {
	var logical []string
	var pending strings.Builder
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		cont := strings.HasSuffix(trimmed, "\\")
		if cont {
			pending.WriteString(strings.TrimSuffix(trimmed, "\\"))
			pending.WriteByte(' ')
			continue
		}
		pending.WriteString(trimmed)
		if s := strings.TrimSpace(pending.String()); s != "" {
			logical = append(logical, s)
		}
		pending.Reset()
	}
	return logical
}

// expandRouteLine handles the multipath case. A multipath route looks
// like (after `\` continuation join):
//
//	default proto static \
//	    nexthop via 10.0.0.1 dev eth0 weight 1 \
//	    nexthop via 10.0.0.2 dev eth1 weight 1
//
// We split on the literal `nexthop` token and emit one Route per
// nexthop, copying the destination/proto/scope/metric attributes from
// the prefix. Single-route lines (no `nexthop`) yield exactly one
// Route from the line as-is (after backslash → space).
func expandRouteLine(
	line, family string,
) []Route {
	if !strings.Contains(line, "nexthop") {
		return []Route{parseIPRouteLine(strings.TrimSpace(line), family)}
	}
	idx := strings.Index(line, "nexthop")
	prefix := strings.TrimSpace(line[:idx])
	prefixRoute := parseIPRouteLine(prefix, family)
	var out []Route
	for _, hop := range strings.Split(line[idx:], "nexthop") {
		hop = strings.TrimSpace(hop)
		if hop == "" {
			continue
		}
		// Re-use parseIPRouteLine by synthesizing a route line —
		// `<destination> nexthop tokens...`. Destination from the
		// prefix; remaining attrs (gateway/dev/weight/...) come from
		// the nexthop block.
		synth := prefixRoute.Destination + " " + hop
		hopRoute := parseIPRouteLine(synth, family)
		// Carry forward proto / scope / metric from the prefix when
		// the nexthop block doesn't restate them.
		if hopRoute.Proto == "" {
			hopRoute.Proto = prefixRoute.Proto
		}
		if hopRoute.Scope == "" {
			hopRoute.Scope = prefixRoute.Scope
		}
		if hopRoute.Metric == 0 {
			hopRoute.Metric = prefixRoute.Metric
		}
		out = append(out, hopRoute)
	}
	return out
}

// resolveRouteInterfacesBySource fills in Route.Interface for routes
// where `dev` was missing but `src` is present, by matching `src`
// against addresses owned by enumerated interfaces. Mirrors Ohai's
// fallback in linux/network.rb.
func resolveRouteInterfacesBySource(
	info *Info,
) {
	if len(info.Routes) == 0 {
		return
	}
	addrToIface := map[string]string{}
	for _, iface := range info.Interfaces {
		for _, a := range iface.Addresses {
			addrToIface[a.Addr] = iface.Name
		}
	}
	for i := range info.Routes {
		r := &info.Routes[i]
		if r.Interface != "" || r.Source == "" {
			continue
		}
		if name, ok := addrToIface[r.Source]; ok {
			r.Interface = name
		}
	}
}

// parseIPRouteLine parses one `ip -o route` line into a Route. The
// format is `<destination> [via <gateway>] [dev <iface>] [proto X]
// [metric N] [src ADDR] [scope S] ...`. Tokens beyond what we know
// are ignored. Caller filters empty lines, so fields[0] always
// exists.
func parseIPRouteLine(
	line, family string,
) Route {
	fields := strings.Fields(line)
	r := Route{Destination: fields[0], Family: family}
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "via":
			if i+1 < len(fields) {
				r.Gateway = fields[i+1]
				i++
			}
		case "dev":
			if i+1 < len(fields) {
				r.Interface = fields[i+1]
				i++
			}
		case "src":
			if i+1 < len(fields) {
				r.Source = fields[i+1]
				i++
			}
		case "scope":
			if i+1 < len(fields) {
				r.Scope = fields[i+1]
				i++
			}
		case "proto":
			if i+1 < len(fields) {
				r.Proto = fields[i+1]
				i++
			}
		case "metric":
			if i+1 < len(fields) {
				if m, err := strconv.Atoi(fields[i+1]); err == nil {
					r.Metric = m
				}
				i++
			}
		}
	}
	return r
}

// attachToInterface appends the route to the matching interface's
// Routes slice. Silent when the interface isn't enumerated.
func attachToInterface(
	info *Info,
	r Route,
) {
	if r.Interface == "" {
		return
	}
	for i := range info.Interfaces {
		if info.Interfaces[i].Name == r.Interface {
			info.Interfaces[i].Routes = append(info.Interfaces[i].Routes, r)
			return
		}
	}
}

// isDefaultDestination reports whether the route destination is the
// kernel's idiomatic "default route" form.
func isDefaultDestination(
	dest string,
) bool {
	switch dest {
	case "default", "0.0.0.0/0", "::/0":
		return true
	}
	return false
}
