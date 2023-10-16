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
