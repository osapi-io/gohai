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
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect runs the cascade and returns Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	cascadeLinux(ctx, l.FS, l.Exec, info)
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
	info *Info,
) {
	// 0. systemd-detect-virt fast-path (not from Ohai but cheap and
	//    authoritative on systemd hosts).
	detectViaSystemd(ctx, exec, info)

	// 1. docker / 2. podman / 5. nova as host binaries.
	if execBinaryOnPath(ctx, exec, "docker") {
		addSystem(info, "docker", "host", false)
	}
	if execBinaryOnPath(ctx, exec, "podman") {
		addSystem(info, "podman", "host", false)
	}
	if execBinaryOnPath(ctx, exec, "nova") {
		addSystem(info, "openstack", "host", false)
	}

	// 3. Xen.
	if fileExists(fs, "/proc/xen") {
		addSystem(info, "xen", "guest", false)
		if fileContains(fs, "/proc/xen/capabilities", "control_d") {
			addSystem(info, "xen", "host", true)
		}
	}

	// 4. VirtualBox via /proc/modules.
	if b, err := fs.ReadFile("/proc/modules"); err == nil {
		text := string(b)
		if containsLineWithPrefix(text, "vboxdrv") {
			addSystem(info, "vbox", "host", false)
		} else if containsLineWithPrefix(text, "vboxguest") {
			addSystem(info, "vbox", "guest", false)
		}
	}

	// 6/7. KVM.
	if b, err := fs.ReadFile("/proc/cpuinfo"); err == nil {
		text := string(b)
		if strings.Contains(text, "QEMU Virtual CPU") ||
			strings.Contains(text, "Common KVM processor") {
			addSystem(info, "kvm", "guest", false)
		}
		if fileExists(fs, "/sys/devices/virtual/misc/kvm") {
			role := "host"
			if strings.Contains(text, " hypervisor") || strings.Contains(text, "\thypervisor") {
				role = "guest"
			}
			addSystem(info, "kvm", role, false)
		}
	}

	// 8. DMI match.
	detectViaDMI(fs, info)

	// 9. OpenVZ.
	if fileExists(fs, "/proc/bc/0") {
		addSystem(info, "openvz", "host", false)
	} else if fileExists(fs, "/proc/vz") {
		addSystem(info, "openvz", "guest", false)
	}

	// 10. Hyper-V KVP. When we see the pool file, extract the host
	// name between "HostName" and "HostingSystemEditionId" — matches
	// Ohai's linux/virtualization.rb extraction. Keeps Raw printable
	// bytes only and lowercases, same as Ohai's behaviour.
	if b, err := fs.ReadFile("/var/lib/hyperv/.kvp_pool_3"); err == nil {
		addSystem(info, "hyperv", "guest", false)
		info.HypervisorHost = parseHypervKVPHostName(b)
	}

	// 11. linux-vserver.
	if b, err := fs.ReadFile("/proc/self/status"); err == nil {
		text := string(b)
		switch {
		case strings.Contains(text, "s_context: 0") || strings.Contains(text, "VxID: 0"):
			addSystem(info, "linux-vserver", "host", false)
		case strings.Contains(text, "s_context:") || strings.Contains(text, "VxID:"):
			addSystem(info, "linux-vserver", "guest", false)
		}
	}

	// 12. cgroup / environ container detection.
	detectViaCgroup(fs, info)

	// 13. .dockerenv / .dockerinit override.
	if fileExists(fs, "/.dockerenv") || fileExists(fs, "/.dockerinit") {
		addSystem(info, "docker", "guest", true)
	}

	// 14. LXD.
	if fileExists(fs, "/dev/lxd/sock") {
		addSystem(info, "lxd", "guest", false)
	}
	if fileExists(fs, "/var/lib/lxd/devlxd") || fileExists(fs, "/var/snap/lxd/common/lxd/devlxd") {
		addSystem(info, "lxd", "host", false)
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
		addSystem(info, v, "guest", false)
	}
}

// detectViaDMI reads sysfs DMI fields and matches against known
// hypervisor strings. Mirrors Ohai's guest_from_dmi_data table.
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
	product := dmi("/sys/class/dmi/id/product_name")
	vendor := dmi("/sys/class/dmi/id/sys_vendor")
	bios := dmi("/sys/class/dmi/id/bios_vendor")

	all := strings.ToLower(product + "|" + vendor + "|" + bios)
	switch {
	case strings.Contains(all, "vmware"):
		addSystem(info, "vmware", "guest", false)
	case strings.Contains(all, "microsoft") && strings.Contains(all, "virtual"):
		addSystem(info, "hyperv", "guest", false)
	case strings.Contains(all, "parallels"):
		addSystem(info, "parallels", "guest", false)
	case strings.Contains(all, "xen"):
		addSystem(info, "xen", "guest", false)
	case strings.Contains(all, "qemu") || strings.Contains(all, "kvm"):
		addSystem(info, "kvm", "guest", false)
	}
}

// cgroupContainerRE matches Docker / LXC / containerd cgroup paths.
var cgroupContainerRE = regexp.MustCompile(`(?m)^\d+:[^:]+:/(docker|lxc|containerd)/`)

// detectViaCgroup parses /proc/self/cgroup and /proc/1/environ for
// container hints. Mirrors Ohai's cascade in the linux plugin.
func detectViaCgroup(
	fs avfs.VFS,
	info *Info,
) {
	if b, err := fs.ReadFile("/proc/self/cgroup"); err == nil {
		if m := cgroupContainerRE.FindStringSubmatch(string(b)); m != nil {
			name := m[1]
			if name == "containerd" {
				name = "docker"
			}
			addSystem(info, name, "guest", false)
		}
	}
	if b, err := fs.ReadFile("/proc/1/environ"); err == nil {
		text := string(b)
		switch {
		case strings.Contains(text, "container=lxc"):
			addSystem(info, "lxc", "guest", false)
		case strings.Contains(text, "container=systemd-nspawn"):
			addSystem(info, "nspawn", "guest", false)
		case strings.Contains(text, "container=podman"):
			addSystem(info, "podman", "guest", false)
		}
	}
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
		if c >= 0x20 && c < 0x7f {
			out = append(out, c)
		}
	}
	return strings.ToLower(string(out))
}
