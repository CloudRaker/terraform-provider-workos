resource "workos_environment_role" "admin" {
  slug        = "admin"
  name        = "Admin"
  description = "Full administrative access"

  permissions = [
    "playbook:create",
    "playbook:start",
    "playbook-run:read",
  ]
}
