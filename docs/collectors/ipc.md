# IPC

> **Status:** Implemented ✅

## Description

The collector reports the kernel parameters of the Linux IPC subsystem. These
are the semaphore limits, the message queue limits and the shared memory limits.
The collector reads them from the `/proc/sys/kernel/` sysctl tree.
`/etc/sysctl.conf` or its drop-in files set these values at boot. The values
control how much IPC resource a process or a container can allocate.

The collector uses the same method as Ohai's `ipc` plugin. It reads the same
`/proc/sys/kernel/sem`, `msgmnb`, `msgmni`, `msgmax`, `shmall`, `shmmax` and
`shmmni` paths.

Consumers use this data to:

- Audit the IPC resource limits for database engines that tune shared memory
  very much, for example PostgreSQL and Oracle.
- Find semaphore limits that are too small, before you deploy workloads that
  need many semaphore sets.
- Compare the configured limits with the recommendations of an application.

## Collected fields

| Field        | Type     | Description                                                                                         | Schema mapping   |
| ------------ | -------- | --------------------------------------------------------------------------------------------------- | ---------------- |
| `sem`        | `object` | The semaphore limits, from `/proc/sys/kernel/sem`.                                                  | gohai convention |
| `sem.semmsl` | `string` | The maximum number of semaphores in one semaphore set (`SEMMSL`).                                   | gohai convention |
| `sem.semmns` | `string` | The maximum number of semaphores on the system (`SEMMNS`).                                          | gohai convention |
| `sem.semopm` | `string` | The maximum number of operations in one `semop(2)` call (`SEMOPM`).                                 | gohai convention |
| `sem.semmni` | `string` | The maximum number of semaphore sets on the system (`SEMMNI`).                                      | gohai convention |
| `msg`        | `object` | The message queue limits.                                                                           | gohai convention |
| `msg.msgmnb` | `string` | The default maximum size in bytes of one message queue, from `/proc/sys/kernel/msgmnb`.             | gohai convention |
| `msg.msgmni` | `string` | The maximum number of message queue identifiers, from `/proc/sys/kernel/msgmni`.                    | gohai convention |
| `msg.msgmax` | `string` | The maximum message size in bytes, from `/proc/sys/kernel/msgmax`.                                  | gohai convention |
| `shm`        | `object` | The shared memory limits.                                                                           | gohai convention |
| `shm.shmall` | `string` | The total quantity of shared memory pages (bytes on newer kernels), from `/proc/sys/kernel/shmall`. | gohai convention |
| `shm.shmmax` | `string` | The maximum size in bytes of one shared memory segment, from `/proc/sys/kernel/shmmax`.             | gohai convention |
| `shm.shmmni` | `string` | The maximum number of shared memory segments, from `/proc/sys/kernel/shmmni`.                       | gohai convention |

The collector returns all values as strings. The strings keep the raw format of
the kernel. If a sysctl file is missing, its field is an empty string. Not all
kernels or container configurations have all parameters.

## Platform support

| Platform | Supported                   |
| -------- | --------------------------- |
| Linux    | ✅                          |
| macOS    | nil (no `/proc/sys/kernel`) |

macOS has no `/proc/sys/kernel`. The Darwin variant returns `nil`.

## Example output

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

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("ipc"))
facts, _ := g.Collect(context.Background())
// facts.IPC.Sem, facts.IPC.Msg, facts.IPC.Shm
```

## Enable/disable

```bash
gohai collect --collector.ipc      # enable
gohai collect --no-collector.ipc   # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/ipc.rb` reads the same `/proc/sys/kernel/` files for the
semaphore, message queue and shared memory limits. gohai uses the same method as
Ohai. It reads the raw kernel tunables and does not parse the `ipcs -l` output.
The collector returns all values as strings to keep the format of the kernel.
This includes the large 64-bit values in `shmall` and `shmmax`.

On Linux:

1. The collector reads `/proc/sys/kernel/sem` through the injected `avfs.VFS`.
   The file has four values on one line, with whitespace between them:
   `SEMMSL SEMMNS SEMOPM SEMMNI`. The collector parses all four into the `sem`
   struct. If the line has fewer than four fields, the collector sets only the
   fields that are there. This occurs if the file is truncated or the kernel is
   not configured correctly.
2. The collector reads `/proc/sys/kernel/msgmnb`, `/proc/sys/kernel/msgmni` and
   `/proc/sys/kernel/msgmax` one at a time. Each file has one decimal value.
3. The collector reads `/proc/sys/kernel/shmall`, `/proc/sys/kernel/shmmax` and
   `/proc/sys/kernel/shmmni` one at a time. Each file has one decimal value.
4. If a file is missing, its field is an empty string. This occurs in minimal
   containers that do not bind-mount all of `/proc/sys`. It also occurs if the
   kernel does not have that parameter.

On macOS, the collector returns `nil`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for all sysctl reads.
  Production uses `osfs`. Tests use `memfs`.
- The Go standard library `strings.Fields` to parse the `sem` line, which has
  more than one value.
