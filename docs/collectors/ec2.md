# EC2

> **Status:** Implemented ✅

## Description

The collector reads the AWS EC2 instance metadata from the link-local server at
`http://169.254.169.254/`. By default, it uses IMDSv2, which needs a token. If
the collector cannot reach the token endpoint, it uses IMDSv1. The collector
**negotiates the API version** with the EC2 IMDS. It does not use a fixed
version. `facts.Ec2 != nil` is the signal that the host is on EC2.

Detection uses all of Ohai's signals that are not for Windows:

- `dmi.BIOS.Manufacturer` contains `"Amazon"` (the same as
  `has_ec2_amazon_dmi?`)
- `dmi.BIOS.Ver` contains `"amazon"` in lowercase (the same as
  `has_ec2_xen_dmi?`)
- `/sys/hypervisor/uuid` starts with `"ec2"` (the same as `has_ec2_xen_uuid?`)

If no signal is found, the collector does not send the HTTP request. Otherwise,
the collector reads:

- The `/meta-data/` tree. This is a selected list of paths, and
  `security-groups` and `local-ipv4s` as arrays split on newlines.
- The full subtree of each ENI under
  `/meta-data/network/interfaces/macs/<mac>/`.
- The IAM information, from `iam/info` only. The collector intentionally does
  not read `security-credentials`. Ohai also removes this subtree, because it
  holds secrets.
- The dynamic instance identity document (`accountId`, `region`,
  `availabilityZone`, `instanceId`).
- The raw user-data. It is base64-encoded when it is binary, and plain text when
  it is UTF-8.

## Collected fields

| Field                   | Type                          | Description                                                                                           | Schema mapping                 |
| ----------------------- | ----------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------ |
| `id`                    | `string`                      | The EC2 instance ID.                                                                                  | OTel `cloud.resource_id`       |
| `type`                  | `string`                      | The instance type, for example `t3.micro`.                                                            | OTel `host.type`               |
| `instance_life_cycle`   | `string`                      | `on-demand`, `spot` or `scheduled`.                                                                   | No direct schema mapping.      |
| `image_id`              | `string`                      | The ID of the boot AMI.                                                                               | OTel `host.image.id`           |
| `ami_launch_index`      | `string`                      | The launch index in the reservation.                                                                  | No direct schema mapping.      |
| `ami_manifest_path`     | `string`                      | The legacy manifest path.                                                                             | No direct schema mapping.      |
| `hostname`              | `string`                      | The hostname that EC2 gives.                                                                          | No direct schema mapping.      |
| `local_hostname`        | `string`                      | The internal DNS name.                                                                                | OCSF `device.hostname`         |
| `public_hostname`       | `string`                      | The public DNS name, if there is one.                                                                 | No direct schema mapping.      |
| `local_ipv4`            | `string`                      | The primary private IPv4 address.                                                                     | No direct schema mapping.      |
| `public_ipv4`           | `string`                      | The primary public IPv4 address.                                                                      | No direct schema mapping.      |
| `mac`                   | `string`                      | The MAC address of the primary interface.                                                             | OCSF `network_interface.mac`   |
| `security_groups`       | `[]string`                    | The security group names.                                                                             | No direct schema mapping.      |
| `region`                | `string`                      | The AWS region, for example `us-east-1`.                                                              | OTel `cloud.region`            |
| `zone`                  | `string`                      | The availability zone, for example `us-east-1a`.                                                      | OTel `cloud.availability_zone` |
| `account_uid`           | `string`                      | The AWS account ID, from the identity document.                                                       | OTel `cloud.account.id`        |
| `reservation_id`        | `string`                      | The reservation ID.                                                                                   | No direct schema mapping.      |
| `profile`               | `string`                      | The virtualization profile (`default-hvm`).                                                           | No direct schema mapping.      |
| `iam_info`              | `*IAMInstanceInfo`            | The attached IAM profile. Refer to the table below.                                                   | No direct schema mapping.      |
| `local_ipv4s`           | `[]string`                    | All private IPv4 addresses of the primary interface.                                                  | No direct schema mapping.      |
| `network_interfaces`    | `map[string]NetworkInterface` | The subtree of each ENI, with the MAC as the key. Refer to the table below.                           | OCSF `network_interface`       |
| `api_version`           | `string`                      | The IMDS API version that the collector negotiated.                                                   | No direct schema mapping.      |
| `user_data`             | `string`                      | The raw user-data. It is base64-encoded when it is binary.                                            | No direct schema mapping.      |
| `availability_zone_id`  | `string`                      | The AZ identifier, for example `use1-az1`.                                                            | No direct schema mapping.      |
| `group_name`            | `string`                      | The placement group name.                                                                             | No direct schema mapping.      |
| `host_id`               | `string`                      | The Dedicated Host ID. It is empty on shared hosts.                                                   | No direct schema mapping.      |
| `partition_number`      | `string`                      | The partition index, for partition placement.                                                         | No direct schema mapping.      |
| `kernel_id`             | `string`                      | The legacy paravirtual kernel ID. It is empty on HVM.                                                 | No direct schema mapping.      |
| `ramdisk_id`            | `string`                      | The legacy paravirtual ramdisk ID.                                                                    | No direct schema mapping.      |
| `instance_action`       | `string`                      | The pending instance action (`none`, `shutdown`, `stop`, `terminate`).                                | No direct schema mapping.      |
| `spot_instance_action`  | `string`                      | The spot lifecycle signal (`stop`, `terminate`, `hibernate`).                                         | No direct schema mapping.      |
| `spot_termination_time` | `string`                      | The time of the spot termination warning, in ISO-8601.                                                | No direct schema mapping.      |
| `services_domain`       | `string`                      | The domain of the AWS service endpoints. GovCloud and China use a domain that is not `amazonaws.com`. | No direct schema mapping.      |
| `cloud_partition`       | `string`                      | The AWS partition (`aws`, `aws-us-gov`, `aws-cn`).                                                    | No direct schema mapping.      |
| `product_codes`         | `[]string`                    | The Marketplace product codes of the AMI.                                                             | No direct schema mapping.      |
| `public_keys`           | `[]string`                    | The public keys in OpenSSH format that EC2 adds at launch.                                            | No direct schema mapping.      |
| `block_device_mapping`  | `map[string]string`           | The virtual disk name to the device path, for example `ami` to `/dev/xvda`.                           | No direct schema mapping.      |

