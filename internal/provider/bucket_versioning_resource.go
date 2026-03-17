package provider

import (
	"context"
	"errors"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketVersioningResource{}
var _ resource.ResourceWithImportState = &BucketVersioningResource{}

func NewBucketVersioningResource() resource.Resource {
	return &BucketVersioningResource{}
}

// BucketVersioningResource defines the resource implementation.
type BucketVersioningResource struct {
	client *s3.Client
}

// BucketVersioningResourceModel describes the resource data model.
type BucketVersioningResourceModel struct {
	Bucket                  types.String `tfsdk:"bucket"`
	VersioningConfiguration types.Object `tfsdk:"versioning_configuration"`
}

type VersioningConfiguration struct {
	Status types.String `tfsdk:"status"`
}

func (r *BucketVersioningResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_versioning"
}

func (r *BucketVersioningResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A bucket versioning resource. Note that there can only be one versioning configuration per bucket. Deleting this resource will set versioning to suspended.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket for which to configure versioning.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"versioning_configuration": schema.SingleNestedBlock{
				MarkdownDescription: "A versioning configuration to apply to the bucket.",
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The versioning status of the bucket.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(s3_types.BucketVersioningStatusEnabled),
								string(s3_types.BucketVersioningStatusSuspended),
							),
						},
					},
				},
			},
		},
	}
}

func (r *BucketVersioningResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = getClientFromProviderData(req.ProviderData)
}

func setVersioningConfigurationValues(ctx context.Context, data *BucketVersioningResourceModel, output *s3.GetBucketVersioningOutput) (diags diag.Diagnostics) {
	var d diag.Diagnostics

	versioningData := VersioningConfiguration{}
	versioningData.Status = types.StringValue(string(output.Status))

	data.VersioningConfiguration, d = types.ObjectValueFrom(ctx, data.VersioningConfiguration.AttributeTypes(ctx), versioningData)
	diags.Append(d...)
	return
}

func (r *BucketVersioningResource) put(ctx context.Context, data *BucketVersioningResourceModel) (diags diag.Diagnostics) {
	versioningData := VersioningConfiguration{}
	diags.Append(data.VersioningConfiguration.As(ctx, &versioningData, basetypes.ObjectAsOptions{})...)

	_, err := r.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: data.Bucket.ValueStringPointer(),
		VersioningConfiguration: &s3_types.VersioningConfiguration{
			Status: s3_types.BucketVersioningStatus(versioningData.Status.ValueString()),
		},
	})
	if err != nil {
		diags.AddError("Unable to create bucket versioning configuration", err.Error())
	}
	return
}

func (r *BucketVersioningResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketVersioningResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketVersioningResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketVersioningResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	output, err := r.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: data.Bucket.ValueStringPointer(),
	})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read bucket versioning configuration", err.Error())
		return
	}

	resp.Diagnostics.Append(setVersioningConfigurationValues(ctx, &data, output)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketVersioningResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketVersioningResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketVersioningResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketVersioningResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: data.Bucket.ValueStringPointer(),
		VersioningConfiguration: &s3_types.VersioningConfiguration{
			Status: s3_types.BucketVersioningStatusSuspended,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete bucket versioning configuration", err.Error())
	}
}

func (r *BucketVersioningResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
