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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects network facts on Linux: each interface of
// /sys/class/net, with its index, MTU, MAC and flags; its addresses from
// `ip -o addr show`; its counters from /proc/net/dev; its speed, duplex
// and driver from sysfs; and the neighbours from `ip neigh show`. It
// also:
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
	Exec source.Runner

	// ethtoolBudget bounds all of the collector's ethtool runs together. A
	// zero budget is defaultEthtoolBudget.
	ethtoolBudget time.Duration
}

// defaultEthtoolBudget bounds all of a collection's ethtool runs. Each run
// also has the source's own command timeout, but six runs for each NIC
// one after another could otherwise cost minutes on a host with a stuck
// driver.
const defaultEthtoolBudget = 60 * time.Second

// NewLinux returns a Linux variant wired to the real OS filesystem
// and the production Executor.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect returns network Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	ifs, err := l.interfaces(ctx)
	if err != nil {
		return nil, err
	}
	applyEncapsulation(l.FS, ifs)
	applyKind(l.FS, ifs)
	ifs = applyOpenVZAliasMerge(l.FS, ifs)
	applyNICStats(l.FS, ifs)
	info := &Info{Interfaces: ifs}
	if l.Exec != nil {
		applyRoutes(ctx, l.Exec, info)
		budget := l.ethtoolBudget
		if budget == 0 {
			budget = defaultEthtoolBudget
		}
		ectx, cancel := context.WithTimeout(ctx, budget)
		applyEthtoolDriverInfo(ectx, l.FS, l.Exec, info)
		applyEthtoolTuning(ectx, l.FS, l.Exec, info)
		cancel()
		applyIPLinkExtras(ctx, l.Exec, info)
		if out, err := l.Exec.Execute(ctx, "ip", "neigh", "show"); err == nil {
			info.Neighbours = parseIPNeigh(out)
		}
	}
	return info, nil
}

// ethtoolTarget reports whether ethtool runs for the interface: an
// Ethernet interface that a device backs. A virtual interface, such as a
// veth, a bridge or a bond, has no /sys/class/net/<iface>/device. A
// container host has hundreds of veths, and ethtool would run six times
// for each one (osapi-io/gohai#356).
func ethtoolTarget(
	fs avfs.VFS,
	iface Interface,
) bool {
	if iface.Encapsulation != "Ethernet" {
		return false
	}
	_, err := fs.Stat("/sys/class/net/" + iface.Name + "/device")
	return err == nil
}

// driverInfoTarget reports whether `ethtool -i` runs for the interface: an
// Ethernet interface that a device backs, or a bond or a bridge, whose
// driver ("bonding", "bridge") Ohai reports too. A veth is neither, so a
// container host's veths cost nothing.
func driverInfoTarget(
	fs avfs.VFS,
	iface Interface,
) bool {
	if ethtoolTarget(fs, iface) {
		return true
	}
	if iface.Encapsulation != "Ethernet" {
		return false
	}
	for _, kind := range []string{"bonding", "bridge"} {
		if _, err := fs.Stat("/sys/class/net/" + iface.Name + "/" + kind); err == nil {
			return true
		}
	}
	return false
}

// applyEthtoolDriverInfo invokes `ethtool -i <iface>` per Ethernet
// interface that driverInfoTarget names, and attaches the parsed driver info
// under `Interface.Ethtool.DriverInfo`. Mirrors Ohai's
// ethernet_driver_info, but skips virtual interfaces (see
// ethtoolTarget), as well as loopback, tunnels and the like. ethtool failures —
// binary missing, unsupported by the driver — silently leave Ethtool
// nil for that interface; non-Ethernet hosts get no Ethtool data at
// all, matching Ohai's behaviour.
func applyEthtoolDriverInfo(
	ctx context.Context,
	fs avfs.VFS,
	exec source.Runner,
	info *Info,
) {
	for i := range info.Interfaces {
		if spent(ctx) {
			return
		}
		if !driverInfoTarget(fs, info.Interfaces[i]) {
			continue
		}
		out, err := exec.Execute(ctx, "ethtool", "-i", info.Interfaces[i].Name)
		if err != nil {
			continue
		}
		di := parseEthtoolDriverInfo(out)
		if len(di) == 0 {
			continue
		}
		if info.Interfaces[i].Ethtool == nil {
			info.Interfaces[i].Ethtool = &EthtoolInfo{}
		}
		info.Interfaces[i].Ethtool.DriverInfo = di
	}
}

