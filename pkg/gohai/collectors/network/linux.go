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
// ethtoolBinary is the command this collector shells out to.
const ethtoolBinary = "ethtool"

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
		applyEthtoolDriverInfo(ctx, l.Exec, info)
		applyEthtoolTuning(ctx, l.Exec, info)
		applyIPLinkExtras(ctx, l.Exec, info)
	}
	if entries, err := neighListFn(); err == nil {
		info.Neighbours = entries
	}
	return info, nil
}

// applyEthtoolDriverInfo invokes `ethtool -i <iface>` per Ethernet
// interface and attaches the parsed driver info under
// `Interface.Ethtool.DriverInfo`. Mirrors Ohai's
// ethernet_driver_info: only Ethernet-encapsulated interfaces are
// queried (loopback, tunnels, etc. are skipped). ethtool failures —
// binary missing, unsupported by the driver — silently leave Ethtool
// nil for that interface; non-Ethernet hosts get no Ethtool data at
// all, matching Ohai's behaviour.
func applyEthtoolDriverInfo(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	for i := range info.Interfaces {
		iface := &info.Interfaces[i]
		if iface.Encapsulation != "Ethernet" {
			continue
		}

		di := parseEthtoolDriverInfo(ethtoolQuery(ctx, exec, iface.Name, "-i"))
		if len(di) == 0 {
			continue
		}

		ensureEthtool(iface).DriverInfo = di
	}
}

// applyIPLinkExtras runs `ip -d link` once and merges per-interface
// VLAN, tunnel, and XDP annotations onto the matching Interface
// entries. Mirrors the relevant subset of Ohai's link_statistics:
// the three signals neither gopsutil nor the ethtool family covers.
//
// `ip` failures (binary missing, permission denied) silently leave
// the three fields nil for every interface — same behaviour as the
// existing applyRoutes path.
func applyIPLinkExtras(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	out, err := exec.Execute(ctx, "ip", "-d", "link")
	if err != nil {
		return
	}
	annotations := parseIPLinkOutput(out)
	if len(annotations) == 0 {
		return
	}
	for i := range info.Interfaces {
		ann, ok := annotations[info.Interfaces[i].Name]
		if !ok {
			continue
		}
		info.Interfaces[i].VLAN = ann.VLAN
		info.Interfaces[i].TunnelInfo = ann.TunnelInfo
		info.Interfaces[i].XDP = ann.XDP
	}
}

// applyEthtoolTuning invokes the five tuning-oriented ethtool
// subcommands (ring, channel, coalesce, offload, pause) per Ethernet
// interface and attaches each parsed result to the corresponding
// EthtoolInfo field. Mirrors Ohai's ethernet_{ring,channel,coalesce,
// offload,pause}_parameters block. Each subcommand is independent —
// failure of one doesn't suppress the others — and any per-call
// error or empty parse silently skips that field.
func applyEthtoolTuning(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	for i := range info.Interfaces {
		if info.Interfaces[i].Encapsulation != "Ethernet" {
			continue
		}

		applyEthtoolTuningTo(ctx, exec, &info.Interfaces[i])
	}
}

// applyEthtoolTuningTo records whatever each ethtool query answers. An
// interface that answers none of them keeps a nil Ethtool rather than an
// empty one, which the parsers returning nil arranges.
func applyEthtoolTuningTo(
	ctx context.Context,
	exec executor.Executor,
	iface *Interface,
) {
	name := iface.Name

	if v := parseEthtoolSectionedInts(
		ethtoolQuery(ctx, exec, name, "-g"), "Ring parameters for",
	); v != nil {
		ensureEthtool(iface).RingParams = v
	}

	if v := parseEthtoolSectionedInts(
		ethtoolQuery(ctx, exec, name, "-l"), "Channel parameters for",
	); v != nil {
		ensureEthtool(iface).ChannelParams = v
	}

	if v := parseEthtoolCoalesceParams(
		ethtoolQuery(ctx, exec, name, "-c"),
	); v != nil {
		ensureEthtool(iface).CoalesceParams = v
	}

	if v := parseEthtoolOffloadParams(
		ethtoolQuery(ctx, exec, name, "-k"),
	); v != nil {
		ensureEthtool(iface).OffloadParams = v
	}

	if v := parseEthtoolPauseParams(
		ethtoolQuery(ctx, exec, name, "-a"),
	); v != nil {
		ensureEthtool(iface).PauseParams = v
	}
}

// ethtoolQuery runs one ethtool query, returning nil when it fails. The
// parsers read nil as empty output, so a caller need not check twice.
func ethtoolQuery(
	ctx context.Context,
	exec executor.Executor,
	name string,
	flag string,
) []byte {
	out, err := exec.Execute(ctx, ethtoolBinary, flag, name)
	if err != nil {
		return nil
	}

	return out
}

