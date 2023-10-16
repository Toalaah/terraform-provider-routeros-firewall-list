# Terraform Provider RouterOS Firewall List

This is a simple provider which facilitates the management of ordering firewall
rules of a Mikrotik router or other device running RouterOS. It is intended to
be used alongside the
[terraform-routeros](https://github.com/terraform-routeros/terraform-provider-routeros)
provider, as on its own it's capabilities are limited. As such, the provider
options are designed to mimic those of the RouterOS provider.

Although the RouterOS provider exposes the option to "softly" order a
`filter`/`nat`/... rule by using the `place_before` attribute, I found this
functionality to be too lacking for my personal use case. In particular, this
made it challenging to create a list of firewall rules using `for_each`, as I
would run into cyclic reference issues. Furthermore, the ordering was only used
during initial resource creation, and would not be checked upon refreshing
state. This means that if firewall rules are re-ordered by an external source
Terraform would have no way of knowing that the rule should be moved back into
it's original position.

This provider aims to fulfill said use case by enforcing and checking on rule
orders, allowing Terraform to "lock" rules/chains to the user's desired order.

## Important Note On Usage

This provider makes use of raw console commands which are **only** available
via the new REST API. As a result, the legacy API (default ports 8728, 8729) is
**not** compatible with this provider.

Prior to usage, you must also configure the `https` REST API with an
appropriate TLS/SSL certificate. For instructions on how to do so, refer to the
[Mikrotik](https://help.mikrotik.com/docs/display/ROS/REST+API) documentation.
This [guide](https://www.medo64.com/2016/11/enabling-https-on-mikrotik/) (not
affiliated), also explains the process quite well.

## Example Usage

```terraform
terraform {
  required_providers {
    routeros-firewall-list = {
      source = "toalaah/routeros-firewall-list"
    }
    routeros = {
      source  = "terraform-routeros/routeros"
      version = "1.18.3"
    }
  }
}

# Refer to provider documentation for configuration options
provider "routeros-firewall-list" {
  hosturl        = "192.168.88.1" # env fallback: ROS_HOSTURL
  username       = "admin"        # env fallback: ROS_USERNAME
  password       = "password"     # env fallback: ROS_PASSWORD
  ca_certificate = "./ca.pem"     # env fallback: ROS_CA_CERTIFICATE
  insecure       = false          # env fallback: ROS_INSECURE
}

# Configure routeros provider for creating firewall rules
provider "routeros" {
  hosturl        = "192.168.88.1" # env fallback: ROS_HOSTURL
  username       = "admin"        # env fallback: ROS_USERNAME
  password       = "password"     # env fallback: ROS_PASSWORD
  ca_certificate = "./ca.pem"     # env fallback: ROS_CA_CERTIFICATE
  insecure       = false          # env fallback: ROS_INSECURE
}

locals {
  # Contrived example, meant to demonstrate ordering capabilities rather than
  # sensible firewall configuration
  rules = [
    { action = "accept", src_address = "192.168.88.10", comment = "rule 1" },
    { action = "accept", src_address = "192.168.88.20", comment = "rule 2" },
    { action = "accept", src_address = "192.168.88.30", comment = "rule 3" },
    { action = "accept", src_address = "192.168.88.40", comment = "rule 4" },
    { action = "accept", src_address = "192.168.88.50", comment = "rule 5" },
  ]

  rule_map = { for idx, rule in local.rules : idx => rule }
}

# Use terraform-routeros provider to provision firewall rules. These may be
# created out-of-order.
resource "routeros_ip_firewall_filter" "rules" {
  for_each = local.rule_map

  action      = each.value.action
  comment     = "(rule ${each.key}): ${each.value.comment}"
  chain       = "input"
  src_address = each.value.src_address
  dst_address = "192.168.88.100"
}

# Create explicit rule ordering to force chain order according to list above
# rather than creation time.
resource "routeros-firewall-list_rule_ordering" "tester" {
  rule_type = "filter"
  rules = [
    for i, _ in local.rule_map : routeros_ip_firewall_filter.rules[i].id
  ]
}
```

## Local Development

To hack on this provider locally, you can configure a development override by
editing your Terraform RC located at `~/.terraformrc`. Make sure to amend the
provider path according to where you cloned the repository.

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/toalaah/routeros-firewall-list" = "/path/to/repo"
  }
  direct {}
}
```

You may create a local debug build by running `make debug`.

## GPG Signatures

Releases are signed with `484ABDF7B593FA5DFAA1101924FC7AC66A59A433`

## Acknowledgments

The following projects were heavily referenced during the development of this
provider

- [terraform-routeros/terraform-provider-routeros](https://github.com/terraform-routeros/terraform-provider-routeros)
- [hashicorp/terraform-provider-hashicups](https://github.com/hashicorp/terraform-provider-hashicups)

## License

This project is released under the terms of the GPLv3 license.
