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
	"strings"
	"sync"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin runs the macOS detection cascade. Mirrors Ohai's
// darwin/virtualization.rb: PATH probes for hypervisor binaries
// (Docker/VBoxManage/prlctl), VMware Fusion app presence,
// `sysctl kern.hv_vmm_present` for QEMU/Virtualization.framework,
// `ioreg -l` grep for Parallels, and `system_profiler
// SPHardwareDataType` for VirtualBox/VMware/Apple-VM Boot ROM /
// Model Identifier strings.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the real OS filesystem
// and the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect runs the cascade and returns Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	cascadeDarwin(ctx, d.FS, d.Exec, info)
	return info, nil
}

// cascadeDarwin walks Ohai's darwin/virtualization.rb detection order.
// The three slow execs (sysctl, ioreg, system_profiler) run concurrently —
// together they used to dominate wall time; fanning them out drops
// virtualization's cost to max(exec) instead of sum(exec). Results are
// applied in the same order as the serial version so System/Role
// precedence is unchanged.
func cascadeDarwin(
	ctx context.Context,
	fs avfs.VFS,
	exec executor.Executor,
	info *Info,
) {
	detectDarwinHostTools(ctx, fs, exec, info) // 1-4.

	if exec == nil {
		return
	}

	sysctlOut, ioregOut, spOut := probeDarwinGuest(ctx, exec)

	if strings.TrimSpace(string(sysctlOut)) == "1" {
		addSystem(info, "qemu", roleGuest)
	}

	if strings.Contains(string(ioregOut), "pci1ab8,4000") {
		addSystem(info, "parallels", roleGuest)
	}

	detectDarwinFromProfile(string(spOut), info)
}

// detectDarwinHostTools treats a hypervisor's own tooling as evidence
// this machine is the host. Steps 1 through 4.
func detectDarwinHostTools(
	ctx context.Context,
	fs avfs.VFS,
	exec executor.Executor,
	info *Info,
) {
	for _, b := range []struct{ bin, system string }{
		{systemDocker, systemDocker},
		{"VBoxManage", "vbox"},
		{"prlctl", "parallels"},
	} {
		if execBinaryOnPath(ctx, exec, b.bin) {
			addSystem(info, b.system, roleHost)
		}
	}

	if fileExists(fs, "/Applications/VMware Fusion.app") {
		addSystem(info, "vmware", roleHost)
	}
}

// probeDarwinGuest runs the three slow probes at once. `ioreg -n
// pci1ab8,4000` targets the Parallels PCI node directly rather than
// dumping the whole I/O registry — the same answer as Ohai's
// `ioreg -l | grep`, about seven times faster.
func probeDarwinGuest(
	ctx context.Context,
	exec executor.Executor,
) (sysctlOut, ioregOut, spOut []byte) {
	var wg sync.WaitGroup

	wg.Go(func() {
		out, err := exec.Execute(ctx, "sysctl", "-n", "kern.hv_vmm_present")
		if err == nil {
			sysctlOut = out
		}
	})

	wg.Go(func() {
		out, err := exec.Execute(ctx, "ioreg", "-n", "pci1ab8,4000")
		if err == nil {
			ioregOut = out
		}
	})

	wg.Go(func() {
		out, err := exec.Execute(ctx, "system_profiler", "SPHardwareDataType")
		if err == nil {
			spOut = out
		}
	})

	wg.Wait()

	return sysctlOut, ioregOut, spOut
}

// detectDarwinFromProfile reads the boot ROM and model identifier, which
// name the hypervisor a guest is running under.
func detectDarwinFromProfile(
	text string,
	info *Info,
) {
	if text == "" {
		return
	}

	switch {
	case bootROMContains(text, "VirtualBox"):
		addSystem(info, "vbox", roleGuest)
	case bootROMContains(text, "VMW"):
		addSystem(info, "vmware", roleGuest)
	default:
		// The boot ROM names no hypervisor.
	}

	if modelIDContains(text, "VirtualMac") {
		addSystem(info, "apple", roleGuest)
	}
}

// bootROMContains scans system_profiler output for `Boot ROM Version`
// containing the substring.
func bootROMContains(
	text, substr string,
) bool {
	for _, line := range strings.Split(text, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "Boot ROM Version:") && strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

// modelIDContains scans system_profiler output for `Model Identifier`
// containing the substring.
func modelIDContains(
	text, substr string,
) bool {
	for _, line := range strings.Split(text, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "Model Identifier:") && strings.Contains(l, substr) {
			return true
		}
	}
	return false
}
