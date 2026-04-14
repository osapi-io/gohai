# Linode

> **Status:** Implemented ✅

## Description

Detects Linode hosts and reports their public / private IPv4 addresses. Unlike
every other cloud collector in gohai, Linode **does not** use a metadata HTTP
endpoint — Ohai's Linode plugin doesn't either. Detection ORs two non-hint
signals (matches Ohai's `looks_like_linode?`):

- `/etc/apt/sources.list` contains `"linode"` (Linode's official images ship a
  Linode-hosted apt mirror).
- The host's resolved FQDN or domain contains `"linode"` (e.g.
  `members.linode.com` — Linode's reverse-DNS suffix).

Data comes from the host's own network interfaces.

## Collected Fields

| Field        | Type     | Description                            | Schema mapping            |
| ------------ | -------- | -------------------------------------- | ------------------------- |
| `public_ip`  | `string` | First non-link-local IPv4 on `eth0`.   | No direct schema mapping. |
| `private_ip` | `string` | First non-link-local IPv4 on `eth0:1`. | No direct schema mapping. |

## Platform Support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅                                     |
| macOS    | ✅ (returns nil — no Linode signature) |
| Other    | ✅ (returns nil — no Linode signature) |

## Example Output

```json
{
  "linode": {
    "public_ip": "50.116.1.2",
    "private_ip": "192.168.128.3"
  }
}
```

## SDK Usage

```go
import (
    "context"
    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("linode"))
facts, _ := g.Collect(context.Background())

if facts.Linode != nil {
    fmt.Println("running on Linode, public IP", facts.Linode.PublicIP)
}
```

## Enable/Disable

```bash
gohai --collector.linode      # enable (opt-in)
gohai --no-collector.linode   # disable (default)
gohai --category=cloud        # pulls this + all cloud collectors
```

## Dependencies

`hostname` — needed for the FQDN/Domain detection signal that mirrors Ohai's
`has_linode_domain?`.

## Data Sources

1. **Detection chain** (any signal triggers detection):
   - **apt:** `/etc/apt/sources.list` contains `"linode"` (case-insensitive).
     Matches Ohai's `has_linode_apt_repos?`.
   - **domain:** the prior `hostname` collector's `FQDN` or `Domain` contains
     `"linode"`. Matches Ohai's `has_linode_domain?`.
2. **Interface reads:** `net.InterfaceByName("eth0")` / `"eth0:1"` via Go
   stdlib. First non-link-local IPv4 on each wins. Missing interface → empty
   string on that field.

Mirrors Ohai's Linode plugin methodology — same apt + domain detection chain,
same eth0 / eth0:1 interface scheme. We do not query Linode's
`metadata.linode.com` API (neither does Ohai).

## Backing library

- Go stdlib `net.InterfaceByName` + `os.ReadFile`.
