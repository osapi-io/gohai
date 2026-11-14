# Scaleway

> **Status:** Implemented ✅

## Description

The collector reads the Scaleway instance metadata from the link-local server at
`http://169.254.42.42/conf?format=json`. Scaleway uses `169.254.42.42`, not the
more common `169.254.169.254`. If `facts.Scaleway != nil`, the host is a
Scaleway instance. On other hosts, the collector gives no data and no error.

For the detection, the collector looks for the substring `"scaleway"` in
`/proc/cmdline`. The Scaleway boot environment always puts its name in the
kernel command line.

## Collected fields

| Field               | Type          | Description                                                                                                    | Schema mapping                 |
| ------------------- | ------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `id`                | `string`      | The instance UUID.                                                                                             | OTel `cloud.resource_id`       |
| `name`              | `string`      | The instance display name.                                                                                     | OTel `host.name`               |
| `hostname`          | `string`      | The instance hostname.                                                                                         | OCSF `device.hostname`         |
| `account_uid`       | `string`      | The Scaleway organization UUID.                                                                                | OTel `cloud.account.id`        |
| `project_uid`       | `string`      | The Scaleway project UUID.                                                                                     | No direct schema mapping.      |
| `type`              | `string`      | The instance type, for example `DEV1-S`.                                                                       | OTel `host.type`               |
| `tags`              | `[]string`    | The instance tags.                                                                                             | No direct schema mapping.      |
| `state_detail`      | `string`      | The instance lifecycle state.                                                                                  | No direct schema mapping.      |
| `public_ip`         | `string`      | The public IPv4 address.                                                                                       | No direct schema mapping.      |
| `public_ip_id`      | `string`      | The resource ID of the public IP.                                                                              | No direct schema mapping.      |
| `public_ip_dynamic` | `bool`        | True if the public IP is dynamic.                                                                              | No direct schema mapping.      |
| `private_ip`        | `string`      | The private IPv4 address.                                                                                      | No direct schema mapping.      |
| `ipv6_address`      | `string`      | The public IPv6 address.                                                                                       | No direct schema mapping.      |
| `ipv6_netmask`      | `string`      | The IPv6 prefix.                                                                                               | No direct schema mapping.      |
| `ipv6_gateway`      | `string`      | The IPv6 gateway.                                                                                              | No direct schema mapping.      |
| `zone`              | `string`      | The availability zone, for example `fr-par-1`.                                                                 | OTel `cloud.availability_zone` |
| `platform_id`       | `string`      | The identifier of the Scaleway hardware platform.                                                              | No direct schema mapping.      |
| `ssh_public_keys`   | `[]string`    | The SSH public keys of the account.                                                                            | No direct schema mapping.      |
| `volumes`           | `[]Volume`    | The attached volumes.                                                                                          | No direct schema mapping.      |
| `timezone`          | `string`      | The instance timezone from the IMDS, for example `Europe/Paris`.                                               | No direct schema mapping.      |
| `bootscript`        | `*Bootscript` | The old boot configuration. It is deprecated, and modern instances use local boot. Refer to the section below. | No direct schema mapping.      |

### Volume

| Field         | Type     | Description                                       |
| ------------- | -------- | ------------------------------------------------- |
| `id`          | `string` | The volume UUID.                                  |
| `name`        | `string` | The volume name.                                  |
| `volume_type` | `string` | The volume class, for example `b_ssd` or `l_ssd`. |
| `size`        | `int64`  | The volume size in bytes.                         |
| `export_uri`  | `string` | The NBD export URI. It is internal to Scaleway.   |

### Bootscript

| Field          | Type     | Description                                    |
| -------------- | -------- | ---------------------------------------------- |
| `id`           | `string` | The bootscript ID.                             |
| `title`        | `string` | The bootscript display name.                   |
| `architecture` | `string` | The target architecture, for example `x86_64`. |
| `kernel`       | `string` | The kernel download URL.                       |
| `initrd`       | `string` | The initrd download URL.                       |
| `bootcmdargs`  | `string` | The boot command-line arguments.               |
| `organization` | `string` | The organization that owns the bootscript.     |
| `public`       | `bool`   | True if the bootscript is public.              |

## Platform support

| Platform | Supported                       |
| -------- | ------------------------------- |
| Linux    | ✅                              |
| macOS    | ✅ (data only on a Scaleway VM) |
| Other    | ✅ (data only on a Scaleway VM) |

## Example output

```json
{
  "scaleway": {
    "id": "sc-abc",
    "name": "prod-1",
    "hostname": "prod-1",
    "type": "DEV1-S",
    "public_ip": "51.0.0.1",
    "private_ip": "10.64.0.1",
    "zone": "fr-par-1"
  }
}
```

## SDK usage

```go
import (
    "context"
    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("scaleway"))
facts, _ := g.Collect(context.Background())

if facts.Scaleway != nil {
    fmt.Println("running in", facts.Scaleway.Zone)
}
```

## Enable/disable

```bash
gohai collect --collector.scaleway      # enable (opt-in)
gohai collect --no-collector.scaleway   # disable (default)
gohai collect --category=cloud          # pulls this + all cloud collectors
```

## Dependencies

None. The detection uses `/proc/cmdline`, not DMI. Scaleway has no stable DMI
signature, but its kernel command line always contains `"scaleway"`.

## Data sources

1. **cmdline gate.** The collector reads `/proc/cmdline` and looks for the
   substring `"scaleway"`. The check is not case-sensitive. If the file or the
   substring is missing, the collector stops and returns `(nil, nil)`. This is
   the same as Ohai's `has_scaleway_cmdline?`.

2. **Endpoint.** The collector reads one JSON document from
   `http://169.254.42.42/conf?format=json`.

3. **User-Agent.** The User-Agent is `gohai`, the cloudmetadata default.

4. **Timeout.** The timeout is 6 seconds. This is the same as Ohai's
   `read_timeout` in `mixin/scaleway_metadata.rb`.

5. **Failures.** If the read fails (transport, a status that is not 2xx, or the
   body read), the collector returns `(nil, nil)`. Then `Facts.Scaleway` is
   empty and there is no error. Only malformed JSON in the response gives an
   error.

6. **Transformation.** The collector changes the response in these ways:

   - It moves the fields of the nested `public_ip`, `ipv6` and `location`
     objects to the top level of Info.
   - It changes `ssh_public_keys` from an array of `{key}` objects to a plain
     string array. It skips blank keys.
   - It changes the `volumes` map, which has an index as the key, to a slice.
   - It keeps `bootscript` as a typed substruct, if it is there. `bootscript` is
     deprecated on modern instances and can be null.
   - It copies `timezone` as a simple string.

   All fields that the collector reports are typed. There is no `Raw` field for
   untyped data.

The method is the same as Ohai's `Ohai::Mixin::ScalewayMetadata`. It uses the
same endpoint, the same single JSON GET and the same 6-second timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors.