// ensureEthtool creates the record the first time a query answers.
func ensureEthtool(
	iface *Interface,
) *EthtoolInfo {
	if iface.Ethtool == nil {
		iface.Ethtool = &EthtoolInfo{}
	}

	return iface.Ethtool
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
		// venet0:0 and friends are aliases whose addresses belong to
		// the interface they are named after.
		if idx, merged := aliasTarget(byName, iface.Name); merged {
			ifs[idx].Addresses = append(ifs[idx].Addresses, iface.Addresses...)

			continue
		}

		out = append(out, iface)
	}

	return out
}

// aliasTarget reports the index of the interface an alias belongs to,
// and whether the name was an alias with a base that is present.
func aliasTarget(
	byName map[string]int,
	name string,
) (int, bool) {
	base, alias := isOpenVZAlias(name)
	if !alias {
		return 0, false
	}

	idx, ok := byName[base]

	return idx, ok
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
		out, err := exec.Execute(
			ctx, "ip", "-o", fam.flag, "route", "show", "table", "main",
		)
		if err != nil {
			continue
		}

		addRoutes(info, string(out), fam.family)
	}

	resolveRouteInterfacesBySource(info)

	for _, r := range info.Routes {
		attachToInterface(info, r)
	}
}

// addRoutes records one family's routes, and notes the first default
// among them.
func addRoutes(
	info *Info,
	out string,
	family string,
) {
	for _, logical := range joinContinuationLines(out) {
		for _, r := range expandRouteLine(logical, family) {
			info.Routes = append(info.Routes, r)
			noteDefaultRoute(info, r, family)
		}
	}
}

// noteDefaultRoute records the first default route of each family, which
// is what Ohai reports. On a multipath default the first nexthop wins;
// the rest are still listed under Routes.
func noteDefaultRoute(
	info *Info,
	r Route,
	family string,
) {
	if !isDefaultDestination(r.Destination) {
		return
	}

	if family == "inet" && info.DefaultInterface == "" {
		info.DefaultInterface = r.Interface
		info.DefaultGateway = r.Gateway
	}

	if family == "inet6" && info.DefaultInet6Interface == "" {
		info.DefaultInet6Interface = r.Interface
		info.DefaultInet6Gateway = r.Gateway
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
			// strings.Builder writes cannot fail.
			_, _ = pending.WriteString(strings.TrimSuffix(trimmed, "\\"))
			_ = pending.WriteByte(' ')

			continue
		}

		_, _ = pending.WriteString(trimmed)
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
	idx := strings.Index(line, "nexthop")
	if idx < 0 {
		return []Route{parseIPRouteLine(strings.TrimSpace(line), family)}
	}

	prefixRoute := parseIPRouteLine(strings.TrimSpace(line[:idx]), family)

	var out []Route

	for _, hop := range strings.Split(line[idx:], "nexthop") {
		hop = strings.TrimSpace(hop)
		if hop == "" {
			continue
		}

		out = append(out, nexthopRoute(prefixRoute, hop, family))
	}

	return out
}

// nexthopRoute builds one route of a multipath default. The destination
// comes from the prefix and the gateway and device from the nexthop
// block; proto, scope and metric carry forward when the block does not
// restate them.
func nexthopRoute(
	prefix Route,
	hop string,
	family string,
) Route {
	// parseIPRouteLine wants a whole line, so give it one.
	r := parseIPRouteLine(prefix.Destination+" "+hop, family)

	if r.Proto == "" {
		r.Proto = prefix.Proto
	}

	if r.Scope == "" {
		r.Scope = prefix.Scope
	}

	if r.Metric == 0 {
		r.Metric = prefix.Metric
	}

	return r
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

	addrToIface := interfaceByAddress(info)

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

// interfaceByAddress indexes the interfaces by the addresses they carry,
// so a route naming only its source can be attributed to one.
func interfaceByAddress(
	info *Info,
) map[string]string {
	out := map[string]string{}

	for _, iface := range info.Interfaces {
		for _, a := range iface.Addresses {
			out[a.Addr] = iface.Name
		}
	}

	return out
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

	// Most of what follows the destination is a "key value" pair, but a
	// bare flag such as onlink or linkdown sits between them, so an
	// unrecognised word advances by one rather than two.
	for i := 1; i+1 < len(fields); i++ {
		if setRouteField(&r, fields[i], fields[i+1]) {
			i++
		}
	}

	return r
}

// setRouteField files one key/value pair of an ip route line, reporting
// whether the key was one it knows — and so whether the value after it
// was consumed.
func setRouteField(
	r *Route,
	key string,
	val string,
) bool {
	if key == "metric" {
		if m, err := strconv.Atoi(val); err == nil {
			r.Metric = m
		}

		return true
	}

	dst := map[string]*string{
		"via":   &r.Gateway,
		"dev":   &r.Interface,
		"src":   &r.Source,
		"scope": &r.Scope,
		"proto": &r.Proto,
	}[key]

	if dst == nil {
		return false
	}

	*dst = val

	return true
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
