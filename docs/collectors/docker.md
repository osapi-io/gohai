# Docker

> **Status:** Implemented ✅

## Description

The collector reports the Docker server version, the running and stopped
containers, and the local images. If Docker is not on the PATH, or if the
collector cannot connect to the daemon, Collect returns nil and no error. This
is the same as Ohai's `docker.rb`, which does not fail when Docker is missing.

Ohai's docker plugin reports the total container counts from `docker info`.
gohai also lists each container and each image with `docker ps -a --no-trunc`
and `docker images`. Thus consumers can see the state of each container. Then
one `docker inspect` call, with the ID of each container, gives the time each
container was made, its health, its ports, its mounts, its networks and the
names of its environment variables. When `docker inspect` fails, each container
has only what `docker ps` gives.

The collector reads through the `docker` CLI, which `source.Sources` runs. A
host that runs `dockerd` but has no `docker` CLI on the PATH has no container
facts, also when the Docker socket is there.

The environment of a container often holds a secret, such as a database
password. Thus the collector gives only the names of the variables, never their
values.

## Collected fields

| Field                    | Type     | Description                                                                                                                 | Schema mapping              |
| ------------------------ | -------- | --------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `version`                | `string` | The Docker server version, for example `24.0.5`.                                                                            | No direct OCSF/OTel mapping |
| `containers`             | `list`   | All containers, running and stopped.                                                                                        | None                        |
| `containers[].id`        | `string` | The full container ID, which `docker start`, `stop`, `exec` and `rm` take.                                                  | OCSF `container.uid`        |
| `containers[].name`      | `string` | The container name, without the leading `/`.                                                                                | OCSF `container.name`       |
| `containers[].image`     | `string` | The image reference that started the container.                                                                             | OCSF `container.image.name` |
| `containers[].state`     | `string` | The Docker state, for example `running`, `exited` or `paused`.                                                              | No direct OCSF/OTel mapping |
| `containers[].status`    | `string` | The status text from `docker ps`.                                                                                           | No direct OCSF/OTel mapping |
| `containers[].created`   | `string` | When the container was made, RFC 3339, from `docker inspect`.                                                               | No direct OCSF/OTel mapping |
| `containers[].health`    | `string` | The status of its health check, such as `healthy`. Empty when it has none.                                                  | No direct OCSF/OTel mapping |
| `containers[].ports`     | `list`   | Each port and protocol (`container_port`, such as `80/tcp`), and the `host_ip` and `host_port` it is published on, by port. | No direct OCSF/OTel mapping |
| `containers[].mounts`    | `list`   | Each volume and bind mount: `type`, `source`, `destination`, `read_only`.                                                   | No direct OCSF/OTel mapping |
| `containers[].networks`  | `list`   | Each network, by name, with the container's `ip_address` and the `gateway`.                                                 | No direct OCSF/OTel mapping |
| `containers[].env_names` | `list`   | The names of its environment variables. The values are left out.                                                            | No direct OCSF/OTel mapping |
| `images`                 | `list`   | The images on the local host.                                                                                               | None                        |
| `images[].id`            | `string` | The image ID, the full SHA256 digest.                                                                                       | No direct OCSF/OTel mapping |
| `images[].repository`    | `string` | The repository name.                                                                                                        | No direct OCSF/OTel mapping |
| `images[].tag`           | `string` | The image tag.                                                                                                              | No direct OCSF/OTel mapping |
| `images[].size`          | `string` | The size as text, from `docker images`.                                                                                     | No direct OCSF/OTel mapping |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |

## Example output

```json
{
  "docker": {
    "version": "24.0.5",
    "containers": [
      {
        "id": "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
        "name": "web",
        "image": "nginx:latest",
        "state": "running",
        "status": "Up 2 hours",
        "created": "2026-11-22T10:00:00.123456789Z",
        "health": "healthy",
        "ports": [
          { "container_port": "80/tcp", "host_ip": "0.0.0.0", "host_port": "8080" }
        ],
        "mounts": [
          {
            "type": "volume",
            "source": "/var/lib/docker/volumes/data/_data",
            "destination": "/data",
            "read_only": false
          }
        ],
        "networks": [
          { "name": "bridge", "ip_address": "172.17.0.2", "gateway": "172.17.0.1" }
        ],
        "env_names": ["PATH", "NGINX_VERSION"]
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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("docker"))
facts, _ := g.Collect(ctx)
```

## Enable/disable

Default: **disabled** (opt-in). Docker is not on all hosts. On a host with many
containers, the commands that list containers can be slow.

```bash
gohai collect --collector.docker          # enable
gohai collect --no-collector.docker       # disable
```

## Dependencies

None.

## Data sources

Ohai's `docker.rb` plugin gets the Docker version and the total container counts
from `docker info`. gohai also lists each container and each image with
`docker ps` and `docker images`. Thus consumers see the state of each container,
not only the totals.

On both Linux and macOS:

1. The collector runs `docker version --format '{{.Server.Version}}'`. This
   finds if Docker is there and gets the version. If the command fails, Collect
   returns nil immediately and no error. The command fails when the daemon does
   not run or when Docker is not installed. Ohai's plugin also continues only
   when `docker info` succeeds.
2. The collector runs `docker ps -a --format '{{json .}}'` to list all
   containers. The output is NDJSON, one JSON object on each line. The collector
   skips lines that are not valid JSON and blank lines. This is a gohai
   extension. Ohai does not list each container.
3. The collector runs `docker images --format '{{json .}}'` to list the local
   images. It parses the NDJSON in the same way. If `docker ps` or
   `docker images` fails, the related field is an empty list. This is also a
   gohai extension that Ohai does not have.
4. Docker gives the container `Names` field with a leading `/`. The collector
   removes it before it stores the name.

## Backing library

`source.Runner` to run the commands.
