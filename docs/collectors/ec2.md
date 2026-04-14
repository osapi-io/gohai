# EC2

> **Status:** Implemented ✅

## Description

Collects AWS EC2 instance metadata from the link-local server at
`http://169.254.169.254/`. Uses IMDSv2 (token-authenticated) by default and
falls back to IMDSv1 when the token endpoint isn't reachable. The collector
**negotiates the API version** with EC2's IMDS rather than hardcoding one.
`facts.Ec2 != nil` is the detection signal.

Detection runs Ohai's full non-Windows signal chain:

- `dmi.BIOS.Vendor` contains `"Amazon"` (matches `has_ec2_amazon_dmi?`)
- `dmi.BIOS.Version` contains `"amazon"` lowercase (matches `has_ec2_xen_dmi?`)
- `/sys/hypervisor/uuid` starts with `"ec2"` (matches `has_ec2_xen_uuid?`)

If none match, the HTTP probe is skipped. The collector then fetches:

- The `/meta-data/` tree (curated paths + newline-split arrays for
  `security-groups` and `local-ipv4s`)
- The full per-ENI subtree under `/meta-data/network/interfaces/macs/<mac>/`
- IAM info (`iam/info` only — `security-credentials` deliberately dropped to
  match Ohai's scrub of the secrets sub-tree)
- The dynamic instance identity document (`accountId`, `region`,
  `availabilityZone`, `instanceId`)
- Raw user-data (base64-encoded when binary, plaintext when UTF-8)

## Collected Fields

| Field                 | Type                          | Description                                          | Schema mapping                 |
| --------------------- | ----------------------------- | ---------------------------------------------------- | ------------------------------ |
| `instance_id`         | `string`                      | EC2 instance ID.                                     | OTel `cloud.resource_id`       |
| `instance_type`       | `string`                      | Instance type (e.g. `t3.micro`).                     | OTel `host.type`               |
| `instance_life_cycle` | `string`                      | `on-demand` / `spot` / `scheduled`.                  | No direct schema mapping.      |
| `ami_id`              | `string`                      | Boot AMI ID.                                         | OTel `host.image.id`           |
| `ami_launch_index`    | `string`                      | Launch index within the reservation.                 | No direct schema mapping.      |
| `ami_manifest_path`   | `string`                      | Legacy manifest path.                                | No direct schema mapping.      |
| `hostname`            | `string`                      | EC2-supplied hostname.                               | No direct schema mapping.      |
| `local_hostname`      | `string`                      | Internal DNS name.                                   | OCSF `device.hostname`         |
| `public_hostname`     | `string`                      | Public DNS name (if assigned).                       | No direct schema mapping.      |
| `local_ipv4`          | `string`                      | Primary private IPv4.                                | No direct schema mapping.      |
| `public_ipv4`         | `string`                      | Primary public IPv4.                                 | No direct schema mapping.      |
| `mac`                 | `string`                      | Primary interface MAC.                               | OCSF `network_interface.mac`   |
| `security_groups`     | `[]string`                    | Security group names.                                | No direct schema mapping.      |
| `region`              | `string`                      | AWS region (e.g. `us-east-1`).                       | OTel `cloud.region`            |
| `availability_zone`   | `string`                      | Availability zone (e.g. `us-east-1a`).               | OTel `cloud.availability_zone` |
| `account_id`          | `string`                      | AWS account ID (from identity document).             | OTel `cloud.account.id`        |
| `reservation_id`      | `string`                      | Reservation ID.                                      | No direct schema mapping.      |
| `profile`             | `string`                      | Virtualization profile (`default-hvm`).              | No direct schema mapping.      |
| `iam_info`            | `*IAMInstanceInfo`            | Attached IAM profile — see below.                    | No direct schema mapping.      |
| `local_ipv4s`         | `[]string`                    | All private IPv4s assigned to the primary interface. | No direct schema mapping.      |
| `network_interfaces`  | `map[string]NetworkInterface` | Per-ENI subtree keyed by MAC — see below.            | OCSF `network_interface`       |
| `api_version`         | `string`                      | The IMDS API version negotiated at collection time.  | No direct schema mapping.      |
| `user_data`           | `string`                      | Raw user-data; base64-encoded when binary.           | No direct schema mapping.      |

### NetworkInterface

| Field                     | Type       | Description                     |
| ------------------------- | ---------- | ------------------------------- |
| `device_number`           | `string`   | ENI device-number index.        |
| `interface_id`            | `string`   | ENI ID (`eni-xxx`).             |
| `local_hostname`          | `string`   | Internal DNS for this ENI.      |
| `local_ipv4s`             | `[]string` | All private IPv4s on this ENI.  |
| `mac`                     | `string`   | MAC address.                    |
| `network_card_index`      | `string`   | Network card index.             |
| `owner_id`                | `string`   | Account that owns the ENI.      |
| `public_hostname`         | `string`   | Public DNS for this ENI.        |
| `public_ipv4s`            | `[]string` | All public IPv4s on this ENI.   |
| `security_group_ids`      | `[]string` | Security-group IDs.             |
| `security_groups`         | `[]string` | Security-group names.           |
| `subnet_id`               | `string`   | Subnet ID.                      |
| `subnet_ipv4_cidr_block`  | `string`   | Subnet IPv4 CIDR.               |
| `subnet_ipv6_cidr_blocks` | `[]string` | Subnet IPv6 CIDRs.              |
| `vpc_id`                  | `string`   | VPC ID.                         |
| `vpc_ipv4_cidr_block`     | `string`   | VPC IPv4 CIDR.                  |
| `vpc_ipv4_cidr_blocks`    | `[]string` | VPC IPv4 CIDRs.                 |
| `vpc_ipv6_cidr_blocks`    | `[]string` | VPC IPv6 CIDRs.                 |
| `ipv6s`                   | `[]string` | All IPv6 addresses on this ENI. |

### IAMInstanceInfo

| Field                  | Type     | Description                           |
| ---------------------- | -------- | ------------------------------------- |
| `code`                 | `string` | `Success` or error code.              |
| `last_updated`         | `string` | Last-rotated timestamp.               |
| `instance_profile_arn` | `string` | Full ARN of the attached IAM profile. |
| `instance_profile_id`  | `string` | IAM profile ID.                       |

`security-credentials` (access keys) is **deliberately dropped** — matches
Ohai's explicit scrub of the secrets-bearing sub-tree.

## Platform Support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (only meaningful on an EC2 instance) |
| Other    | ✅ (only meaningful on an EC2 instance) |

## Example Output

```json
{
  "ec2": {
    "instance_id": "i-abc",
    "instance_type": "t3.micro",
    "region": "us-east-1",
    "availability_zone": "us-east-1a",
    "local_ipv4": "10.0.0.5",
    "public_ipv4": "1.2.3.4",
    "account_id": "123456789012",
    "iam_info": {
      "code": "Success",
      "instance_profile_arn": "arn:aws:iam::123456789012:instance-profile/web"
    }
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("ec2"))
facts, _ := g.Collect(context.Background())

if facts.Ec2 != nil {
    fmt.Println(facts.Ec2.Region, facts.Ec2.AccountID)
}
```

## Enable/Disable

```bash
gohai --collector.ec2      # enable (opt-in)
gohai --no-collector.ec2   # disable (default)
gohai --category=cloud     # pulls this + all cloud collectors
```

## Dependencies

`dmi`. EC2 writes `"Amazon"` / `"Amazon EC2"` as `bios_vendor` and on Xen hosts
`amazon` shows up in `bios_version`. The `/sys/hypervisor/uuid` check runs in
addition for Xen PV / older HVM-on-Xen instances. Fails open when dmi wasn't
run.

## Data Sources

1. **Detection gate:** any of the following triggers detection:
   - `dmi.BIOS.Vendor` contains `"Amazon"` (Ohai's `has_ec2_amazon_dmi?`)
   - `dmi.BIOS.Version` contains `"amazon"` lowercase (Ohai's `has_ec2_xen_dmi?`
     — catches HVM instances on Xen-based hypervisors)
   - `/sys/hypervisor/uuid` starts with `"ec2"` (Ohai's `has_ec2_xen_uuid?`)
2. **IMDSv2 token:** `PUT /latest/api/token` with
   `X-aws-ec2-metadata-token-ttl-seconds: 60`. The response body is the token.
   404 → IMDSv1 fallback (no token header on subsequent reads), matching Ohai's
   behavior.
3. **API version negotiation:** `GET /` lists EC2's known versions
   (newline-separated). The collector intersects this with its
   `supportedAPIVersions` allowlist (the same set Ohai's
   `EC2_SUPPORTED_VERSIONS` carries) and picks the latest match. Any negotiation
   failure (transport error, 404, no intersection, empty body) falls back to
   `"latest"` — which EC2 aliases to its newest supported version. Matches
   Ohai's `best_api_version` handshake.
4. **Meta-data tree:** GETs against a curated list of `/<version>/meta-data/*`
   paths. `security-groups` and `local-ipv4s` are split on newlines into arrays
   (matches Ohai's `EC2_ARRAY_VALUES`).
5. **Per-ENI subtree:** the collector walks
   `/<version>/meta-data/network/interfaces/macs/` to enumerate ENIs, then
   recurses into each MAC's subdirectory to populate the typed
   `NetworkInterface` (subnet, vpc, security groups, IP arrays). Matches Ohai's
   `fetch_dir_metadata` for `EC2_ARRAY_DIR`.
6. **Security scrub:**
   `identity_credentials_ec2_security_credentials_ec2_instance` is dropped from
   any newline-split list (mirror's Ohai's explicit skip).
7. **IAM info:** `GET /<version>/meta-data/iam/info` — parsed as JSON. We
   deliberately do NOT fetch `iam/security-credentials/<role>/` because it
   contains short-lived AWS access keys — matches Ohai's scrub of secrets.
8. **Identity document:** `GET /<version>/dynamic/instance-identity/document` —
   JSON with `accountId`, `region`, `availabilityZone`, `instanceId`. Fills the
   top-level fields when the meta-data tree doesn't supply them.
9. **User-data:** `GET /<version>/user-data/` — stored plaintext when valid
   UTF-8, base64-encoded when binary. Matches Ohai's `Encoding::BINARY` check.
10. **User-Agent:** `gohai` (the cloudmetadata default).
11. **Timeout:** 10 seconds per request, matching Ohai's `read_timeout` and
    `keep_alive_timeout`.
12. **Failure handling:** first-probe (`/ami-id`) failure returns `(nil, nil)`.
    Per-path failures on subsequent paths are tolerated and leave their field
    zero-valued.

Mirrors Ohai's `Ohai::Mixin::Ec2Metadata` methodology — same IMDSv2→IMDSv1
fallback, same version negotiation, same recursive ENI walk, same security
scrubs, same 10s timeout. Windows-specific signal
(`Win32_ComputerSystemProduct.identifyingnumber`) is not implemented.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
