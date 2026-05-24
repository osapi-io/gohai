# Docker

> **Status:** Implemented ✅

## Description

Reports Docker server version, running and stopped containers, and local images.
If Docker is not on PATH or the daemon is unreachable, Collect returns nil
gracefully — mirrors Ohai's `docker.rb` stance of not failing when Docker is
absent.

Ohai's docker plugin focuses on aggregate container counts from `docker info`.
gohai extends that by enumerating individual containers and images via
`docker ps -a` and `docker images`, which gives consumers per-container state
visibility.

## Collected Fields

| Field                 | Type     | Description                                       | Schema mapping                |
| --------------------- | -------- | ------------------------------------------------- | ----------------------------- |
| `version`             | `string` | Docker server version, e.g. `24.0.5`              | No direct OCSF/OTel mapping   |
| `containers`          | `list`   | All containers (running and stopped)               | —                             |
| `containers[].id`     | `string` | Short container ID                                 | OCSF `container.uid`          |
| `containers[].name`   | `string` | Container name, leading `/` stripped               | OCSF `container.name`         |
| `containers[].image`  | `string` | Image reference used to start the container        | OCSF `container.image.name`   |
| `containers[].state`  | `string` | Docker state: `running`, `exited`, `paused`, etc.  | No direct OCSF/OTel mapping   |
| `containers[].status` | `string` | Human-readable status string from `docker ps`      | No direct OCSF/OTel mapping   |
| `images`              | `list`   | Locally pulled images                              | —                             |
| `images[].id`         | `string` | Image ID (full SHA256 digest)                      | No direct OCSF/OTel mapping   |
| `images[].repository` | `string` | Repository name                                    | No direct OCSF/OTel mapping   |
| `images[].tag`        | `string` | Image tag                                          | No direct OCSF/OTel mapping   |
| `images[].size`       | `string` | Human-readable size from `docker images`           | No direct OCSF/OTel mapping   |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |

## Example Output

```json
{
  "docker": {
    "version": "24.0.5",
    "containers": [
      {
        "id": "abc123def456",
        "name": "web",
        "image": "nginx:latest",
        "state": "running",
        "status": "Up 2 hours"
      }
    ],
    "images": [
      {
        "id": "sha256:abc123",
        "repository": "nginx",
        "tag": "latest",
        "size": "187MB"
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("docker"))
facts, _ := g.Collect(ctx)
```

## Enable/Disable

Default: **disabled** (opt-in). Docker may not be present on the host; the
container-listing commands can be slow on hosts with many containers.

```bash
gohai --collector.docker          # enable
gohai --no-collector.docker       # disable
```

## Dependencies

None.

## Data Sources

On both Linux and macOS:

1. Runs `docker version --format '{{.Server.Version}}'` to probe presence and
   retrieve the version. If this command fails (daemon not running, docker not
   installed), Collect returns nil immediately — no error.
2. Runs `docker ps -a --format '{{json .}}'` to list all containers. Output is
   NDJSON (one JSON object per line). Invalid JSON lines and blank lines are
   skipped.
3. Runs `docker images --format '{{json .}}'` to list local images. Same NDJSON
   parsing. If either `docker ps` or `docker images` fails, the respective field
   is returned as an empty list.
4. Container `Names` field from Docker has a leading `/` — stripped before
   storing.

## Backing library

`internal/executor.Executor` for command execution.
