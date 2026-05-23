# IPC

> **Status:** Implemented ✅

## Description

Reports Linux IPC subsystem kernel parameters — semaphore limits, message queue
limits, and shared memory limits — by reading the `/proc/sys/kernel/` sysctl
tree. These values are set at boot by `/etc/sysctl.conf` (or drop-ins) and
influence how much IPC resource any process or container can allocate.

The collector mirrors Ohai's `ipc` plugin methodology: it reads the same
`/proc/sys/kernel/sem`, `msgmnb`, `msgmni`, `msgmax`, `shmall`, `shmmax`, and
`shmmni` paths.

Consumers use this to:

- Audit IPC resource limits for database engines (PostgreSQL, Oracle) that tune
  shared memory extensively.
- Detect undersized semaphore limits before deploying workloads that need many
  semaphore sets.
- Compare configured limits against application recommendations.

## Collected Fields

| Field        | Type     | Description                                                                                      | Schema mapping   |
| ------------ | -------- | ------------------------------------------------------------------------------------------------ | ---------------- |
| `sem`        | `object` | Semaphore limits from `/proc/sys/kernel/sem`.                                                    | gohai convention |
| `sem.semmsl` | `string` | Maximum semaphores per semaphore set (`SEMMSL`).                                                 | gohai convention |
| `sem.semmns` | `string` | Maximum semaphores system-wide (`SEMMNS`).                                                       | gohai convention |
| `sem.semopm` | `string` | Maximum operations per `semop(2)` call (`SEMOPM`).                                               | gohai convention |
| `sem.semmni` | `string` | Maximum semaphore sets system-wide (`SEMMNI`).                                                   | gohai convention |
| `msg`        | `object` | Message queue limits.                                                                            | gohai convention |
| `msg.msgmnb` | `string` | Default maximum size in bytes for a single message queue, from `/proc/sys/kernel/msgmnb`.        | gohai convention |
| `msg.msgmni` | `string` | Maximum number of message queue identifiers, from `/proc/sys/kernel/msgmni`.                     | gohai convention |
| `msg.msgmax` | `string` | Maximum message size in bytes, from `/proc/sys/kernel/msgmax`.                                   | gohai convention |
| `shm`        | `object` | Shared memory limits.                                                                            | gohai convention |
| `shm.shmall` | `string` | Total amount of shared memory pages (or bytes on newer kernels), from `/proc/sys/kernel/shmall`. | gohai convention |
| `shm.shmmax` | `string` | Maximum size of a single shared memory segment in bytes, from `/proc/sys/kernel/shmmax`.         | gohai convention |
| `shm.shmmni` | `string` | Maximum number of shared memory segments, from `/proc/sys/kernel/shmmni`.                        | gohai convention |

All values are returned as strings preserving the raw kernel representation.
Missing sysctl files yield empty strings for their fields — not all kernels or
container configurations expose all parameters.

## Platform Support

| Platform | Supported                   |
| -------- | --------------------------- |
| Linux    | ✅                          |
| macOS    | nil (no `/proc/sys/kernel`) |

macOS does not expose `/proc/sys/kernel`. The Darwin variant returns `nil`.

## Example Output

```json
{
  "ipc": {
    "sem": {
      "semmsl": "250",
      "semmns": "32000",
      "semopm": "32",
      "semmni": "128"
    },
    "msg": {
      "msgmnb": "65536",
      "msgmni": "32000",
      "msgmax": "8192"
    },
    "shm": {
      "shmall": "18446744073692774399",
      "shmmax": "18446744073692774399",
      "shmmni": "4096"
    }
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("ipc"))
facts, _ := g.Collect(context.Background())
// facts.IPC.Sem, facts.IPC.Msg, facts.IPC.Shm
```

## Enable/Disable

```bash
gohai --collector.ipc      # enable
gohai --no-collector.ipc   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Read `/proc/sys/kernel/sem` via the injected `avfs.VFS`. The file contains
   four whitespace-separated values on a single line:
   `SEMMSL SEMMNS SEMOPM SEMMNI`. Parse all four into the `sem` struct. If fewer
   than four fields are present (truncated or misconfigured kernel), populate
   only the available ones.
2. Read `/proc/sys/kernel/msgmnb`, `/proc/sys/kernel/msgmni`, and
   `/proc/sys/kernel/msgmax` individually; each contains a single decimal value.
3. Read `/proc/sys/kernel/shmall`, `/proc/sys/kernel/shmmax`, and
   `/proc/sys/kernel/shmmni` individually; each contains a single decimal value.
4. Missing files are soft-missed to empty string — this happens in minimal
   container environments where `/proc/sys` is not fully bind-mounted or the
   kernel does not expose a given parameter.

On macOS: returns `nil`.

## Backing Library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for all sysctl reads.
- Go stdlib `strings.Fields` for the multi-value `sem` line parsing.
