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

// Package languages reports installed programming language runtimes on
// the host. Each language field carries the version string when that
// runtime is on PATH, or nil when it is absent. The set of detected
// runtimes mirrors the Ohai languages plugin.
package languages

import (
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the detected runtime versions. A nil pointer for a
// language means the runtime was not found on PATH.
type Info struct {
	Go     *string `json:"go,omitempty"`
	Python *string `json:"python,omitempty"`
	Ruby   *string `json:"ruby,omitempty"`
	Node   *string `json:"node,omitempty"`
	Java   *string `json:"java,omitempty"`
	Perl   *string `json:"perl,omitempty"`
}

// Collector is the public interface every languages variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields shared by every OS variant.
type base struct{}

// Name returns "languages".
func (base) Name() string { return "languages" }

// Category returns "software".
func (base) Category() string { return collector.CategorySoftware }

// DefaultEnabled returns false — runtime detection shells out multiple times.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the languages collector variant for the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// collectLanguages probes each well-known language runtime and returns
// an Info struct. It is shared between Linux and Darwin — version flags
// are identical on both platforms.
func collectLanguages(
	ctx context.Context,
	exec executor.Executor,
) *Info {
	info := &Info{}
	if exec == nil {
		return info
	}
	probe := func(
		cmd string,
		args []string,
		parser func(string) *string,
	) *string {
		out, err := exec.Execute(ctx, cmd, args...)
		if err != nil {
			return nil
		}
		return parser(string(out))
	}

	info.Go = probe("go", []string{"version"}, parseGoVersion)
	info.Python = probe("python3", []string{"--version"}, parsePythonVersion)
	info.Ruby = probe("ruby", []string{"--version"}, parseRubyVersion)
	info.Node = probe("node", []string{"--version"}, parseNodeVersion)
	info.Java = probe("java", []string{"-version"}, parseJavaVersion)
	info.Perl = probe("perl", []string{"--version"}, parsePerlVersion)
	return info
}

// strPtr returns a pointer to s.
func strPtr(
	s string,
) *string {
	return &s
}

// parseGoVersion extracts the version from `go version go1.21.0 ...`.
func parseGoVersion(
	out string,
) *string {
	// e.g. "go version go1.21.0 linux/amd64"
	out = strings.TrimSpace(out)
	parts := strings.Fields(out)
	for _, p := range parts {
		if strings.HasPrefix(p, "go") && len(p) > 2 {
			return strPtr(strings.TrimPrefix(p, "go"))
		}
	}
	return strPtr(out)
}

// parsePythonVersion extracts the version from `Python 3.11.0`.
func parsePythonVersion(
	out string,
) *string {
	out = strings.TrimSpace(out)
	parts := strings.Fields(out)
	if len(parts) >= 2 {
		return strPtr(parts[1])
	}
	return strPtr(out)
}

// parseRubyVersion extracts the version from `ruby 3.2.0 (...)`.
func parseRubyVersion(
	out string,
) *string {
	out = strings.TrimSpace(out)
	parts := strings.Fields(out)
	if len(parts) >= 2 {
		return strPtr(parts[1])
	}
	return strPtr(out)
}

// parseNodeVersion extracts the version from `v20.1.0`.
func parseNodeVersion(
	out string,
) *string {
	out = strings.TrimSpace(out)
	return strPtr(strings.TrimPrefix(out, "v"))
}

// parseJavaVersion extracts the version from the first line of
// `java -version` output. Java writes to stderr; the combined output
// format is "openjdk version \"21.0.1\" ...".
func parseJavaVersion(
	out string,
) *string {
	out = strings.TrimSpace(out)
	// First line only — remaining lines are JVM details.
	line := strings.SplitN(out, "\n", 2)[0]
	// Extract the quoted version: java version "21.0.1"
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start >= 0 && end > start {
		return strPtr(line[start+1 : end])
	}
	return strPtr(line)
}

// parsePerlVersion extracts the version from `This is perl 5, version 36 ...`
// or the `v5.36.0` form.
func parsePerlVersion(
	out string,
) *string {
	out = strings.TrimSpace(out)
	// `perl --version` first line: "This is perl 5, version 36, subversion 0 (v5.36.0)"
	// Extract the "(v…)" portion if present.
	start := strings.Index(out, "(v")
	end := strings.Index(out, ")")
	if start >= 0 && end > start {
		return strPtr(out[start+1 : end])
	}
	// Fallback: first line.
	return strPtr(strings.SplitN(out, "\n", 2)[0])
}
