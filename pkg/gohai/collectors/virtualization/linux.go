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
	"slices"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux runs the full Ohai-parity detection cascade on Linux. Every
// positive hit contributes to Info.Systems; the last positive hit
// also sets Info.System / Info.Role for backward compat with
// single-layer consumers.
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// the production Executor.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect runs the cascade and returns Info.
func (l *Linux) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	info := &Info{}
	cascadeLinux(ctx, l.FS, l.Exec, prior, info)
	// The cascade skips each probe that fails, since most hosts lack most
	// probes. A cancelled collection stops every probe, so its empty
	// result would look like a host with no virtualization: it is an
	// error instead.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return info, nil
}

// cascadeLinux walks Ohai's linux/virtualization.rb detection order
// and populates info. The order matters: later detections take
// precedence for the primary System/Role. Each step is independent
// and silent on absence.
func cascadeLinux(
	ctx context.Context,
	fs avfs.VFS,
	exec source.Runner,
	prior collector.PriorResults,
	info *Info,
) {
	// 0. systemd-detect-virt fast-path (not from Ohai but cheap and
	//    authoritative on systemd hosts).
	detectViaSystemd(ctx, exec, info)

	// 1. docker / 2. podman / 5. nova as hosts. A CLI on the PATH is not a
	//    runtime: a container image can hold the docker CLI. docker is a
	//    host when its daemon's socket is there. podman has no daemon, so
	//    it is a host when its binary is there outside a podman container.
	if execBinaryOnPath(ctx, exec, "docker") && dockerSocket(fs) {
		addSystem(info, "docker", "host", false)
	}
	if execBinaryOnPath(ctx, exec, "podman") && !fileExists(fs, "/run/.containerenv") {
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
			strings.Contains(text, "Common KVM processor") ||
			strings.Contains(text, "Common 32-bit KVM processor") {
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
	// 7b. KVM via cpu prior (lscpu's hypervisor_vendor / virtualization_type).
	// Covers nested VMs where /sys/devices/virtual/misc/kvm isn't present
	// but lscpu still reports a hypervisor. Matches Ohai's
	// cpu[:hypervisor_vendor] == "KVM" + cpu[:virtualization_type] check.
	if cpuInfo, ok := collector.GetDep[*cpu.Info](prior, "cpu"); ok && cpuInfo != nil {
		if strings.EqualFold(cpuInfo.HypervisorVendor, "KVM") &&
			(cpuInfo.VirtualizationType == "full" || cpuInfo.VirtualizationType == "para") {
			addSystem(info, "kvm", "guest", false)
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
	// name between "HostName" and "HostingSystemEditionId", as
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

	// 12a. LXC host: lxc-version or lxc-start on PATH AND cgroup root
	// paths are all "/" (host-side cgroup namespace, not a container).
	// Only fires when nothing else set System, as Ohai's OHAI-573
	// guard to prevent false positives on lxc hosts that also look
	// container-like via other signals.
	if info.System == "" && cgroupRootsAllSlash(fs) {
		if execBinaryOnPath(ctx, exec, "lxc-version") || execBinaryOnPath(ctx, exec, "lxc-start") {
			addSystem(info, "lxc", "host", false)
		}
	}

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
	exec source.Runner,
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

// detectViaDMI reads sysfs DMI fields and matches against Ohai's
// full guest_from_dmi_data table. Manufacturer (sys_vendor) is
// checked first, then product_name, as in mixin/dmi_decode.rb.
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

	// Manufacturer-keyed signals (Ohai checks these first).
	switch {
	case strings.Contains(vendor, "openstack"):
		addSystem(info, "openstack", "guest", false)
		return
	case strings.Contains(vendor, "xen"):
		addSystem(info, "xen", "guest", false)
		return
	case strings.Contains(vendor, "vmware"):
		addSystem(info, "vmware", "guest", false)
		return
	case strings.Contains(vendor, "microsoft") && strings.Contains(product, "virtual machine"):
		addSystem(info, "hyperv", "guest", false)
		return
	case strings.Contains(vendor, "amazon ec2"):
		addSystem(info, "amazonec2", "guest", false)
		return
	case strings.Contains(vendor, "qemu"):
		addSystem(info, "kvm", "guest", false)
		return
	case strings.Contains(vendor, "veertu"):
		addSystem(info, "veertu", "guest", false)
		return
	case strings.Contains(vendor, "parallels"):
		addSystem(info, "parallels", "guest", false)
		return
	}
	// Product-keyed signals (fallback when vendor didn't match).
	switch {
	case strings.Contains(product, "virtualbox"):
		addSystem(info, "vbox", "guest", false)
	case strings.Contains(product, "openstack"):
		addSystem(info, "openstack", "guest", false)
	case strings.Contains(product, "kvm") || strings.Contains(product, "rhev"):
		addSystem(info, "kvm", "guest", false)
	case strings.Contains(product, "bhyve"):
		addSystem(info, "bhyve", "guest", false)
	}
}

// cgroupContainerRE matches Docker / LXC / containerd cgroup paths
// that sit directly under the cgroup root (classic docker/lxc layout).
var cgroupContainerRE = regexp.MustCompile(`(?m)^\d+:[^:]+:/(docker|lxc|containerd)/`)

// cgroupNestedContainerRE matches systemd-managed and docker-ce layouts
// where the runtime appears as a named cgroup under a parent slice:
// `/system.slice/docker-<hash>.scope`, `/docker-ce/docker/<hash>`,
// `/kubepods/.../docker-<hash>.scope`, etc. Mirrors Ohai's second regex
// in linux/virtualization.rb.
var cgroupNestedContainerRE = regexp.MustCompile(`(?m)^\d+:[^:]*:/[^/]+/(docker|lxc)-?`)

// cgroupScopeRE matches the cgroup v2 path of a container that a
// Kubernetes node or systemd runs, such as
// `0::/kubepods.slice/.../cri-containerd-<hash>.scope` or
// `.../crio-<hash>.scope`. The other two regexes need the runtime as the
// first or second part of the path, and these paths have it last.
var cgroupScopeRE = regexp.MustCompile(
	`(?m)^\d+:[^:]*:/\S*/(cri-containerd|crio|docker)-[0-9a-f]+\.scope$`,
)

// scopeRuntimes is the system that each runtime of cgroupScopeRE records.
// containerd is docker, as the direct regex records it.
var scopeRuntimes = map[string]string{
	"cri-containerd": "docker",
	"crio":           "crio",
	"docker":         "docker",
}

// detectViaCgroup parses /proc/self/cgroup, /proc/1/environ,
// /proc/self/environ and /run/.containerenv for container hints. The
// first two mirror Ohai's cascade in the linux plugin.
func detectViaCgroup(
	fs avfs.VFS,
	info *Info,
) {
	if b, err := fs.ReadFile("/proc/self/cgroup"); err == nil {
		text := string(b)
		matched := false
		if m := cgroupContainerRE.FindStringSubmatch(text); m != nil {
			name := m[1]
			if name == "containerd" {
				name = "docker"
			}
			addSystem(info, name, "guest", false)
			matched = true
		}
		if !matched {
			if m := cgroupNestedContainerRE.FindStringSubmatch(text); m != nil {
				addSystem(info, m[1], "guest", false)
				matched = true
			}
		}
		if !matched {
			if m := cgroupScopeRE.FindStringSubmatch(text); m != nil {
				addSystem(info, scopeRuntimes[m[1]], "guest", false)
			}
		}
	}
	// /proc/1/environ is readable only by root. The collector's own
	// environment, /proc/self/environ, is readable by its user, and a
	// process in a container gets the container's variables.
	for _, path := range []string{"/proc/1/environ", "/proc/self/environ"} {
		b, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		vars := strings.Split(string(b), "\x00")
		switch {
		case hasVar(vars, "container=lxc"):
			addSystem(info, "lxc", "guest", false)
		case hasVar(vars, "container=systemd-nspawn"):
			addSystem(info, "nspawn", "guest", false)
		case hasVar(vars, "container=podman"):
			addSystem(info, "podman", "guest", false)
		}
		// A pod gets KUBERNETES_SERVICE_HOST. Kubernetes is a layer, not
		// the runtime, so it is the primary system only when nothing else
		// is.
		if hasVar(vars, "KUBERNETES_SERVICE_HOST=") {
			if info.System == "" {
				addSystem(info, "kubernetes", "guest", false)
			} else {
				info.Systems["kubernetes"] = "guest"
			}
		}
	}
	// podman writes /run/.containerenv in each container that it runs.
	if fileExists(fs, "/run/.containerenv") {
		addSystem(info, "podman", "guest", false)
	}
}

// hasVar reports whether one of vars starts with prefix, such as
// container=lxc for container=lxc-libvirt.
func hasVar(
	vars []string,
	prefix string,
) bool {
	return slices.ContainsFunc(vars, func(v string) bool { return strings.HasPrefix(v, prefix) })
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
		fields := strings.SplitN(line, ":", 3)
		if len(fields) < 3 {
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

// dockerSocket reports whether a docker daemon's socket is there: the
// system daemon's, or a rootless daemon's under /run/user/<uid>.
func dockerSocket(
	fs avfs.VFS,
) bool {
	if fileExists(fs, "/var/run/docker.sock") || fileExists(fs, "/run/docker.sock") {
		return true
	}
	users, err := fs.ReadDir("/run/user")
	if err != nil {
		return false
	}
	for _, u := range users {
		if fileExists(fs, "/run/user/"+u.Name()+"/docker.sock") {
			return true
		}
	}

	return false
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

// pathProbe is the shell script that execBinaryOnPath runs. `command` is
// a shell builtin: Linux has no /usr/bin/command, so it runs through sh,
// with the name as "$1" and never as part of the script.
const pathProbe = `command -v "$1"`

// execBinaryOnPath runs `command -v <name>` in sh through the executor;
// an exit code of 0 means the binary is on PATH.
func execBinaryOnPath(
	ctx context.Context,
	exec source.Runner,
	name string,
) bool {
	if exec == nil {
		return false
	}
	_, err := exec.Execute(ctx, "sh", "-c", pathProbe, "sh", name)
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
