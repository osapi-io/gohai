# DNS

> **Status:** Implemented ✅

## Description

The collector reports the host's resolver configuration: the name servers,
search domains and options of `/etc/resolv.conf`, and the name servers and
domains of each interface where the host keeps them apart. On Linux, that is
systemd-resolved. On macOS, that is the scoped resolvers of `configd`.

Consumers use this data to:

- See which name servers a host asks, and through which interface.
- Find hosts whose search domains or options differ from the fleet.

DefaultEnabled is `false`.

## Collected fields

| Field                      | Type       | Description                                                                                                                                                                                                                                                   | Schema mapping            |
| -------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `nameservers`              | `[]string` | The `nameserver` lines of `/etc/resolv.conf`, in order.                                                                                                                                                                                                       | No direct schema mapping. |
| `search`                   | `[]string` | The last `search` or `domain` line.                                                                                                                                                                                                                           | No direct schema mapping. |
| `options`                  | `[]string` | The `options` lines, each option a value.                                                                                                                                                                                                                     | No direct schema mapping. |
| `interfaces`               | `map`      | The resolver of each interface that has one, by name.                                                                                                                                                                                                         | No direct schema mapping. |
| `interfaces.*.nameservers` | `[]string` | The name servers of the interface, without an IPv6 zone, a port or a TLS server name.                                                                                                                                                                         | No direct schema mapping. |
| `interfaces.*.domains`     | `[]string` | The domains of the interface. `~.` routes every name there.                                                                                                                                                                                                   | No direct schema mapping. |
| `global`                   | `object`   | The resolver of no interface: its `nameservers` and `domains`. On Linux, systemd-resolved's (`resolvectl`'s `Global` line). With systemd-resolved, `resolv.conf` names only the stub, so these are the upstream servers. On macOS, scutil's default resolver. | No direct schema mapping. |

## Platform support

| Platform | Supported                                        |
| -------- | ------------------------------------------------ |
| Linux    | ✅ (`/etc/resolv.conf`, `resolvectl` when there) |
| macOS    | ✅ (`/etc/resolv.conf`, `scutil --dns`)          |

## Example output

### Typical Linux with systemd-resolved

```json
{
  "dns": {
    "nameservers": ["127.0.0.53"],
    "search": ["corp.example"],
    "options": ["edns0", "trust-ad"],
    "interfaces": {
      "eth0": {
        "nameservers": ["10.0.0.1"],
        "domains": ["corp.example"]
      }
    }
  }
}
```

### Typical macOS

```json
{
  "dns": {
    "nameservers": ["1.1.1.1"],
    "interfaces": {
      "en0": { "nameservers": ["1.1.1.1"] }
    }
  }
}
```

## SDK usage

```go
g, err := gohai.New(gohai.WithCollectors("dns"))
if err != nil {
    return err
}
facts, err := g.Collect(ctx)
if err != nil {
    return err
}
fmt.Println(facts.DNS.Nameservers)
```

## Enable/disable

```bash
gohai collect --collector.dns      # enable (not in the defaults)
gohai collect --no-collector.dns   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. The collector reads `/etc/resolv.conf` through its filesystem. A line that
   starts with `#` or `;` is a comment. If the file is not there, the top-level
   fields stay empty. If the collector cannot read it, the collector returns an
   error.
2. When `/run/systemd/resolve` exists, systemd-resolved runs, and the collector
   runs `resolvectl dns` and `resolvectl domain`. Each `Link N (name): …` line
   gives that interface's name servers or domains, and the `Global: …` line
   gives `global`. A server loses its `%zone`, its `#name` and its port. Without
   that directory, or without `resolvectl`, the interfaces stay empty. If
   `resolvectl` fails while systemd-resolved runs, the collector keeps the facts
   of `/etc/resolv.conf` with the error, so the status is `partial`. A cancelled
   context gives no fact.

On macOS:

1. The collector reads `/etc/resolv.conf`, which macOS writes from its primary
   resolver.
2. It runs `scutil --dns`. Each `resolver #N` block with an `if_index` gives
   that interface's `nameserver[…]` and `search domain[…]` values. scutil prints
   the scoped resolvers a second time. The first block of an interface wins.
3. The default resolver gives `global`. It is the first block of the unscoped
   `DNS configuration` section that has name servers, has no `domain` line (a
   resolver for one domain only, such as `local`) and is not `Supplemental` (a
   resolver that a VPN adds for its own names). macOS writes `resolv.conf` from
   it.
4. If `scutil` fails, the collector keeps the facts of `/etc/resolv.conf` with
   the error, so the status is `partial`. A cancelled context gives no fact.

Ohai has no DNS plugin. The data is configuration, not the state of a daemon, so
it is a gohai fact.

## Backing library

None. The collector reads the files through its filesystem, and runs
`resolvectl` and `scutil` through its command runner.
