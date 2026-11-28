# Alibaba

> **Status:** Implemented ✅

## Description

The collector reads the Alibaba Cloud ECS instance metadata from the link-local
server at `http://100.100.100.200/2016-01-01/`. Alibaba uses `100.100.100.200`,
not the more usual `169.254.169.254` link-local range. If
`facts.Alibaba != nil`, the host is on Alibaba Cloud.

The collector runs only if the DMI `sys_vendor` contains `"Alibaba"`. This check
is the same as Ohai's `has_ali_dmi?`.

## Collected fields

| Field                             | Type                          | Description                                                                  | Schema mapping                 |
| --------------------------------- | ----------------------------- | ---------------------------------------------------------------------------- | ------------------------------ |
| `id`                              | `string`                      | The ECS instance ID.                                                         | OTel `cloud.resource_id`       |
| `name`                            | `string`                      | The instance display name.                                                   | OTel `host.name`               |
| `type`                            | `string`                      | The ECS instance type, for example `ecs.g6.large`.                           | OTel `host.type`               |
| `hostname`                        | `string`                      | The hostname.                                                                | OCSF `device.hostname`         |
| `image_id`                        | `string`                      | The source image ID.                                                         | OTel `host.image.id`           |
| `serial_number`                   | `string`                      | The instance serial.                                                         | No direct schema mapping.      |
| `network_type`                    | `string`                      | `vpc` or `classic`.                                                          | No direct schema mapping.      |
| `region`                          | `string`                      | The region, for example `cn-hangzhou`.                                       | OTel `cloud.region`            |
| `zone`                            | `string`                      | The availability zone, for example `cn-hangzhou-b`.                          | OTel `cloud.availability_zone` |
| `mac`                             | `string`                      | The MAC of the primary interface.                                            | OCSF `network_interface.mac`   |
| `private_ipv4`                    | `string`                      | The primary private IPv4.                                                    | No direct schema mapping.      |
| `public_ipv4`                     | `string`                      | The EIP (elastic IP), if attached.                                           | No direct schema mapping.      |
| `vpc_id`                          | `string`                      | The VPC ID.                                                                  | No direct schema mapping.      |
| `vpc_cidr_block`                  | `string`                      | The VPC CIDR.                                                                | No direct schema mapping.      |
| `vswitch_id`                      | `string`                      | The vSwitch ID (VPC subnet).                                                 | No direct schema mapping.      |
| `vswitch_cidr_block`              | `string`                      | The vSwitch CIDR.                                                            | No direct schema mapping.      |
| `dns_nameservers`                 | `[]string`                    | The configured DNS resolvers.                                                | No direct schema mapping.      |
| `ntp_servers`                     | `[]string`                    | The configured NTP servers.                                                  | No direct schema mapping.      |
| `max_bandwidth_ingress`           | `int64`                       | The maximum ingress bandwidth (bytes/sec).                                   | No direct schema mapping.      |
| `max_bandwidth_egress`            | `int64`                       | The maximum egress bandwidth (bytes/sec).                                    | No direct schema mapping.      |
| `ram_role_name`                   | `string`                      | The name of the attached RAM role.                                           | No direct schema mapping.      |
| `account_uid`                     | `string`                      | The Alibaba Cloud account that owns the instance.                            | No direct schema mapping.      |
| `source_address`                  | `string`                      | The URL of the package-manager mirror that Alibaba gives for the region.     | No direct schema mapping.      |
| `virtualization_solution`         | `string`                      | The hypervisor family, for example `ECS Virt`.                               | No direct schema mapping.      |
| `virtualization_solution_version` | `string`                      | The build version of the hypervisor.                                         | No direct schema mapping.      |
| `spot_termination_time`           | `string`                      | The ISO-8601 warning time for spot termination. Only spot instances have it. | No direct schema mapping.      |
| `network_interfaces`              | `map[string]NetworkInterface` | All ENIs, with the MAC as the key. Refer to NetworkInterface.                | OCSF `network_interface`       |
| `disks`                           | `map[string]Disk`             | The attached disks, with the serial number as the key. Refer to Disk.        | No direct schema mapping.      |
| `marketplace`                     | `*Marketplace`                | The billing data of a Marketplace image. Only Marketplace images have it.    | No direct schema mapping.      |
| `tags`                            | `map[string]string`           | The instance tags that the user defined.                                     | No direct schema mapping.      |

### NetworkInterface

| Field                     | Type       | Description                             |
| ------------------------- | ---------- | --------------------------------------- |
| `network_interface_id`    | `string`   | The ENI ID (`eni-...`).                 |
| `primary_ip_address`      | `string`   | The primary private IPv4.               |
| `private_ipv4s`           | `[]string` | All private IPv4 addresses on this ENI. |
| `ipv4_prefixes`           | `[]string` | The delegated IPv4 prefixes.            |
| `netmask`                 | `string`   | The subnet netmask of the primary IP.   |
| `gateway`                 | `string`   | The default gateway of this ENI.        |
| `vpc_id`                  | `string`   | The VPC ID.                             |
| `vpc_cidr_block`          | `string`   | The VPC IPv4 CIDR.                      |
| `vpc_ipv6_cidr_blocks`    | `[]string` | The VPC IPv6 CIDRs.                     |
| `vswitch_id`              | `string`   | The vSwitch (subnet) ID.                |
| `vswitch_cidr_block`      | `string`   | The vSwitch IPv4 CIDR.                  |
| `vswitch_ipv6_cidr_block` | `string`   | The vSwitch IPv6 CIDR.                  |
| `ipv6s`                   | `[]string` | All IPv6 addresses on this ENI.         |
| `ipv6_prefixes`           | `[]string` | The delegated IPv6 prefixes.            |
| `ipv6_gateway`            | `string`   | The IPv6 default gateway.               |

