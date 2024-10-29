package provider

import (
	"context"
	"errors"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketPolicyResource{}
var _ resource.ResourceWithImportState = &BucketPolicyResource{}

func NewBucketPolicyResource() resource.Resource {
	return &BucketPolicyResource{}
}

// BucketPolicyResource defines the resource implementation.
type BucketPolicyResource struct {
	client *s3.Client
}

// BucketPolicyResourceModel describes the resource data model.
type BucketPolicyResourceModel struct {
	Bucket types.String `tfsdk:"bucket"`
	Policy types.String `tfsdk:"policy"`
}

func (r *BucketPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_policy"
}

func (r *BucketPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A bucket policy resource that represents a bucket policy in an object storage service.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The policy to attach to the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BucketPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = getClientFromProviderData(req.ProviderData)
}

func (r *BucketPolicyResource) put(ctx context.Context, data *BucketPolicyResourceModel) (diags diag.Diagnostics) {
	_, err := r.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: data.Bucket.ValueStringPointer(),
		Policy: data.Policy.ValueStringPointer(),
	})
	if err != nil {
		diags.AddError("Unable to put bucket policy", err.Error())
	}
	return
}

func (r *BucketPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	output, err := r.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: data.Bucket.ValueStringPointer()})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read bucket policy", err.Error())
		return
	}

	data.Policy = types.StringValue(*output.Policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	passthroughUpdate[BucketPolicyResourceModel](ctx, req, resp)
}

func (r *BucketPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{Bucket: data.Bucket.ValueStringPointer()})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete bucket policy", err.Error())
	}
}

func (r *BucketPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
