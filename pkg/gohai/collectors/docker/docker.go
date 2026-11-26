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
	"maps"
	"slices"
	"strings"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Container holds metadata for a single Docker container. ID is the full
// ID, which docker start, stop, exec and rm take. The fields after Status
// come from docker inspect; they are empty when inspect fails.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Created string `json:"created,omitempty"`
	// Health is the status of the container's health check, such as
	// healthy, or empty when it has none.
	Health   string    `json:"health,omitempty"`
	Ports    []Port    `json:"ports,omitempty"`
	Mounts   []Mount   `json:"mounts,omitempty"`
	Networks []Network `json:"networks,omitempty"`
	// EnvNames are the names of the container's environment variables. The
	// values are left out: they often hold secrets, such as a password.
	EnvNames []string `json:"env_names,omitempty"`
}

// Port is a port of a container, and the host address it is published on.
type Port struct {
	// ContainerPort is the port and protocol, such as 80/tcp.
	ContainerPort string `json:"container_port"`
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      string `json:"host_port,omitempty"`
}

// Mount is a volume or a bind mount of a container.
type Mount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only"`
}

// Network is a network a container is on, with its address there.
type Network struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
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
func (base) Category() collector.Category { return collector.CategorySoftware }

// DefaultEnabled returns false — docker may not be present on the host.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the docker collector variant for the detected host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
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

// dockerInspect is the part of one element of `docker inspect` output that
// the collector reads.
type dockerInspect struct {
	ID      string `json:"Id"`
	Created string `json:"Created"`
	State   struct {
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	Config struct {
		Env []string `json:"Env"`
	} `json:"Config"`
	Mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
			Gateway   string `json:"Gateway"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
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
	inspectContainers(ctx, exec, containers)
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
	out, err := exec.Execute(ctx, "docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}")
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

// inspectContainers adds what docker inspect has to each container, in one
// call for all of them. A failed or unreadable inspect leaves the
// containers as docker ps gave them.
func inspectContainers(
	ctx context.Context,
	exec executor.Executor,
	containers []Container,
) {
	if len(containers) == 0 {
		return
	}
	args := []string{"inspect"}
	for _, c := range containers {
		args = append(args, c.ID)
	}
	// docker inspect exits non-zero when a container went away after docker
	// ps, and still gives the others on stdout.
	out, err := exec.Execute(ctx, "docker", args...)
	if err != nil && len(out) == 0 {
		return
	}
	var details []dockerInspect
	if json.Unmarshal(out, &details) != nil {
		return
	}
	byID := make(map[string]dockerInspect, len(details))
	for _, d := range details {
		byID[d.ID] = d
	}
	for i := range containers {
		d, ok := byID[containers[i].ID]
		if !ok {
			continue
		}
		applyInspect(&containers[i], d)
	}
}

// applyInspect sets the fields of c that docker inspect has, sorted so that
// the facts of one host are the same from run to run.
func applyInspect(
	c *Container,
	d dockerInspect,
) {
	c.Created = d.Created
	if d.State.Health != nil {
		c.Health = d.State.Health.Status
	}
	for _, e := range d.Config.Env {
		name, _, _ := strings.Cut(e, "=")
		c.EnvNames = append(c.EnvNames, name)
	}
	for _, m := range d.Mounts {
		c.Mounts = append(c.Mounts, Mount{
			Type: m.Type, Source: m.Source, Destination: m.Destination, ReadOnly: !m.RW,
		})
	}
	for _, port := range slices.Sorted(maps.Keys(d.NetworkSettings.Ports)) {
		bindings := d.NetworkSettings.Ports[port]
		if len(bindings) == 0 {
			c.Ports = append(c.Ports, Port{ContainerPort: port})

			continue
		}
		for _, b := range bindings {
			c.Ports = append(
				c.Ports,
				Port{ContainerPort: port, HostIP: b.HostIP, HostPort: b.HostPort},
			)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(d.NetworkSettings.Networks)) {
		n := d.NetworkSettings.Networks[name]
		c.Networks = append(
			c.Networks,
			Network{Name: name, IPAddress: n.IPAddress, Gateway: n.Gateway},
		)
	}
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
