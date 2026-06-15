// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// EnvironmentRole represents a WorkOS environment-level role. Environment roles are defined
// once per environment and are available to every organization in that environment (unlike
// organization roles, which are scoped to a single organization).
type EnvironmentRole struct {
	ID               string    `json:"id"`
	Object           string    `json:"object"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Type             string    `json:"type"`
	Permissions      []string  `json:"permissions"`
	ResourceTypeSlug string    `json:"resource_type_slug,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EnvironmentRoleCreateRequest is the body for creating an environment role.
type EnvironmentRoleCreateRequest struct {
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	ResourceTypeSlug string `json:"resource_type_slug,omitempty"`
}

// EnvironmentRoleUpdateRequest is the body for updating an environment role's metadata.
type EnvironmentRoleUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// SetRolePermissionsRequest is the body for replacing a role's full permission set.
type SetRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// EnvironmentRoleListResponse is the response from listing environment roles.
type EnvironmentRoleListResponse struct {
	Data         []EnvironmentRole `json:"data"`
	ListMetadata ListMetadata      `json:"list_metadata"`
}

// CreateEnvironmentRole creates a new environment-level role.
func (c *Client) CreateEnvironmentRole(ctx context.Context, req *EnvironmentRoleCreateRequest) (*EnvironmentRole, error) {
	var role EnvironmentRole
	if err := c.Post(ctx, "/authorization/roles", req, &role); err != nil {
		return nil, fmt.Errorf("failed to create environment role: %w", err)
	}
	return &role, nil
}

// GetEnvironmentRole retrieves an environment role by slug.
func (c *Client) GetEnvironmentRole(ctx context.Context, slug string) (*EnvironmentRole, error) {
	var role EnvironmentRole
	if err := c.Get(ctx, fmt.Sprintf("/authorization/roles/%s", url.PathEscape(slug)), &role); err != nil {
		return nil, fmt.Errorf("failed to get environment role: %w", err)
	}
	return &role, nil
}

// UpdateEnvironmentRole updates an environment role's name and/or description.
func (c *Client) UpdateEnvironmentRole(ctx context.Context, slug string, req *EnvironmentRoleUpdateRequest) (*EnvironmentRole, error) {
	var role EnvironmentRole
	if err := c.Patch(ctx, fmt.Sprintf("/authorization/roles/%s", url.PathEscape(slug)), req, &role); err != nil {
		return nil, fmt.Errorf("failed to update environment role: %w", err)
	}
	return &role, nil
}

// SetEnvironmentRolePermissions replaces the full permission set on an environment role.
func (c *Client) SetEnvironmentRolePermissions(ctx context.Context, slug string, permissions []string) error {
	if permissions == nil {
		permissions = []string{}
	}
	req := &SetRolePermissionsRequest{Permissions: permissions}
	if err := c.Put(ctx, fmt.Sprintf("/authorization/roles/%s/permissions", url.PathEscape(slug)), req, nil); err != nil {
		return fmt.Errorf("failed to set environment role permissions: %w", err)
	}
	return nil
}

// DeleteEnvironmentRole deletes an environment role by slug.
func (c *Client) DeleteEnvironmentRole(ctx context.Context, slug string) error {
	if err := c.Delete(ctx, fmt.Sprintf("/authorization/roles/%s", url.PathEscape(slug))); err != nil {
		return fmt.Errorf("failed to delete environment role: %w", err)
	}
	return nil
}

// ListEnvironmentRoles lists all environment-level roles, following pagination.
func (c *Client) ListEnvironmentRoles(ctx context.Context) (*EnvironmentRoleListResponse, error) {
	var all EnvironmentRoleListResponse
	params := url.Values{}
	applyDefaultPagination(params)

	for {
		var page EnvironmentRoleListResponse
		if err := c.Get(ctx, pathWithQuery("/authorization/roles", params), &page); err != nil {
			return nil, fmt.Errorf("failed to list environment roles: %w", err)
		}

		all.Data = append(all.Data, page.Data...)
		all.ListMetadata = page.ListMetadata
		if page.ListMetadata.After == "" {
			break
		}
		params.Set("after", page.ListMetadata.After)
	}

	return &all, nil
}
