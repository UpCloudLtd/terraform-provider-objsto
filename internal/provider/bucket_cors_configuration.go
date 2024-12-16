package provider

import (
	"context"
	"errors"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketCORSConfigurationResource{}
var _ resource.ResourceWithImportState = &BucketCORSConfigurationResource{}

func NewBucketCORSConfigurationResource() resource.Resource {
	return &BucketCORSConfigurationResource{}
}

// BucketCORSConfigurationResource defines the resource implementation.
type BucketCORSConfigurationResource struct {
	client *s3.Client
}

// BucketCORSConfigurationResourceModel describes the resource data model.
type BucketCORSConfigurationResourceModel struct {
	Bucket types.String `tfsdk:"bucket"`
	Rules  types.List   `tfsdk:"cors_rule"`
}

type CORSRule struct {
	AllowedHeaders types.Set    `tfsdk:"allowed_headers"`
	AllowedMethods types.Set    `tfsdk:"allowed_methods"`
	AllowedOrigins types.Set    `tfsdk:"allowed_origins"`
	ExposeHeaders  types.Set    `tfsdk:"expose_headers"`
	ID             types.String `tfsdk:"id"`
	MaxAgeSeconds  types.Int32  `tfsdk:"max_age_seconds"`
}

func (r *BucketCORSConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_cors_configuration"
}

func (r *BucketCORSConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A bucket CORS configuration resource. Note that there can only be one CORS configuration per bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket for which to configure the CORS.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"cors_rule": schema.ListNestedBlock{
				MarkdownDescription: "A CORS rule to apply to the bucket.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"allowed_headers": schema.SetAttribute{
							Optional:            true,
							MarkdownDescription: "The headers to include in `Access-Control-Request-Headers` header.",
							ElementType:         types.StringType,
						},
						"allowed_methods": schema.SetAttribute{
							Required:            true,
							MarkdownDescription: "The allowed HTTP methods for this rule.",
							ElementType:         types.StringType,
						},
						"allowed_origins": schema.SetAttribute{
							Required:            true,
							MarkdownDescription: "The allowed origins for this rule.",
							ElementType:         types.StringType,
						},
						"expose_headers": schema.SetAttribute{
							Optional:            true,
							MarkdownDescription: "The headers to include in the `Access-Control-Expose-Headers` header.",
							ElementType:         types.StringType,
						},
						"id": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The identifier of the rule.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"max_age_seconds": schema.Int32Attribute{
							Optional:            true,
							MarkdownDescription: "The cache time in seconds.",
						},
					},
				},
			},
		},
	}
}

func (r *BucketCORSConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = getClientFromProviderData(req.ProviderData)
}

func setCORSConfigurationValues(ctx context.Context, data *BucketCORSConfigurationResourceModel, output *s3.GetBucketCorsOutput) (diags diag.Diagnostics) {
	var d diag.Diagnostics

	rulesData := []CORSRule{}
	for _, rule := range output.CORSRules {
		ruleData := CORSRule{
			ID:            types.StringPointerValue(rule.ID),
			MaxAgeSeconds: types.Int32PointerValue(rule.MaxAgeSeconds),
		}

		ruleData.AllowedHeaders, d = types.SetValueFrom(ctx, types.StringType, rule.AllowedHeaders)
		diags.Append(d...)

		ruleData.AllowedMethods, d = types.SetValueFrom(ctx, types.StringType, rule.AllowedMethods)
		diags.Append(d...)

		ruleData.AllowedOrigins, d = types.SetValueFrom(ctx, types.StringType, rule.AllowedOrigins)
		diags.Append(d...)

		ruleData.ExposeHeaders, d = types.SetValueFrom(ctx, types.StringType, rule.ExposeHeaders)
		diags.Append(d...)

		rulesData = append(rulesData, ruleData)
	}

	data.Rules, d = types.ListValueFrom(ctx, data.Rules.ElementType(ctx), rulesData)
	diags.Append(d...)
	return
}

func (r *BucketCORSConfigurationResource) put(ctx context.Context, data *BucketCORSConfigurationResourceModel) (diags diag.Diagnostics) {
	var rulesData []CORSRule
	diags.Append(data.Rules.ElementsAs(ctx, &rulesData, false)...)

	rules := []s3_types.CORSRule{}
	for _, ruleData := range rulesData {
		rule := s3_types.CORSRule{
			ID:            ruleData.ID.ValueStringPointer(),
			MaxAgeSeconds: ruleData.MaxAgeSeconds.ValueInt32Pointer(),
		}

		diags.Append(ruleData.AllowedMethods.ElementsAs(ctx, &rule.AllowedMethods, false)...)
		diags.Append(ruleData.AllowedOrigins.ElementsAs(ctx, &rule.AllowedOrigins, false)...)
		diags.Append(ruleData.AllowedHeaders.ElementsAs(ctx, &rule.AllowedHeaders, false)...)
		diags.Append(ruleData.ExposeHeaders.ElementsAs(ctx, &rule.ExposeHeaders, false)...)

		rules = append(rules, rule)
	}

	_, err := r.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: data.Bucket.ValueStringPointer(),
		CORSConfiguration: &s3_types.CORSConfiguration{
			CORSRules: rules,
		},
	})
	if err != nil {
		diags.AddError("Unable to create bucket CORS configuration", err.Error())
	}
	return
}

func (r *BucketCORSConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketCORSConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCORSConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketCORSConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	output, err := r.client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: data.Bucket.ValueStringPointer(),
	})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read bucket CORS configuration", err.Error())
		return
	}

	resp.Diagnostics.Append(setCORSConfigurationValues(ctx, &data, output)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCORSConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketCORSConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCORSConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketCORSConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteBucketCors(ctx, &s3.DeleteBucketCorsInput{
		Bucket: data.Bucket.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete bucket CORS", err.Error())
	}
}

func (r *BucketCORSConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
