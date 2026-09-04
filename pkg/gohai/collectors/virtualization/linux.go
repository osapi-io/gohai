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

package virtualization

import (
	"context"
	"regexp"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

// Linux runs the full Ohai-parity detection cascade on Linux. Every
// positive hit contributes to Info.Systems; the last positive hit
// also sets Info.System / Info.Role for backward compat with
// single-layer consumers.
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// the production Executor.
// The printable range of ASCII, which is all this keeps.
const (
	asciiSpace = 0x20
	asciiDEL   = 0x7f
)

func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect runs the cascade and returns Info.
func (l *Linux) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	info := &Info{}
	cascadeLinux(ctx, l.FS, l.Exec, prior, info)
	return info, nil
}

// cascadeLinux walks Ohai's linux/virtualization.rb detection order
// and populates info. The order matters — later detections take
// precedence for the primary System/Role. Each step is independent
// and silent on absence.
func cascadeLinux(
	ctx context.Context,
	fs avfs.VFS,
	exec executor.Executor,
	prior collector.PriorResults,
	info *Info,
) {
	// Ohai's order, and its numbering. Each step contributes what it
	// finds; the last authoritative signal wins.
	detectViaSystemd(ctx, exec, info) // 0.
	detectHostBinaries(ctx, exec, info)
	detectXen(fs, info)        // 3.
	detectVirtualBox(fs, info) // 4.
	detectKVM(fs, prior, info) // 6, 7.
	detectViaDMI(fs, info)     // 8.
	detectOpenVZ(fs, info)     // 9.
	detectHyperV(fs, info)     // 10.
	detectVServer(fs, info)    // 11.
	detectViaCgroup(fs, info)  // 12.
	detectLXCHost(ctx, fs, exec, info)
	detectDockerEnv(fs, info) // 13.
	detectLXD(fs, info)       // 14.
}

// detectHostBinaries treats a management binary on PATH as evidence the
// host runs that platform. Steps 1, 2 and 5.
func detectHostBinaries(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	for _, b := range []struct{ bin, system string }{
		{systemDocker, systemDocker},
		{"podman", "podman"},
		{"nova", systemOpenStack},
	} {
		if execBinaryOnPath(ctx, exec, b.bin) {
			addSystem(info, b.system, roleHost)
		}
	}
}

// detectXen reads /proc/xen. control_d in its capabilities means this is
// dom0 — the host — rather than a guest.
func detectXen(
	fs avfs.VFS,
	info *Info,
) {
	if !fileExists(fs, "/proc/xen") {
		return
	}

	addSystem(info, systemXen, roleGuest)

	if fileContains(fs, "/proc/xen/capabilities", "control_d") {
		overrideSystem(info, systemXen, roleHost)
	}
}

// detectVirtualBox tells the host driver from the guest additions by
// which module is loaded.
func detectVirtualBox(
	fs avfs.VFS,
	info *Info,
) {
	b, err := fs.ReadFile("/proc/modules")
	if err != nil {
		return
	}

	text := string(b)

	switch {
	case containsLineWithPrefix(text, "vboxdrv"):
		addSystem(info, "vbox", roleHost)
	case containsLineWithPrefix(text, "vboxguest"):
		addSystem(info, "vbox", roleGuest)
	default:
		// Neither module is loaded.
	}
}

// detectKVM looks at the CPU model, the kvm device, and what lscpu
// reported. Steps 6, 7 and 7b.
func detectKVM(
	fs avfs.VFS,
	prior collector.PriorResults,
	info *Info,
) {
	if b, err := fs.ReadFile("/proc/cpuinfo"); err == nil {
		detectKVMFromCPUInfo(fs, string(b), info)
	}

	// Covers a nested VM, where /sys/devices/virtual/misc/kvm is absent
	// but lscpu still reports a hypervisor. Matches Ohai's
	// cpu[:hypervisor_vendor] == "KVM" check.
	cpuInfo, ok := collector.GetDep[*cpu.Info](prior, "cpu")
	if !ok || cpuInfo == nil {
		return
	}

	if strings.EqualFold(cpuInfo.HypervisorVendor, "KVM") &&
		(cpuInfo.VirtualizationType == "full" ||
			cpuInfo.VirtualizationType == "para") {
		addSystem(info, systemKVM, roleGuest)
	}
}

