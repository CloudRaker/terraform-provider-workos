// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentRoleResource_Basic(t *testing.T) {
	suffix := time.Now().UnixNano()
	slug := fmt.Sprintf("tf-acc-env-role-%d", suffix)
	permA := fmt.Sprintf("tf-acc-perm-a-%d", suffix)
	permB := fmt.Sprintf("tf-acc-perm-b-%d", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing — role granting both permissions.
			{
				Config: testAccEnvironmentRoleResourceConfig(slug, permA, permB, "Test Role", "A test role", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_environment_role.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_environment_role.test", "name", "Test Role"),
					resource.TestCheckResourceAttr("workos_environment_role.test", "description", "A test role"),
					resource.TestCheckResourceAttr("workos_environment_role.test", "permissions.#", "2"),
					resource.TestCheckResourceAttrSet("workos_environment_role.test", "id"),
					resource.TestCheckResourceAttrSet("workos_environment_role.test", "type"),
					resource.TestCheckResourceAttrSet("workos_environment_role.test", "created_at"),
					resource.TestCheckResourceAttrSet("workos_environment_role.test", "updated_at"),
				),
			},
			// ImportState testing — env roles import by slug.
			{
				ResourceName:      "workos_environment_role.test",
				ImportState:       true,
				ImportStateId:     slug,
				ImportStateVerify: true,
			},
			// Update testing — change metadata and drop a permission.
			{
				Config: testAccEnvironmentRoleResourceConfig(slug, permA, permB, "Updated Role", "An updated test role", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_environment_role.test", "name", "Updated Role"),
					resource.TestCheckResourceAttr("workos_environment_role.test", "description", "An updated test role"),
					resource.TestCheckResourceAttr("workos_environment_role.test", "permissions.#", "1"),
				),
			},
		},
	})
}

func TestAccEnvironmentRoleResource_NoPermissions(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-env-role-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentRoleResourceConfigNoPermissions(slug, "Test Role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_environment_role.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_environment_role.test", "name", "Test Role"),
					resource.TestCheckResourceAttr("workos_environment_role.test", "permissions.#", "0"),
					resource.TestCheckResourceAttrSet("workos_environment_role.test", "id"),
				),
			},
		},
	})
}

// testAccEnvironmentRoleResourceConfig builds a role that grants permA, and optionally permB.
func testAccEnvironmentRoleResourceConfig(slug, permA, permB, name, description string, withBoth bool) string {
	perms := "[workos_permission.a.slug]"
	if withBoth {
		perms = "[workos_permission.a.slug, workos_permission.b.slug]"
	}

	return fmt.Sprintf(`
resource "workos_permission" "a" {
  slug = %[2]q
  name = "Perm A"
}

resource "workos_permission" "b" {
  slug = %[3]q
  name = "Perm B"
}

resource "workos_environment_role" "test" {
  slug        = %[1]q
  name        = %[4]q
  description = %[5]q
  permissions = %[6]s
}
`, slug, permA, permB, name, description, perms)
}

func testAccEnvironmentRoleResourceConfigNoPermissions(slug, name string) string {
	return fmt.Sprintf(`
resource "workos_environment_role" "test" {
  slug = %[1]q
  name = %[2]q
}
`, slug, name)
}
