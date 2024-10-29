package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

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
var _ resource.Resource = &ObjectResource{}
var _ resource.ResourceWithImportState = &ObjectResource{}

func NewObjectResource() resource.Resource {
	return &ObjectResource{}
}

// ObjectResource defines the resource implementation.
type ObjectResource struct {
	client *s3.Client
}

// ObjectResourceModel describes the resource data model.
type ObjectResourceModel struct {
	Bucket  types.String `tfsdk:"bucket"`
	Id      types.String `tfsdk:"id"`
	Key     types.String `tfsdk:"key"`
	Content types.String `tfsdk:"content"`
	URL     types.String `tfsdk:"url"`
}

func (r *ObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *ObjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A object resource that represents an object stored in a bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket where to store the object.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The id of the object. The id is in `{bucket}/{key}` format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The key of the object.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The content of the object.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the object.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ObjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = getClientFromProviderData(req.ProviderData)
}

func (r *ObjectResource) put(ctx context.Context, data *ObjectResourceModel) (diags diag.Diagnostics) {
	body := strings.NewReader(data.Content.ValueString())
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: data.Bucket.ValueStringPointer(),
		Key:    data.Key.ValueStringPointer(),
		Body:   body,
	})
	if err != nil {
		diags.AddError("Unable to create object", err.Error())
	}
	return
}

func buildURL(endpoint, bucket, key string) string {
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	return fmt.Sprintf("%s%s/%s", endpoint, bucket, key)
}

func (r *ObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s/%s", data.Bucket.ValueString(), data.Key.ValueString()))
	data.URL = types.StringValue(buildURL(*r.client.Options().BaseEndpoint, data.Bucket.ValueString(), data.Key.ValueString()))
	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func parseId(data *ObjectResourceModel) (err error) {
	id := data.Id.ValueString()
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("invalid id format: %s", id)
		return
	}
	data.Bucket = types.StringValue(parts[0])
	data.Key = types.StringValue(parts[1])
	return
}

func (r *ObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := parseId(&data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to parse object id", err.Error())
		return
	}

	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: data.Bucket.ValueStringPointer(),
		Key:    data.Key.ValueStringPointer(),
	})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read object", err.Error())
		return
	}

	buf := new(strings.Builder)
	n, err := io.Copy(buf, output.Body)
	if err != nil || n != *output.ContentLength {
		if err == nil {
			err = fmt.Errorf("expected %d bytes, got %d", *output.ContentLength, n)
		}
		resp.Diagnostics.AddError("Unable to read object content", err.Error())
		return
	}

	data.Content = types.StringValue(buf.String())
	data.URL = types.StringValue(buildURL(*r.client.Options().BaseEndpoint, data.Bucket.ValueString(), data.Key.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: data.Bucket.ValueStringPointer(),
		Key:    data.Key.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete object", err.Error())
	}
}

func (r *ObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
