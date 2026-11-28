# Linode

> **Status:** Implemented ✅

## Description

The collector finds Linode hosts and reports their public and private IPv4
addresses. All other cloud collectors in gohai use a metadata HTTP endpoint. The
Linode collector **does not**, and Ohai's Linode plugin does not either. The
collector finds a Linode host if one of three signals is true. No signal is a
hint file:

- The DMI `sys_vendor` is `Linode`.
- `/etc/apt/sources.list`, or a file in `/etc/apt/sources.list.d`, contains
  `"linode"`. Linode's official images include an apt mirror that Linode hosts.
- The resolved FQDN or domain of the host contains `"linode"`, for example
  `members.linode.com`. This is Linode's reverse-DNS suffix.

The last two are Ohai's `looks_like_linode?`.

The data comes from the network interfaces of the host.

## Collected fields

| Field        | Type     | Description                                                | Schema mapping            |
| ------------ | -------- | ---------------------------------------------------------- | ------------------------- |
| `public_ip`  | `string` | The first IPv4 address on `eth0` that is not link-local.   | No direct schema mapping. |
| `private_ip` | `string` | The first IPv4 address on `eth0:1` that is not link-local. | No direct schema mapping. |

## Platform support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (returns `nil`, no Linode signature) |
| Other    | ✅ (returns `nil`, no Linode signature) |

## Example output

```json
{
  "linode": {
    "public_ip": "50.116.1.2",
    "private_ip": "192.168.128.3"
  }
}
```

## SDK usage

```go
import (
    "context"
    "fmt"
    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("linode"))
facts, _ := g.Collect(context.Background())

if facts.Linode != nil {
    fmt.Println("running on Linode, public IP", facts.Linode.PublicIP)
}
```

## Enable/disable

```bash
gohai collect --collector.linode      # enable (opt-in)
gohai collect --no-collector.linode   # disable (default)
gohai collect --category=cloud        # pulls this + all cloud collectors
```

## Dependencies

`dmi` and `hostname`. The DMI vendor signal needs `dmi`. The FQDN and domain
signal needs `hostname`.

## Data sources

1. The collector decides whether the host is a Linode host:
   - **DMI.** When the `dmi` collector ran, its `sys_vendor` decides alone. It
     is `Linode`, or `Akamai` on newer Akamai Cloud VMs. cloud-init's
     `ds-identify` finds its Akamai data source by the same values. A host
     with another vendor is not a Linode host, even if it uses
     `mirrors.linode.com` or is named under `linode.com`. A DMI with no product
     table is not a Linode host either.
   - **apt and domain, only when `dmi` did not run.** These are best effort,
     as in Ohai. Either one makes the host a Linode host:
     - `/etc/apt/sources.list`, or a file in `/etc/apt/sources.list.d`,
       contains `"linode"`. The match is not case-sensitive. Newer images, such
       as Ubuntu 24.04 and Debian 13, keep their sources in
       `sources.list.d/*.sources`. This is Ohai's `has_linode_apt_repos?`, with
       the directory added. With no filesystem, apt is no signal.
     - The `FQDN` or `Domain` from the `hostname` collector is `linode.com` or
       ends in `.linode.com`, such as `members.linode.com`. Ohai's
       `has_linode_domain?` matches the substring `linode`, so a name such as
       `linode-builder.example.com` was a match there and is not here.
2. The collector runs `ip -o -4 addr show` through the shared `source.Runner`.
   Each line names the label of its address: the interface, such as `eth0`, or
   an alias, such as `eth0:1`. For `eth0` and `eth0:1`, the collector uses the
   first IPv4 address that is not link-local. If a label is missing, or `ip` is
   not installed, its field is an empty string. An `ip` that runs and fails, or
   a cancelled context, is an error.

The method is the same as Ohai's Linode plugin, with the DMI signal and the
`sources.list.d` directory added. It uses the same domain signal and the same
`eth0` and `eth0:1` interfaces. gohai does not query Linode's
`metadata.linode.com` API. Ohai does not query it either.

## Backing library

- [`source.Runner`](../../pkg/gohai/source) to run `ip -o -4 addr show`.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read
  `/etc/apt/sources.list` and `/etc/apt/sources.list.d`. Production uses `osfs`.
  Tests use `memfs`.
