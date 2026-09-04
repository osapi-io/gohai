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

// Package hostnamectl parses `hostnamectl` output on Linux to report
// static hostname, chassis type, deployment environment, kernel info,
// and hardware model. Matches Ohai's linux/hostnamectl plugin.
package hostnamectl

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// nonASCIIRe strips visual icons produced by newer systemd versions
// (Unicode decorators in hostnamectl output). Mirrors Ohai's
// `/[^\p{ASCII}]/u` regex.
var nonASCIIRe = regexp.MustCompile(`[^\x00-\x7F]+`)

// Info holds hostnamectl fields.
type Info struct {
	StaticHostname            string `json:"static_hostname,omitempty"`
	IconName                  string `json:"icon_name,omitempty"`
	Chassis                   string `json:"chassis,omitempty"`
	Deployment                string `json:"deployment,omitempty"`
	Location                  string `json:"location,omitempty"`
	KernelName                string `json:"kernel_name,omitempty"`
	KernelRelease             string `json:"kernel_release,omitempty"`
	OperatingSystemPrettyName string `json:"operating_system_pretty_name,omitempty"`
	OperatingSystemCPEName    string `json:"operating_system_cpe_name,omitempty"`
	Virtualization            string `json:"virtualization,omitempty"`
	HardwareVendor            string `json:"hardware_vendor,omitempty"`
	HardwareModel             string `json:"hardware_model,omitempty"`
	FirmwareVersion           string `json:"firmware_version,omitempty"`
}

// Collector is the public interface every hostnamectl variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "hostnamectl" }
func (base) Category() string       { return collector.CategoryLinux }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the hostnamectl variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// parseHostnamectl parses `hostnamectl` key: value output into an Info
// struct. Mirrors Ohai's linux/hostnamectl.rb: splits on ": ", strips
// non-ASCII from values, and maps known keys to typed fields.
//
// Ohai converts keys to snake_case by downcasing and replacing spaces with
// underscores — we do the same mapping from the canonical English labels.
func parseHostnamectl(
	output string,
) *Info {
	info := &Info{}

	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		key, val, ok := hostnamectlField(sc.Text())
		if ok {
			setHostnamectlField(info, key, val)
		}
	}

	return info
}

// hostnamectlField splits one "Key: value" line, normalising the key the
// way Ohai does — downcase, spaces to underscores — and stripping the
// emoji decorators systemd v250 and later print.
func hostnamectlField(
	line string,
) (key string, val string, ok bool) {
	rawKey, rawVal, found := strings.Cut(line, ": ")
	if !found {
		return "", "", false
	}

	v := strings.TrimSpace(nonASCIIRe.ReplaceAllString(strings.TrimSpace(rawVal), ""))

	// Stripping the decorators can leave runs of spaces behind.
	for strings.Contains(v, "  ") {
		v = strings.ReplaceAll(v, "  ", " ")
	}

	k := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(rawKey), " ", "_"))

	return k, strings.TrimSpace(v), true
}

// setHostnamectlField files one normalised key.
func setHostnamectlField(
	info *Info,
	key string,
	val string,
) {
	if key == "kernel" {
		// "Kernel: Linux 5.15.0-91-generic" is a name and a release.
		name, release, _ := strings.Cut(val, " ")
		info.KernelName = name
		info.KernelRelease = release

		return
	}

	dst := map[string]*string{
		"static_hostname":              &info.StaticHostname,
		"icon_name":                    &info.IconName,
		"chassis":                      &info.Chassis,
		"deployment":                   &info.Deployment,
		"location":                     &info.Location,
		"operating_system_pretty_name": &info.OperatingSystemPrettyName,
		"operating_system":             &info.OperatingSystemPrettyName,
		"operating_system_cpe_name":    &info.OperatingSystemCPEName,
		"cpe_os_name":                  &info.OperatingSystemCPEName,
		"virtualization":               &info.Virtualization,
		"hardware_vendor":              &info.HardwareVendor,
		"hardware_model":               &info.HardwareModel,
		"firmware_version":             &info.FirmwareVersion,
	}[key]

	if dst != nil {
		*dst = val
	}
}