### NetworkInterface

| Field                     | Type       | Description                             |
| ------------------------- | ---------- | --------------------------------------- |
| `device_number`           | `string`   | The ENI device-number index.            |
| `interface_id`            | `string`   | The ENI ID (`eni-xxx`).                 |
| `local_hostname`          | `string`   | The internal DNS name of this ENI.      |
| `local_ipv4s`             | `[]string` | All private IPv4 addresses of this ENI. |
| `mac`                     | `string`   | The MAC address.                        |
| `network_card_index`      | `string`   | The network card index.                 |
| `owner_id`                | `string`   | The account that owns the ENI.          |
| `public_hostname`         | `string`   | The public DNS name of this ENI.        |
| `public_ipv4s`            | `[]string` | All public IPv4 addresses of this ENI.  |
| `security_group_ids`      | `[]string` | The security group IDs.                 |
| `security_groups`         | `[]string` | The security group names.               |
| `subnet_id`               | `string`   | The subnet ID.                          |
| `subnet_ipv4_cidr_block`  | `string`   | The subnet IPv4 CIDR.                   |
| `subnet_ipv6_cidr_blocks` | `[]string` | The subnet IPv6 CIDRs.                  |
| `vpc_id`                  | `string`   | The VPC ID.                             |
| `vpc_ipv4_cidr_block`     | `string`   | The VPC IPv4 CIDR.                      |
| `vpc_ipv4_cidr_blocks`    | `[]string` | The VPC IPv4 CIDRs.                     |
| `vpc_ipv6_cidr_blocks`    | `[]string` | The VPC IPv6 CIDRs.                     |
| `ipv6s`                   | `[]string` | All IPv6 addresses of this ENI.         |

### IAMInstanceInfo

| Field                  | Type     | Description                               |
| ---------------------- | -------- | ----------------------------------------- |
| `code`                 | `string` | `Success` or an error code.               |
| `last_updated`         | `string` | The time of the last rotation.            |
| `instance_profile_arn` | `string` | The full ARN of the attached IAM profile. |
| `instance_profile_id`  | `string` | The IAM profile ID.                       |

The collector **intentionally does not read** `security-credentials` (the access
keys). Ohai also removes this subtree, because it holds secrets.

## Platform support

| Platform | Supported                           |
| -------- | ----------------------------------- |
| Linux    | ✅                                  |
| macOS    | ✅ (useful only on an EC2 instance) |
| Other    | ✅ (useful only on an EC2 instance) |

## Example output

```json
{
  "ec2": {
    "id": "i-abc",
    "type": "t3.micro",
    "region": "us-east-1",
    "zone": "us-east-1a",
    "local_ipv4": "10.0.0.5",
    "public_ipv4": "1.2.3.4",
    "account_uid": "123456789012",
    "iam_info": {
      "code": "Success",
      "instance_profile_arn": "arn:aws:iam::123456789012:instance-profile/web"
    }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("ec2"))
facts, _ := g.Collect(context.Background())

if facts.Ec2 != nil {
    fmt.Println(facts.Ec2.Region, facts.Ec2.AccountUID)
}
```

## Enable/disable

```bash
gohai collect --collector.ec2      # enable (opt-in)
gohai collect --no-collector.ec2   # disable (default)
gohai collect --category=cloud     # pulls this + all cloud collectors
```

## Dependencies

`dmi`. EC2 writes `"Amazon"` or `"Amazon EC2"` as `bios_vendor`. On Xen hosts,
`bios_version` contains `amazon`. The collector also checks
`/sys/hypervisor/uuid`, for Xen PV instances and older HVM instances on Xen. If
dmi did not run, the check fails open: the collector continues to the HTTP
request.