// detectKVMFromCPUInfo reads the guest signature out of the CPU model,
// then decides host or guest from the hypervisor flag.
func detectKVMFromCPUInfo(
	fs avfs.VFS,
	text string,
	info *Info,
) {
	if strings.Contains(text, "QEMU Virtual CPU") ||
		strings.Contains(text, "Common KVM processor") ||
		strings.Contains(text, "Common 32-bit KVM processor") {
		addSystem(info, systemKVM, roleGuest)
	}

	if !fileExists(fs, "/sys/devices/virtual/misc/kvm") {
		return
	}

	role := roleHost
	if strings.Contains(text, " hypervisor") ||
		strings.Contains(text, "\thypervisor") {
		role = roleGuest
	}

	addSystem(info, systemKVM, role)
}

// detectOpenVZ distinguishes the host's /proc/bc/0 from a guest's
// /proc/vz.
func detectOpenVZ(
	fs avfs.VFS,
	info *Info,
) {
	switch {
	case fileExists(fs, "/proc/bc/0"):
		addSystem(info, "openvz", roleHost)
	case fileExists(fs, "/proc/vz"):
		addSystem(info, "openvz", roleGuest)
	default:
		// Neither path is present.
	}
}

// detectHyperV reads the KVP pool file, which names the hosting system.
func detectHyperV(
	fs avfs.VFS,
	info *Info,
) {
	b, err := fs.ReadFile("/var/lib/hyperv/.kvp_pool_3")
	if err != nil {
		return
	}

	addSystem(info, "hyperv", roleGuest)
	info.HypervisorHost = parseHypervKVPHostName(b)
}

// detectVServer reads the vserver context out of /proc/self/status.
// Context zero is the host.
func detectVServer(
	fs avfs.VFS,
	info *Info,
) {
	b, err := fs.ReadFile("/proc/self/status")
	if err != nil {
		return
	}

	text := string(b)

	switch {
	case strings.Contains(text, "s_context: 0") ||
		strings.Contains(text, "VxID: 0"):
		addSystem(info, "linux-vserver", roleHost)
	case strings.Contains(text, "s_context:") ||
		strings.Contains(text, "VxID:"):
		addSystem(info, "linux-vserver", roleGuest)
	default:
		// Not a vserver at all.
	}
}

// detectLXCHost fires only when nothing else claimed the host, which is
// Ohai's OHAI-573 guard against an LXC host that also looks
// container-like through another signal.
func detectLXCHost(
	ctx context.Context,
	fs avfs.VFS,
	exec executor.Executor,
	info *Info,
) {
	if info.System != "" || !cgroupRootsAllSlash(fs) {
		return
	}

	if execBinaryOnPath(ctx, exec, "lxc-version") ||
		execBinaryOnPath(ctx, exec, "lxc-start") {
		addSystem(info, "lxc", roleHost)
	}
}

// detectDockerEnv is authoritative: these files exist only inside a
// container, so they override whatever an earlier step decided.
func detectDockerEnv(
	fs avfs.VFS,
	info *Info,
) {
	if fileExists(fs, "/.dockerenv") || fileExists(fs, "/.dockerinit") {
		overrideSystem(info, systemDocker, roleGuest)
	}
}

// detectLXD tells the guest socket from the host's devlxd endpoint.
func detectLXD(
	fs avfs.VFS,
	info *Info,
) {
	if fileExists(fs, "/dev/lxd/sock") {
		addSystem(info, "lxd", roleGuest)
	}

	if fileExists(fs, "/var/lib/lxd/devlxd") ||
		fileExists(fs, "/var/snap/lxd/common/lxd/devlxd") {
		addSystem(info, "lxd", roleHost)
	}
}

