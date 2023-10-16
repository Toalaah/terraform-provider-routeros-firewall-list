# Enforces specified order for firewall rules in "filter" category
resource "routeros-firewall-list_rule_ordering" "rules" {
  rule_type = "filter"
  rules = [
    "*A",
    "*B",
    "*C",
    "*9",
  ]
}
