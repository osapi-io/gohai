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

// Package docker reports Docker server version, containers, and images
// on the host. If Docker is not on PATH or the daemon is unreachable,
// Collect returns nil gracefully — mirrors Ohai's docker.rb stance of
// not failing the run when Docker is absent.
package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
)

// Container holds metadata for a single Docker container.
type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
}

// Image holds metadata for a single Docker image.
type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
}

// Info holds Docker server version and resource lists.
type Info struct {
	Version    string      `json:"version"`
	Containers []Container `json:"containers"`
	Images     []Image     `json:"images"`
}

// Collector is the public interface every docker variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields shared by every OS variant.
type base struct{}

// Name returns "docker".
func (base) Name() string { return "docker" }

// Category returns "software".
func (base) Category() string { return collector.CategorySoftware }

// DefaultEnabled returns false — docker may not be present on the host.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the docker collector variant for the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// dockerJSONContainer is the JSON shape produced by
// `docker ps -a --format '{{json .}}'`.
type dockerJSONContainer struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

// dockerJSONImage is the JSON shape produced by
// `docker images --format '{{json .}}'`.
type dockerJSONImage struct {
	ID         string `json:"ID"`
	Repository string `json:"Repository"`
	Tag        string `json:"Tag"`
	Size       string `json:"Size"`
}

// collectDocker queries Docker version, containers, and images.
// Returns nil when Docker is absent (version call fails).
func collectDocker(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	if exec == nil {
		return nil, nil
	}
	versionOut, err := exec.Execute(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		// Docker not on PATH or daemon unreachable — soft-miss.
		return nil, nil
	}
	version := strings.TrimSpace(string(versionOut))

	containers, err := collectContainers(ctx, exec)
	if err != nil {
		containers = []Container{}
	}
	images, err := collectImages(ctx, exec)
	if err != nil {
		images = []Image{}
	}
	return &Info{
		Version:    version,
		Containers: containers,
		Images:     images,
	}, nil
}

// collectContainers runs `docker ps -a --format '{{json .}}'` and
// parses the NDJSON output (one JSON object per line).
func collectContainers(
	ctx context.Context,
	exec executor.Executor,
) ([]Container, error) {
	out, err := exec.Execute(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var result []Container
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var c dockerJSONContainer
		if jsonErr := json.Unmarshal([]byte(line), &c); jsonErr != nil {
			continue
		}
		result = append(result, Container{
			ID:     c.ID,
			Name:   strings.TrimPrefix(c.Names, "/"),
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
		})
	}
	return result, nil
}

// collectImages runs `docker images --format '{{json .}}'` and parses
// the NDJSON output.
func collectImages(
	ctx context.Context,
	exec executor.Executor,
) ([]Image, error) {
	out, err := exec.Execute(ctx, "docker", "images", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var result []Image
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var img dockerJSONImage
		if jsonErr := json.Unmarshal([]byte(line), &img); jsonErr != nil {
			continue
		}
		result = append(result, Image(img))
	}
	return result, nil
}