// detectViaSystemd asks systemd-detect-virt for both VM and container
// answers. Each non-"none" / non-empty result contributes a Systems
// entry as guest (systemd-detect-virt only reports the role of the
// caller, which is always guest when running inside virt).
func detectViaSystemd(
	ctx context.Context,
	exec executor.Executor,
	info *Info,
) {
	if exec == nil {
		return
	}
	for _, mode := range []string{"--vm", "--container"} {
		out, err := exec.Execute(ctx, "systemd-detect-virt", mode)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(out))
		if v == "" || v == "none" {
			continue
		}
		addSystem(info, v, roleGuest)
	}
}

// detectViaDMI reads sysfs DMI fields and matches against Ohai's
// full guest_from_dmi_data table. Manufacturer (sys_vendor) is
// checked first, then product_name — matches mixin/dmi_decode.rb.
// dmiSignal is one row of the DMI match table: a substring to look for,
// the platform it identifies, and — for Hyper-V, which needs both DMI
// fields — a product string that must also match.
type dmiSignal struct {
	match       string
	system      string
	alsoProduct string
}

// The DMI tables, in Ohai's order. Manufacturer is consulted first, and
// the product name only when no manufacturer matched.
var (
	dmiVendorSignals = []dmiSignal{
		{match: systemOpenStack, system: systemOpenStack},
		{match: systemXen, system: systemXen},
		{match: "vmware", system: "vmware"},
		{match: "microsoft", system: "hyperv", alsoProduct: "virtual machine"},
		{match: "amazon ec2", system: "amazonec2"},
		{match: "qemu", system: systemKVM},
		{match: "veertu", system: "veertu"},
		{match: "parallels", system: "parallels"},
	}

	dmiProductSignals = []dmiSignal{
		{match: "virtualbox", system: "vbox"},
		{match: systemOpenStack, system: systemOpenStack},
		{match: systemKVM, system: systemKVM},
		{match: "rhev", system: systemKVM},
		{match: "bhyve", system: "bhyve"},
	}
)

func detectViaDMI(
	fs avfs.VFS,
	info *Info,
) {
	dmi := func(path string) string {
		b, err := fs.ReadFile(path)
		if err != nil {
			return ""
		}

		return strings.TrimSpace(string(b))
	}

	product := strings.ToLower(dmi("/sys/class/dmi/id/product_name"))
	vendor := strings.ToLower(dmi("/sys/class/dmi/id/sys_vendor"))

	if matchDMI(info, vendor, product, dmiVendorSignals) {
		return
	}

	matchDMI(info, product, product, dmiProductSignals)
}

// matchDMI records the first signal that matches, and reports whether
// one did.
func matchDMI(
	info *Info,
	value string,
	product string,
	signals []dmiSignal,
) bool {
	for _, s := range signals {
		if !strings.Contains(value, s.match) {
			continue
		}

		if s.alsoProduct != "" && !strings.Contains(product, s.alsoProduct) {
			continue
		}

		addSystem(info, s.system, roleGuest)

		return true
	}

	return false
}

// cgroupContainerRE matches Docker / LXC / containerd cgroup paths
// that sit directly under the cgroup root (classic docker/lxc layout).
var cgroupContainerRE = regexp.MustCompile(`(?m)^\d+:[^:]+:/(docker|lxc|containerd)/`)

// cgroupNestedContainerRE matches systemd-managed and docker-ce layouts
// where the runtime appears as a named cgroup under a parent slice —
// `/system.slice/docker-<hash>.scope`, `/docker-ce/docker/<hash>`,
// `/kubepods/.../docker-<hash>.scope`, etc. Mirrors Ohai's second regex
// in linux/virtualization.rb.
var cgroupNestedContainerRE = regexp.MustCompile(`(?m)^\d+:[^:]*:/[^/]+/(docker|lxc)-?`)

// detectViaCgroup parses /proc/self/cgroup and /proc/1/environ for
// container hints. Mirrors Ohai's cascade in the linux plugin.
func detectViaCgroup(
	fs avfs.VFS,
	info *Info,
) {
	if b, err := fs.ReadFile("/proc/self/cgroup"); err == nil {
		detectFromCgroupPaths(string(b), info)
	}

	if b, err := fs.ReadFile("/proc/1/environ"); err == nil {
		detectFromPID1Environ(string(b), info)
	}
}

