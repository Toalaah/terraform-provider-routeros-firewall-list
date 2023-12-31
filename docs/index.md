---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "routeros-firewall-list Provider"
subcategory: ""
description: |-
  A provider for declaratively  managing firewall lists on RouterOS devices.
---

# routeros-firewall-list Provider

A provider for declaratively  managing firewall lists on RouterOS devices.

## Example Usage

```terraform
terraform {
  required_providers {
    routeros-firewall-list = {
      source = "toalaah/routeros-firewall-list"
    }
  }
}

provider "routeros-firewall-list" {
  hosturl        = "192.168.88.1" # env fallback: ROS_HOSTURL
  username       = "admin"        # env fallback: ROS_USERNAME
  password       = "password"     # env fallback: ROS_PASSWORD
  ca_certificate = "./ca.pem"     # env fallback: ROS_CA_CERTIFICATE
  insecure       = false          # env fallback: ROS_INSECURE
}

resource "routeros-firewall-list_rule_ordering" "rules" {
  rule_type = "filter"
  rules = [
    "*A",
    "*B",
    "*C",
    "*9",
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ca_certificate` (String) Path to the CA root certificate
- `hosturl` (String) Address of the host device. Do not specify the protocol or port, these are hard-coded to 'https' and '443' respectively
- `insecure` (Boolean) Whether to skip verifying the SSL certificate used by the API service
- `password` (String, Sensitive) Password to use for API authentication
- `username` (String) Username to use for API authentication
