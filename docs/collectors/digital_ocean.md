# DigitalOcean

> **Status:** Implemented ✅

## Description

The collector reads the DigitalOcean droplet metadata from the link-local server
at `http://169.254.169.254/metadata/v1.json`. If the collector cannot reach the
endpoint, or the DMI signature is different, it returns nil and no error. If
`facts.DigitalOcean != nil`, the host is on DigitalOcean.

The collector runs only if `dmi.BIOS.Manufacturer == "DigitalOcean"`. The match
must be exact. This check is the same as Ohai's `has_do_dmi?`. The collector
removes `vendor_data` from the response, because it frequently contains
cloud-init user scripts with credentials. Ohai also removes it.

## Collected fields

| Field              | Type          | Description                                            | Schema mapping            |
| ------------------ | ------------- | ------------------------------------------------------ | ------------------------- |
| `id`               | `int64`       | The numeric ID of the DigitalOcean droplet.            | OTel `cloud.resource_id`  |
| `hostname`         | `string`      | The droplet hostname.                                  | OCSF `device.hostname`    |
| `region`           | `string`      | The DigitalOcean region, for example `nyc3`.           | OTel `cloud.region`       |
| `public_keys`      | `[]string`    | The SSH public keys of the droplet.                    | No direct schema mapping. |
| `tags`             | `[]string`    | The tags that the user defined.                        | No direct schema mapping. |
| `features`         | `[]string`    | The DO feature flags that are enabled on this droplet. | No direct schema mapping. |
| `floating_ip`      | `string`      | The floating IPv4 of the droplet, if it has one.       | No direct schema mapping. |
| `reserved_ip`      | `string`      | The newer DO replacement for floating_ip (IPv4).       | No direct schema mapping. |
| `auth_key`         | `string`      | The DO internal auth token. Frequently empty.          | No direct schema mapping. |
| `user_data`        | `string`      | The cloud-init data that the user gave.                | No direct schema mapping. |
| `ipv4_nameservers` | `[]string`    | The DNS resolvers.                                     | No direct schema mapping. |
| `interfaces`       | `[]Interface` | The attached VNICs. Refer to Interface.                | OCSF `network_interface`  |

### Interface

| Field          | Type     | Description                                         |
| -------------- | -------- | --------------------------------------------------- |
| `scope`        | `string` | `"public"` or `"private"`.                          |
| `mac`          | `string` | The MAC address.                                    |
| `type`         | `string` | The interface type.                                 |
| `ipv4`         | `string` | The primary IPv4 address.                           |
| `ipv4_netmask` | `string` | The IPv4 netmask.                                   |
| `ipv4_gateway` | `string` | The IPv4 gateway.                                   |
| `ipv6`         | `string` | The primary IPv6 address.                           |
| `ipv6_cidr`    | `int`    | The IPv6 CIDR prefix length.                        |
| `ipv6_gateway` | `string` | The IPv6 gateway.                                   |
| `anchor_ipv4`  | `string` | The DO anchor IP, for the network between droplets. |

## Platform support

| Platform | Supported                            |
| -------- | ------------------------------------ |
| Linux    | ✅                                   |
| macOS    | ✅ (useful only on a DO droplet)     |
| Other    | ✅ (only meaningful on a DO droplet) |

## Example output

```json
{
  "digital_ocean": {
    "id": 123456,
    "hostname": "web-1",
    "region": "nyc3",
    "tags": ["prod"],
    "floating_ip": "138.1.2.3",
    "interfaces": [
      { "scope": "public", "ipv4": "138.4.5.6", "mac": "f6:a0:..." },
      { "scope": "private", "ipv4": "10.132.0.2", "mac": "f6:a0:..." }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("digital_ocean"))
facts, _ := g.Collect(context.Background())

if facts.DigitalOcean != nil {
    fmt.Println("running on DO,", facts.DigitalOcean.Region)
}
```

## Enable/disable

```bash
gohai collect --collector.digital_ocean      # enable (opt-in)
gohai collect --no-collector.digital_ocean   # disable (default)
gohai collect --category=cloud               # pulls this + all cloud collectors
```

## Dependencies

`dmi`. DO writes `"DigitalOcean"` as `bios_vendor`. The dmi collector reports it
as `BIOS.Manufacturer`. The collector makes the HTTP call only on an exact
match. If `dmi` has no data because the user disabled it, the check passes and
the collector tries the endpoint.

## Data sources

1. **DMI gate:** `dmi.BIOS.Manufacturer == "DigitalOcean"` (exact match). This
   is the same as Ohai's `has_do_dmi?`.
2. **Endpoint:** `http://169.254.169.254/metadata/v1.json`. It returns one JSON
   document.
3. **User-Agent:** `gohai` (the cloudmetadata default).
4. **Timeout:** 6 seconds. This is the same as Ohai's `read_timeout` in
   `mixin/do_metadata.rb`.
5. **Failure handling:** if the transport fails, or the endpoint returns 404 or
   an other error, the collector returns `(nil, nil)`.
6. **Security scrub:** the collector removes `vendor_data` from the output, as
   Ohai does. It can contain cloud-init scripts and credentials. The collector
   keeps `user_data` and `auth_key`. `user_data` is the cloud-init payload of
   the droplet owner. `auth_key` is the DO internal token, which Ohai also
   reports.
7. **Transformation:** the collector moves `reserved_ip` from the nested
   `reserved_ip.ipv4.ip_address` path to the flat top-level field. `reserved_ip`
   is the newer DO replacement for the legacy `floating_ip`. Both fields are in
   the output, because older droplets still use `floating_ip`.

The method is the same as Ohai's `Ohai::Mixin::DoMetadata`. It uses the same
endpoint, the same DMI gate, the same `vendor_data` removal and the same 6s
timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors. Its transport is the `HTTP` of the
  sources, so a fake host can answer the metadata reads.
