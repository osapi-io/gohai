# OpenStack

> **Status:** Implemented ✅

## Description

The collector reads the OpenStack (Nova) instance metadata from the link-local
server at `http://169.254.169.254/`. OpenStack gives two sources on purpose. One
is an EC2-compatible metadata service under `/latest/meta-data/...`. The other
is a Nova document with more data, at `/openstack/latest/meta_data.json`. gohai
reads both. If `facts.OpenStack != nil`, the host is an OpenStack instance.

The collector does the detection only if the DMI `product.name` contains
`"OpenStack"`. Ohai's plugin uses the openstack signal of the virtualization
plugin, and that signal reads DMI. gohai reads DMI directly, with the same
result.

## Collected fields

| Field             | Type                | Description                                                                                                                                      | Schema mapping                 |
| ----------------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------ |
| `id`              | `string`            | The instance ID from the EC2-compatible tree.                                                                                                    | OTel `cloud.resource_id`       |
| `type`            | `string`            | The instance flavor, for example `m1.small`.                                                                                                     | OTel `host.type`               |
| `hostname`        | `string`            | The hostname.                                                                                                                                    | OCSF `device.hostname`         |
| `local_hostname`  | `string`            | The internal DNS name.                                                                                                                           | No direct schema mapping.      |
| `public_hostname` | `string`            | The public DNS name.                                                                                                                             | No direct schema mapping.      |
| `zone`            | `string`            | The Nova availability zone.                                                                                                                      | OTel `cloud.availability_zone` |
| `local_ipv4`      | `string`            | The primary private IPv4 address.                                                                                                                | No direct schema mapping.      |
| `public_ipv4`     | `string`            | The primary public IPv4 address.                                                                                                                 | No direct schema mapping.      |
| `security_groups` | `[]string`          | The attached security groups.                                                                                                                    | No direct schema mapping.      |
| `image_id`        | `string`            | The AMI ID from the EC2-compatible tree.                                                                                                         | OTel `host.image.id`           |
| `kernel_id`       | `string`            | The boot kernel image.                                                                                                                           | No direct schema mapping.      |
| `ramdisk_id`      | `string`            | The boot ramdisk image.                                                                                                                          | No direct schema mapping.      |
| `reservation_id`  | `string`            | The reservation ID from the EC2-compatible tree.                                                                                                 | No direct schema mapping.      |
| `name`            | `string`            | The Nova `name` field.                                                                                                                           | OTel `host.name`               |
| `project_uid`     | `string`            | The Nova project ID.                                                                                                                             | OTel `cloud.account.id`        |
| `uuid`            | `string`            | The Nova UUID.                                                                                                                                   | No direct schema mapping.      |
| `public_keys`     | `map[string]string` | A map of the SSH keys, from name to key.                                                                                                         | No direct schema mapping.      |
| `launch_index`    | `int`               | The Nova launch_index. This is the position of the instance in a launch request for many instances.                                              | No direct schema mapping.      |
| `devices`         | `[]Device`          | The list of attached block devices, from meta_data.json: type, bus, serial, path, address, tags.                                                 | No direct schema mapping.      |
| `provider`        | `string`            | `"openstack"` or `"dreamhost"`. Old Dreamhost OpenStack images had a `dhc-user` account. The value is the same as Ohai's `openstack[:provider]`. | No direct schema mapping.      |

The collector does not read three fields of meta_data.json. `network_info` holds
Neutron data that is different for each deployment. `random_seed` is sensitive
and has no inventory value. `meta` holds the key-value pairs that the user sets,
which can hold secrets.

## Platform support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (data only on an OpenStack VM)       |
| Other    | ✅ (only meaningful on an OpenStack VM) |

## Example output

```json
{
  "openstack": {
    "id": "i-abc",
    "type": "m1.small",
    "zone": "nova",
    "local_ipv4": "10.0.0.5",
    "uuid": "uuid-xxx",
    "project_uid": "proj-1"
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("openstack"))
facts, _ := g.Collect(context.Background())

if facts.OpenStack != nil {
    fmt.Println(facts.OpenStack.ProjectUID, facts.OpenStack.Zone)
}
```

## Enable/disable

```bash
gohai collect --collector.openstack      # enable (opt-in)
gohai collect --no-collector.openstack   # disable (default)
gohai collect --category=cloud           # pulls this + all cloud collectors
```

## Dependencies

`dmi`. OpenStack writes `"OpenStack Nova"` or a similar value as `product_name`.
If `dmi` did not run, the collector fails open and continues. A host whose DMI
has no product table, such as a container, is not OpenStack, and the collector
does not probe it.

## Data sources

1. **DMI gate.** The collector continues only if `dmi.Product.Name` contains
   `"OpenStack"`. This is the same as the gate of Ohai's virtualization plugin,
   which also reads DMI.
2. **EC2-compatible tree.** The collector starts at `/latest/meta-data/` and
   reads the directory listings recursively. An entry that ends in `/` is a
   subdirectory, and the collector reads it too. Other entries are leaves. This
   is the same as Ohai's `fetch_metadata` method for the EC2-compatible tree.
   This tree gives the identity, placement and network fields: `id`, `type`,
   `image_id`, `kernel_id`, `ramdisk_id`, `reservation_id`, `local_ipv4`,
   `public_ipv4`, `zone`, `security_groups` and others.
3. **Nova endpoint.** The collector reads
   `GET /openstack/latest/meta_data.json`, the Nova document with more data.
   This document gives `uuid`, `name`, `project_uid`, `hostname`,
   `launch_index`, `public_keys`, and the `devices[]` list of attached block
   devices (type, bus, serial, path, address, tags). If the EC2-compatible tree
   did not give `zone` or `hostname`, this document gives them. The collector
   does not read `network_info` and `random_seed`, on purpose. `network_info` is
   Neutron data that is different for each deployment. `random_seed` is
   sensitive and has no inventory value.
4. **Provider field.** The collector reads `/etc/passwd` for a `dhc-user` entry.
   If the entry is there, the value is `"dreamhost"`. If not, the value is
   `"openstack"`. This is the same as Ohai's `openstack_provider`.
5. **User-Agent.** The User-Agent is `gohai`, the cloudmetadata default.
6. **Timeout.** The timeout is 2 seconds. This is Ohai's default. Ohai lets you
   change it, but gohai uses the cloudmetadata default.
7. **Failures.** A document, directory or value that gets no answer or a 404 is
   not there, and is skipped. If neither the EC2-compatible tree nor the Nova
   document is there, the collector returns `(nil, nil)`. One source alone is
   sufficient to give an Info with data. When `dmi` did not run, so nothing has
   said the host is on OpenStack, any 4xx is also "not there", because it comes
   from another service. Any other failure is an error: an HTTP 5xx, a 4xx on
   a host that DMI says is on OpenStack, a Nova document that isn't valid JSON,
   or a cancelled context.

The method is the same as the recursive read of Ohai's
`Ohai::Mixin::Ec2Metadata`, which the openstack plugin also uses. Like Ohai,
gohai also reads the Nova document and finds the provider (dreamhost).

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors. Its transport is the `HTTP` of the
  sources, so a fake host can answer the metadata reads.
