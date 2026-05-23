# docker

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

| Field                 | Type   | Schema mapping                | Notes                                             |
| --------------------- | ------ | ----------------------------- | ------------------------------------------------- |
| `version`             | string | OCSF `device_hw_info.version` | Docker server version, e.g. `24.0.5`              |
| `containers`          | list   | —                             | All containers (running and stopped)              |
| `containers[].id`     | string | OCSF `container.uid`          | Short container ID                                |
| `containers[].name`   | string | OCSF `container.name`         | Container name, leading `/` stripped              |
| `containers[].image`  | string | OCSF `container.image.name`   | Image reference used to start the container       |
| `containers[].state`  | string | OCSF `container.status`       | Docker state: `running`, `exited`, `paused`, etc. |
| `containers[].status` | string | gohai convention: `status`    | Human-readable status string from `docker ps`     |
| `images`              | list   | —                             | Locally pulled images                             |
| `images[].id`         | string | OCSF `container.image.uid`    | Image ID (full SHA256 digest)                     |
| `images[].repository` | string | OCSF `container.image.name`   | Repository name                                   |
| `images[].tag`        | string | gohai convention: `tag`       | Image tag                                         |
| `images[].size`       | string | gohai convention: `size`      | Human-readable size from `docker images`          |

## Platform Support

| Platform | Supported | Backing source                              |
| -------- | --------- | ------------------------------------------- |
| Linux    | Yes       | Docker CLI via `executor.Executor`          |
| macOS    | Yes       | Docker Desktop exposes the same CLI surface |

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
g := gohai.New(gohai.WithEnabled("docker"))
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

## Backing Library

`internal/executor.Executor` for command execution.