// applyIPLinkExtras runs `ip -d link` once and merges per-interface
// VLAN, tunnel, and XDP annotations onto the matching Interface
// entries. Mirrors the relevant subset of Ohai's link_statistics:
// the three signals that neither sysfs nor the ethtool family covers.
//
// `ip` failures (binary missing, permission denied) silently leave
// the three fields nil for every interface — same behaviour as the
// existing applyRoutes path.
func applyIPLinkExtras(
	ctx context.Context,
	exec source.Runner,
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
// interface that a device backs (see ethtoolTarget), and attaches each parsed result to the corresponding
// EthtoolInfo field. Mirrors Ohai's ethernet_{ring,channel,coalesce,
// offload,pause}_parameters block. Each subcommand is independent —
// failure of one doesn't suppress the others — and any per-call
// error or empty parse silently skips that field.
func applyEthtoolTuning(
	ctx context.Context,
	fs avfs.VFS,
	exec source.Runner,
	info *Info,
) {
	for i := range info.Interfaces {
		if spent(ctx) {
			return
		}
		if !ethtoolTarget(fs, info.Interfaces[i]) {
			continue
		}
		name := info.Interfaces[i].Name
		ensureEthtool := func() *EthtoolInfo {
			if info.Interfaces[i].Ethtool == nil {
				info.Interfaces[i].Ethtool = &EthtoolInfo{}
			}
			return info.Interfaces[i].Ethtool
		}
		if out, err := exec.Execute(ctx, "ethtool", "-g", name); err == nil {
			if v := parseEthtoolSectionedInts(out, "Ring parameters for"); v != nil {
				ensureEthtool().RingParams = v
			}
		}
		if out, err := exec.Execute(ctx, "ethtool", "-l", name); err == nil {
			if v := parseEthtoolSectionedInts(out, "Channel parameters for"); v != nil {
				ensureEthtool().ChannelParams = v
			}
		}
		if out, err := exec.Execute(ctx, "ethtool", "-c", name); err == nil {
			if v := parseEthtoolCoalesceParams(out); v != nil {
				ensureEthtool().CoalesceParams = v
			}
		}
		if out, err := exec.Execute(ctx, "ethtool", "-k", name); err == nil {
			if v := parseEthtoolOffloadParams(out); v != nil {
				ensureEthtool().OffloadParams = v
			}
		}
		if out, err := exec.Execute(ctx, "ethtool", "-a", name); err == nil {
			if v := parseEthtoolPauseParams(out); v != nil {
				ensureEthtool().PauseParams = v
			}
		}
	}
}

// applyNICStats fills each interface's speed and duplex, and its driver,
// from sysfs.
func applyNICStats(
	fs avfs.VFS,
	ifs []Interface,
) {
	for i := range ifs {
		st := readNIC(fs, ifs[i].Name)
		ifs[i].Speed, ifs[i].Duplex = st.Speed, st.Duplex
		ifs[i].Driver = readSysfsDriver(fs, ifs[i].Name)
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

// applyKind sets each interface's kind from sysfs: what sort of device it
// is, which its encapsulation does not tell (a bridge, a bond, a Wi-Fi card
// and a NIC are each Ethernet). A tool that writes the interface's config,
// such as netplan, puts each kind in a section of its own. The order of the
// checks matters: a bridge's uevent can also name it, but its bridge/
// directory is always there.
func applyKind(
	fs avfs.VFS,
	ifs []Interface,
) {
	for i := range ifs {
		ifs[i].Kind = kindOf(fs, ifs[i])
	}
}

// devtypeKind is the kind of a DEVTYPE in an interface's uevent.
var devtypeKind = map[string]string{
	"bridge":    "bridge",
	"bond":      "bond",
	"wlan":      "wifi",
	"vlan":      "vlan",
	"vxlan":     "vxlan",
	"wireguard": "wireguard",
}

// kindOf is the kind of one interface; empty for one that is not Ethernet
// and that nothing else names, such as PPP.
func kindOf(
	fs avfs.VFS,
	iface Interface,
) string {
	dir := "/sys/class/net/" + iface.Name
	has := func(name string) bool {
		_, err := fs.Stat(dir + "/" + name)
		return err == nil
	}
	switch {
	case iface.Encapsulation == "Loopback":
		return "loopback"
	case has("bridge"):
		return "bridge"
	case has("bonding"):
		return "bond"
	case has("wireless"), has("phy80211"):
		return "wifi"
	}
	if b, err := fs.ReadFile(dir + "/uevent"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if t, ok := strings.CutPrefix(strings.TrimSpace(line), "DEVTYPE="); ok {
				if k, ok := devtypeKind[t]; ok {
					return k
				}
			}
		}
	}
	switch {
	case has("tun_flags"):
		return "tun"
	case iface.Encapsulation != "Ethernet":
		return ""
	case has("device"):
		// A device backs it: a NIC.
		return "ethernet"
	default:
		// Ethernet with no device: a dummy, a veth or another link that
		// software makes.
		return "virtual"
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
	exec source.Runner,
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

// sysClassNet has a directory for each interface.
const sysClassNet = "/sys/class/net"

// The IFF_* bits of /sys/class/net/<iface>/flags that Go names.
var iffFlags = map[string]uint64{
	"up": 0x1, "broadcast": 0x2, "loopback": 0x8, "pointtopoint": 0x10,
	"multicast": 0x1000,
}

// interfaces is each interface of /sys/class/net, in index order, with
// its addresses from `ip -o addr show` and its counters from
// /proc/net/dev.
func (l *Linux) interfaces(
	ctx context.Context,
) ([]Interface, error) {
	if l.FS == nil {
		return nil, fmt.Errorf("network: no filesystem")
	}
	entries, err := l.FS.ReadDir(sysClassNet)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sysClassNet, err)
	}
	read := func(name, file string) string {
		b, _ := l.FS.ReadFile(sysClassNet + "/" + name + "/" + file)
		return strings.TrimSpace(string(b))
	}
	addrs := map[string][]Address{}
	if l.Exec != nil {
		if out, err := l.Exec.Execute(ctx, "ip", "-o", "addr", "show"); err == nil {
			addrs = parseIPAddr(out)
		}
	}
	counters := readNetDev(l.FS)
	ifs := []Interface{}
	for _, e := range entries {
		name := e.Name()
		item := Interface{Name: name, Addresses: addrs[name], Counters: counters[name]}
		item.Number, _ = strconv.Atoi(read(name, "ifindex"))
		item.MTU, _ = strconv.Atoi(read(name, "mtu"))
		if mac := read(name, "address"); strings.Trim(mac, "0:") != "" {
			item.MAC = mac
		}
		bits, _ := strconv.ParseUint(read(name, "flags"), 0, 64)
		set := map[string]bool{}
		for f, bit := range iffFlags {
			set[f] = bits&bit != 0
		}
		item.Flags = orderFlags(set)
		item.State = stateFromFlags(item.Flags)
		ifs = append(ifs, item)
	}
	sort.SliceStable(ifs, func(i, j int) bool { return ifs[i].Number < ifs[j].Number })

	return ifs, nil
}

// parseIPAddr is the addresses of `ip -o addr show`, by interface:
// "2: eth0    inet 10.0.0.5/24 brd 10.0.0.255 scope global eth0".
func parseIPAddr(
	out []byte,
) map[string][]Address {
	addrs := map[string][]Address{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || (f[2] != "inet" && f[2] != "inet6") {
			continue
		}
		if a, ok := parseAddress(withPrefix(f[3])); ok {
			addrs[f[1]] = append(addrs[f[1]], a)
		}
	}

	return addrs
}

// readNetDev is the counters of /proc/net/dev, by interface: received
// bytes, packets, errors and drops, then the same sent.
func readNetDev(
	fs avfs.VFS,
) map[string]*Counters {
	counters := map[string]*Counters{}
	b, err := fs.ReadFile("/proc/net/dev")
	if err != nil {
		return counters
	}
	for _, line := range strings.Split(string(b), "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 12 {
			continue
		}
		n := func(i int) uint64 {
			v, _ := strconv.ParseUint(f[i], 10, 64)
			return v
		}
		counters[strings.TrimSpace(name)] = &Counters{
			BytesRecv: n(0), PacketsRecv: n(1), Errin: n(2), Dropin: n(3),
			BytesSent: n(8), PacketsSent: n(9), Errout: n(10), Dropout: n(11),
		}
	}

	return counters
}

// spent reports whether the ethtool budget in ctx is used up. A timer
// context sets Err only when its timer fires on another goroutine, so a
// deadline that has passed can still have a nil Err: the deadline decides.
// time.Now here reads no fact of the host; it measures the budget.
func spent(
	ctx context.Context,
) bool {
	if ctx.Err() != nil {
		return true
	}
	deadline, ok := ctx.Deadline()

	return ok && !time.Now().Before(deadline)
}