// detectFromCgroupPaths matches the classic layout first, then the
// systemd and docker-ce layouts where the runtime is a named cgroup
// under a parent slice.
func detectFromCgroupPaths(
	text string,
	info *Info,
) {
	if m := cgroupContainerRE.FindStringSubmatch(text); m != nil {
		name := m[1]
		// containerd is how Docker appears on a modern host.
		if name == "containerd" {
			name = systemDocker
		}

		addSystem(info, name, roleGuest)

		return
	}

	if m := cgroupNestedContainerRE.FindStringSubmatch(text); m != nil {
		addSystem(info, m[1], roleGuest)
	}
}

// detectFromPID1Environ reads the container= variable the runtime sets
// on the init process.
func detectFromPID1Environ(
	text string,
	info *Info,
) {
	switch {
	case strings.Contains(text, "container=lxc"):
		addSystem(info, "lxc", roleGuest)
	case strings.Contains(text, "container=systemd-nspawn"):
		addSystem(info, "nspawn", roleGuest)
	case strings.Contains(text, "container=podman"):
		addSystem(info, "podman", roleGuest)
	default:
		// PID 1 names no container runtime.
	}
}

// cgroupRootsAllSlash reports whether every line of /proc/self/cgroup
// has a root path of "/". Ohai uses this as the "is a real LXC host,
// not a container itself" signal. Matches Ohai's `roots.uniq == ["/"]`
// check on `/proc/self/cgroup` field 2's trailing path.
func cgroupRootsAllSlash(
	fs avfs.VFS,
) bool {
	b, err := fs.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for _, line := range lines {
		fields := strings.SplitN(line, ":", cgroupFields)
		if len(fields) < cgroupFields {
			return false
		}
		if strings.TrimSpace(fields[2]) != "/" {
			return false
		}
	}
	return true
}

// fileExists reports whether path exists on the FS.
func fileExists(
	fs avfs.VFS,
	path string,
) bool {
	_, err := fs.Stat(path)
	return err == nil
}

// fileContains reports whether path's contents include needle.
func fileContains(
	fs avfs.VFS,
	path, needle string,
) bool {
	b, err := fs.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), needle)
}

// containsLineWithPrefix reports whether any line in text starts with
// the given prefix.
func containsLineWithPrefix(
	text, prefix string,
) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// execBinaryOnPath runs `command -v <name>` through the executor; an
// exit code of 0 means the binary is on PATH. Tests stub it through
// the gomock executor; production wraps real shell behaviour.
func execBinaryOnPath(
	ctx context.Context,
	exec executor.Executor,
	name string,
) bool {
	if exec == nil {
		return false
	}
	_, err := exec.Execute(ctx, "command", "-v", name)
	return err == nil
}

// hypervKVPHostRE matches the HostName → HostingSystemEditionId span
// Ohai uses. The Hyper-V KVP pool file is a binary blob with embedded
// NULs; Ohai keeps only printable bytes from the match and lowercases.
var hypervKVPHostRE = regexp.MustCompile(`HostName([\s\S]*?)HostingSystemEditionId`)

// parseHypervKVPHostName extracts the hypervisor hostname from a
// /var/lib/hyperv/.kvp_pool_3 blob. Matches Ohai's linux/virtualization.rb
// extraction exactly: regex between `HostName` and `HostingSystemEditionId`,
// keep printable ASCII bytes only, lowercase. Returns "" when the pool
// doesn't carry the HostName key (rare but possible on non-Hyper-V
// hypervisors that create the file).
func parseHypervKVPHostName(
	b []byte,
) string {
	m := hypervKVPHostRE.FindSubmatch(b)
	if m == nil {
		return ""
	}
	var out []byte
	for _, c := range m[1] {
		if c >= asciiSpace && c < asciiDEL {
			out = append(out, c)
		}
	}
	return strings.ToLower(string(out))
}
