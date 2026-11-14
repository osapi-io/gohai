# Kernel modules

> **Status:** Implemented ✅

## Description

The collector lists the loaded kernel modules on Linux and the legacy kernel
extensions (kexts) on macOS. Each module entry has the size, the reference count
and the version. Thus downstream tools can correlate the loaded modules with CVE
feeds or policy baselines.

This collector was split from the [`kernel`](kernel.md) collector in 2026-04.
The reason is that it takes much time to list the modules. On macOS,
`kextstat -k -l` takes about 280ms. On Linux, the collector reads
`/sys/module/<name>/version` for each loaded module, which is a large part of
the `/sys` tree. Most consumers want only the kernel identity (name, release,
machine). They must not pay that cost by default. Thus this collector is opt-in.
Enable it only if you need the module list.

Consumers use this data to:

- Audit the loaded modules and kexts for security compliance. For example: Is
  `cramfs` loaded? Is an unsigned third-party module inserted? Which EDR kext is
  active?
- Correlate the loaded modules with CVE feeds (module version to known CVE).
- Verify that EDR and anti-malware agents are loaded. Their kexts or modules
  must always be there.

## Collected fields

| Field     | Type            | Description                         | Schema mapping            |
| --------- | --------------- | ----------------------------------- | ------------------------- |
| `modules` | map[name]Module | The loaded kernel modules or kexts. | No direct schema mapping. |

### `Module` entry

| Field      | Type   | Description                                                                                             |
| ---------- | ------ | ------------------------------------------------------------------------------------------------------- |
| `size`     | uint64 | The memory size in bytes. Linux: from `/proc/modules`. macOS: the `kextstat` size column, in hex.       |
| `refcount` | int    | The number of holders or dependent modules.                                                             |
| `version`  | string | Linux: `/sys/module/<name>/version`, if it is there. macOS: the version in parentheses from `kextstat`. |
| `index`    | int    | macOS only: the kext load order index from `kextstat`. Not in the output on Linux.                      |

On macOS, `modules` lists **only legacy kernel extensions**. macOS 11 and later
have System Extensions under `/Library/SystemExtensions/`. To read them, a tool
must use `systemextensionsctl`. The collector does not query it at this time.

## Platform support

| Platform | Supported                                          |
| -------- | -------------------------------------------------- |
| Linux    | ✅ (`/proc/modules` and `/sys/module/<n>/version`) |
| macOS    | ✅ (`kextstat -k -l`)                              |

## Example output

### Linux

```json
{
  "kernel_modules": {
    "modules": {
      "nf_tables": { "size": 217088, "refcount": 25, "version": "1.2.3" },
      "ipv6": { "size": 557056, "refcount": 24 }
    }
  }
}
```

### macOS

```json
{
  "kernel_modules": {
    "modules": {
      "com.apple.iokit.IOPCIFamily": {
        "size": 2216,
        "refcount": 0,
        "version": "2.9"
      }
    }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("kernel", "kernel_modules"))
facts, _ := g.Collect(context.Background())

mods := facts.KernelModules
if mods == nil { return }

// Security check: is an unsigned module loaded?
if _, ok := mods.Modules["suspicious_mod"]; ok {
    log.Println("unexpected kernel module present")
}
```

## Enable/disable

```bash
gohai --collector.kernel_modules      # enable (opt-in)
gohai --no-collector.kernel_modules   # disable (default)
gohai --category=system                # enabling the system category also pulls this in
```

## Dependencies

None.

## Data sources

In Ohai, the kernel plugin `kernel.rb` lists the modules. It parses
`/proc/modules` on Linux and `kextstat` on macOS. gohai puts the modules in a
separate collector, so that you can enable or disable them independently. But
gohai uses the same parse method as Ohai on the two platforms.

On Linux:

1. The collector reads `/proc/modules` through the injected `avfs.VFS`. It
   parses the file one line at a time to get the `name`, `size` and `refcount`
   of each module. If the file is missing, the collector returns an empty `Info`
   and no error. This occurs on systems that do not mount `procfs`.
2. For each module, the collector reads `/sys/module/<name>/version`, if it is
   there. It removes the whitespace and puts the value in `modules[].version`.
   If the file is missing, the field stays empty. Many built-in modules do not
   have a version. This is the same as Ohai, which also gives no message if the
   file is missing.

On macOS:

1. The collector runs `kextstat -k -l` through the shared `internal/executor`
   runner. Each row has fixed columns:
   `Index Refs Address Size Wired Name (Version)`. The collector parses each row
   into a `Module` with `name`, `version`, `size` and `refcount`. It removes the
   parentheses from `version`. `size` is in hex. If `kextstat` is missing or
   gives a non-zero exit code, the modules map stays empty. macOS has no
   equivalent of `/proc/modules`.
2. The collector does not read System Extensions (macOS 11 and later). A future
   change can use `systemextensionsctl list`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs). The virtual filesystem
  that reads `/proc/modules` and `/sys/module/<name>/version` on Linux. Tests
  inject `memfs` with fixed content.
- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `kextstat` on macOS. Tests mock it with
  `go.uber.org/mock`.
