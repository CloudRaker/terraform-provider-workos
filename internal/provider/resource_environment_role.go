// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &EnvironmentRoleResource{}
var _ resource.ResourceWithImportState = &EnvironmentRoleResource{}

func NewEnvironmentRoleResource() resource.Resource {
	return &EnvironmentRoleResource{}
}

// EnvironmentRoleResource defines the resource implementation.
type EnvironmentRoleResource struct {
	client *client.Client
}

// EnvironmentRoleResourceModel describes the resource data model.
type EnvironmentRoleResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Type             types.String `tfsdk:"type"`
	ResourceTypeSlug types.String `tfsdk:"resource_type_slug"`
	Permissions      types.Set    `tfsdk:"permissions"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *EnvironmentRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_role"
}

func (r *EnvironmentRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS environment-level role and its permissions.",
		MarkdownDescription: `
Manages a WorkOS environment-level role.

Environment roles are defined once per environment and apply to every organization in that
environment, providing a consistent set of roles throughout your application. Roles are
identified by their ` + "`slug`" + `. The ` + "`permissions`" + ` set is fully managed by Terraform — it is
replaced to match the configuration, and may reference system permission slugs that are not
themselves managed as ` + "`workos_permission`" + ` resources.

## Example Usage

` + "```hcl" + `
resource "workos_environment_role" "admin" {
  slug        = "admin"
  name        = "Admin"
  description = "Full administrative access"

  permissions = [
    "playbook:create",
    "playbook:start",
  ]
}
` + "```" + `

## Import

Environment roles can be imported using their slug:

` + "```shell" + `
terraform import workos_environment_role.example admin
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the environment role.",
				MarkdownDescription: "The unique identifier of the environment role.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier for the role. Must be unique within the environment.",
				MarkdownDescription: "The slug identifier for the role. Must be unique within the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the role.",
				MarkdownDescription: "The display name of the role.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A description of the role.",
				MarkdownDescription: "A description of the role.",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The type of the role (e.g. EnvironmentRole).",
				MarkdownDescription: "The type of the role (e.g. `EnvironmentRole`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_slug": schema.StringAttribute{
				Description:         "The resource type slug this role is scoped to.",
				MarkdownDescription: "The resource type slug this role is scoped to. Changing this value recreates the role.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.SetAttribute{
				Description:         "The set of permission slugs granted by this role. Fully managed by Terraform.",
				MarkdownDescription: "The set of permission slugs granted by this role. Fully managed by Terraform.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the role was created.",
				MarkdownDescription: "The timestamp when the role was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the role was last updated.",
				MarkdownDescription: "The timestamp when the role was last updated (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					useStateForUnknownIfConfigUnchanged{
						configAttributes: []path.Path{
							path.Root("name"),
							path.Root("description"),
							path.Root("resource_type_slug"),
							path.Root("permissions"),
						},
					},
				},
			},
		},
	}
}