## Data sources

01. **Detection.** The collector continues if one of these signals is found:

    - `dmi.BIOS.Manufacturer` contains `"Amazon"` (Ohai's `has_ec2_amazon_dmi?`)
    - `dmi.BIOS.Ver` contains `"amazon"` in lowercase (Ohai's
      `has_ec2_xen_dmi?`). This finds HVM instances on Xen-based hypervisors.
    - `/sys/hypervisor/uuid` starts with `"ec2"` (Ohai's `has_ec2_xen_uuid?`)

02. **IMDSv2 token.** The collector sends `PUT /latest/api/token` with
    `X-aws-ec2-metadata-token-ttl-seconds: 60`. The response body is the token.
    If the response is 404, the collector uses IMDSv1 and sends no token header
    on the subsequent reads. Ohai does the same.

03. **API version negotiation.** `GET /` gives the versions that EC2 knows, one
    on each line. The collector compares this list with its
    `supportedAPIVersions` allowlist and selects the latest version that is in
    both. The allowlist is the same set as Ohai's `EC2_SUPPORTED_VERSIONS`. If
    the negotiation fails, the collector uses `"latest"`. EC2 uses `"latest"` as
    an alias for its newest supported version. The negotiation fails on a
    transport error, a 404, no common version, or an empty body. This is the
    same as Ohai's `best_api_version` handshake.

04. **Meta-data tree.** The collector sends a GET request for each path in a
    selected list of `/<version>/meta-data/*` paths. The list has the main
    identity and network fields. It also has:

    - The placement subpaths (`placement/availability-zone-id`,
      `placement/group-name`, `placement/host-id`,
      `placement/partition-number`).
    - The legacy paravirtual boot paths (`kernel-id`, `ramdisk-id`).
    - The lifecycle signals (`instance-action`, `spot/instance-action`,
      `spot/termination-time`).
    - `product-codes`.
    - The `services/{domain,partition}` endpoints. The collector needs them to
      tell GovCloud and China from the commercial partition.

    The collector splits `security-groups` and `local-ipv4s` on newlines into
    arrays. This is the same as Ohai's `EC2_ARRAY_VALUES`.

05. **ENI subtree.** The collector reads
    `/<version>/meta-data/network/interfaces/macs/` to list the ENIs. Then it
    reads the subdirectory of each MAC recursively, and fills the typed
    `NetworkInterface` (subnet, VPC, security groups, IP arrays). This is the
    same as Ohai's `fetch_dir_metadata` for `EC2_ARRAY_DIR`.

06. **Directory listings.** The collector reads two listings into typed
    collections. It does not store them as raw text:

    - `GET /<version>/meta-data/public-keys/` gives lines such as `0=my-key`.
      For each index, the collector reads `public-keys/<i>/openssh-key`. It adds
      the OpenSSH-format string to `public_keys[]`. If a line has no `=`
      delimiter, or if the read of one key fails, the collector continues.
    - `GET /<version>/meta-data/block-device-mapping/` lists the virtual names
      (`ami`, `root`, `ebs1` and others). The collector reads each leaf and
      stores it in `block_device_mapping[<name>] = <device-path>`. This is the
      same shape that Ohai gives under `block_device_mapping`.

07. **Security filter.** The collector removes
    `identity_credentials_ec2_security_credentials_ec2_instance` from all lists
    that it splits on newlines. Ohai also skips this item explicitly.

08. **IAM information.** The collector sends `GET /<version>/meta-data/iam/info`
    and parses the response as JSON. It intentionally does NOT read
    `iam/security-credentials/<role>/`, because that path contains short-lived
    AWS access keys. Ohai also removes these secrets.

09. **Identity document.** The collector sends
    `GET /<version>/dynamic/instance-identity/document`. The response is JSON
    with `accountId`, `region`, `availabilityZone` and `instanceId`. These
    values fill the top-level fields when the meta-data tree does not give them.

10. **User-data.** The collector sends `GET /<version>/user-data/`. It stores
    the data as plain text when it is valid UTF-8, and base64-encoded when it is
    binary. This is the same as Ohai's `Encoding::BINARY` check.

11. **User-Agent.** `gohai` (the cloudmetadata default).

12. **Timeout.** 10 seconds for each request. This is the same as Ohai's
    `read_timeout` and `keep_alive_timeout`.

13. **Failures.** If the first request (`/ami-id`) fails, the collector returns
    `(nil, nil)`. If a request for a subsequent path fails, the collector
    continues, and that field keeps its zero value.

The method is the same as Ohai's `Ohai::Mixin::Ec2Metadata`. These parts are the
same:

- the change from IMDSv2 to IMDSv1
- the version negotiation
- the recursive ENI read
- the security filters
- the 10 s timeout

gohai does not implement the Windows signal
(`Win32_ComputerSystemProduct.identifyingnumber`).

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors.