### Disk

| Field  | Type     | Description                           |
| ------ | -------- | ------------------------------------- |
| `id`   | `string` | The disk resource ID (`d-...`).       |
| `name` | `string` | The disk name that the operator gave. |

### Marketplace

| Field          | Type     | Description                                |
| -------------- | -------- | ------------------------------------------ |
| `product_code` | `string` | The Marketplace product code.              |
| `charge_type`  | `string` | The billing method, for example `PrePaid`. |

## Platform support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅                                     |
| macOS    | ✅ (useful only on an Alibaba ECS)     |
| Other    | ✅ (only meaningful on an Alibaba ECS) |

## Example output

```json
{
  "alibaba": {
    "id": "i-abc",
    "region": "cn-hangzhou",
    "zone": "cn-hangzhou-b",
    "type": "ecs.g6.large",
    "private_ipv4": "172.16.0.5",
    "public_ipv4": "47.1.2.3",
    "vpc_id": "vpc-1",
    "ram_role_name": "ecs-default"
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("alibaba"))
facts, _ := g.Collect(context.Background())

if facts.Alibaba != nil {
    fmt.Println(facts.Alibaba.Region, facts.Alibaba.Type)
}
```

## Enable/disable

```bash
gohai collect --collector.alibaba      # enable (opt-in)
gohai collect --no-collector.alibaba   # disable (default)
gohai collect --category=cloud         # pulls this + all cloud collectors
```

## Dependencies

`dmi`. Alibaba writes `"Alibaba Cloud"` as `sys_vendor`. If `dmi` did not run,
the check passes and the collector continues. If `dmi` ran and found no product
table, as in a container or on a host with no `/sys/class/dmi`, the host is not
on Alibaba Cloud, and the collector sends no request.

## Data sources

01. **DMI gate:** the collector continues only if `dmi.Product.VendorName`
    contains `"Alibaba"`. The dmi collector reads `/sys/class/dmi/id/sys_vendor`
    into `Product.VendorName`. Thus this check is the same as Ohai's
    `has_ali_dmi?`.
02. **Endpoint:** `http://100.100.100.200/2016-01-01/`. This link-local address
    is not standard. It is `100.100.100.200`, not `169.254.169.254`.
03. **Recursive walk:** the collector starts at `/` and gets each directory
    listing. A listing has one entry on each line. The collector goes into each
    entry that ends in `/`, and gets each leaf as a value. This algorithm is the
    same as Ohai's `fetch_metadata` in `mixin/alibaba_metadata.rb`. The walk
    goes at most 8 directory levels down, and sends at most 256 requests, so a
    server that lists a directory in itself cannot make it run without end. Ohai
    has no such limit.
04. **`/user-data` excluded:** the collector skips `/user-data`, at the root
    level only. Ohai also skips it. The cloud-init scripts in it can contain
    credentials. The walk also skips a listed name that has a character other
    than a letter, a digit, `_`, `.`, `:` or `-`, and the names `.` and `..`.
    Such a name would change the path or the query of the next request. Ohai
    does not check the names.
05. **Leaf parsing:** the collector keeps each leaf as its trimmed text, and
    reads the bandwidth fields as integers from that text. Ohai parses each leaf
    as JSON first. That turns a numeric leaf, such as `owner-account-id`, into a
    number, which loses digits above 2^53 and which a text field cannot keep.
06. **Key sanitization:** dashes and slashes in path segments become underscores
    in the map keys. The collector removes underscores at the end of a key. This
    is the same as Ohai's `sanitize_key`.
07. **Transformation:** the collector puts the walked tree into the typed
    `Info`. It flattens the identity, location and network fields from
    `meta-data/`. It also adds the nested sub-objects `instance/` (with
    `spot/termination-time`), `image/market-place/`, `tags/instance/`,
    `disks/<serial>/` and `network/interfaces/macs/<mac>/`. All reported fields
    are typed. There is no `Raw` field for other data.
08. **Deliberately excluded:** the walk does not read
    `meta-data/ram/security-credentials/` (a short-lived AccessKey and
    SecurityToken), so the credentials are never fetched or held in memory. The
    typed output also leaves out `public-keys/<keypair>/openssh-key`. Both are
    sensitive for security and have no value for an inventory.
09. **User-Agent:** `gohai` (the cloudmetadata default). Alibaba's metadata
    service filters some requests by User-Agent.
10. **Timeout:** 6 seconds for each request. This is the same as Ohai's
    `read_timeout` + `keep_alive_timeout`. The whole walk also has a deadline,
    `alibaba.WalkTimeout` (30 seconds). Without it, a server that answers the
    first listing and then hangs could hold the walk for 256 requests of 6
    seconds each, about 25 minutes. When the deadline ends, the collector
    returns the context error.
11. **Failure handling:** a path that the server does not answer, such as a 404,
    a 5xx or no connection, is left out: if it is the root listing, the
    collector returns `(nil, nil)`, as for a host that is not on Alibaba.
    Otherwise those fields stay empty in the typed output. Any other failure
    stops the walk and is the collector's error: a cancelled or expired context,
    a response that is more than `cloudmetadata.MaxBody`, or a walk that reaches
    its 256-request limit.

The method is the same as Ohai's `Ohai::Mixin::AlibabaMetadata`. These parts are
the same:

- the endpoint
- the recursive walk
- the key sanitization
- the `/user-data` removal
- the User-Agent pattern
- the 6s timeout

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors. Its transport is the `HTTP` of the
  sources, so a fake host can answer the metadata reads.