func (r *EnvironmentRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *EnvironmentRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating environment role", map[string]any{
		"slug": plan.Slug.ValueString(),
		"name": plan.Name.ValueString(),
	})

	createReq := &client.EnvironmentRoleCreateRequest{
		Slug: plan.Slug.ValueString(),
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.ResourceTypeSlug.IsNull() && !plan.ResourceTypeSlug.IsUnknown() {
		createReq.ResourceTypeSlug = plan.ResourceTypeSlug.ValueString()
	}

	if _, err := r.client.CreateEnvironmentRole(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Environment Role",
			"Could not create environment role, unexpected error: "+err.Error(),
		)
		return
	}

	perms := r.permissionsFromModel(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(perms) > 0 {
		if err := r.client.SetEnvironmentRolePermissions(ctx, plan.Slug.ValueString(), perms); err != nil {
			resp.Diagnostics.AddError(
				"Error Setting Environment Role Permissions",
				"Created the role but could not set its permissions: "+err.Error(),
			)
			return
		}
	}

	role, err := r.client.GetEnvironmentRole(ctx, plan.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment Role After Create",
			"Could not read back the environment role: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.mapRoleToModel(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Created environment role", map[string]any{"id": role.ID, "slug": role.Slug})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetEnvironmentRole(ctx, state.Slug.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Environment role not found, removing from state", map[string]any{"slug": state.Slug.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Environment Role",
			"Could not read environment role "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.mapRoleToModel(ctx, role, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentRoleResourceModel
	var state EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadataChanged := !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description)
	permissionsChanged := !plan.Permissions.Equal(state.Permissions)

	// Nothing the API cares about changed — keep computed values from state.
	if !metadataChanged && !permissionsChanged {
		plan.ID = state.ID
		plan.Type = state.Type
		plan.ResourceTypeSlug = state.ResourceTypeSlug
		plan.CreatedAt = state.CreatedAt
		plan.UpdatedAt = state.UpdatedAt
		plan.Permissions = state.Permissions
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	slug := state.Slug.ValueString()

	if metadataChanged {
		updateReq := &client.EnvironmentRoleUpdateRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		}
		if _, err := r.client.UpdateEnvironmentRole(ctx, slug, updateReq); err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Environment Role",
				"Could not update environment role, unexpected error: "+err.Error(),
			)
			return
		}
	}

	if permissionsChanged {
		perms := r.permissionsFromModel(ctx, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetEnvironmentRolePermissions(ctx, slug, perms); err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Environment Role Permissions",
				"Could not update environment role permissions, unexpected error: "+err.Error(),
			)
			return
		}
	}

	role, err := r.client.GetEnvironmentRole(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment Role After Update",
			"Could not read back the environment role: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.mapRoleToModel(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updated environment role", map[string]any{"id": role.ID, "slug": role.Slug})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := state.Slug.ValueString()
	err := r.client.DeleteEnvironmentRole(ctx, slug)
	if err == nil {
		tflog.Info(ctx, "Deleted environment role", map[string]any{"slug": slug})
		return
	}

	if client.IsNotFound(err) {
		tflog.Info(ctx, "Environment role already deleted", map[string]any{"slug": slug})
		return
	}

	// Some WorkOS environments may not permit deleting environment roles via the API. Rather
	// than wedging `terraform destroy`, drop it from state with a warning so it can be removed
	// in the dashboard.
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusMethodNotAllowed {
		resp.Diagnostics.AddWarning(
			"Environment Role Not Deleted",
			"WorkOS did not allow deleting environment role "+slug+" via the API; it has been removed "+
				"from Terraform state only. Delete it in the WorkOS dashboard if it is no longer needed.",
		)
		return
	}

	resp.Diagnostics.AddError(
		"Error Deleting Environment Role",
		"Could not delete environment role, unexpected error: "+err.Error(),
	)
}

func (r *EnvironmentRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing environment role", map[string]any{"id": req.ID})
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), req.ID)...)
}

// mapRoleToModel writes the API role onto the model, normalizing optional/computed fields.
func (r *EnvironmentRoleResource) mapRoleToModel(ctx context.Context, role *client.EnvironmentRole, m *EnvironmentRoleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(role.ID)
	m.Slug = types.StringValue(role.Slug)
	m.Name = types.StringValue(role.Name)
	m.Description = types.StringValue(role.Description)
	m.Type = types.StringValue(role.Type)
	if role.ResourceTypeSlug != "" {
		m.ResourceTypeSlug = types.StringValue(role.ResourceTypeSlug)
	} else {
		m.ResourceTypeSlug = types.StringNull()
	}
	m.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	m.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	perms := role.Permissions
	if perms == nil {
		perms = []string{}
	}
	permsSet, d := types.SetValueFrom(ctx, types.StringType, perms)
	diags.Append(d...)
	m.Permissions = permsSet

	return diags
}

// permissionsFromModel extracts the configured permission slugs from the model.
func (r *EnvironmentRoleResource) permissionsFromModel(ctx context.Context, m EnvironmentRoleResourceModel, diags *diag.Diagnostics) []string {
	if m.Permissions.IsNull() || m.Permissions.IsUnknown() {
		return []string{}
	}
	out := make([]string, 0, len(m.Permissions.Elements()))
	diags.Append(m.Permissions.ElementsAs(ctx, &out, false)...)
	return out
}
