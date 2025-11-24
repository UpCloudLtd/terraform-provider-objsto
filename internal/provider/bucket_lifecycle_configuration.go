package provider

import (
	"context"
	"errors"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketLifecycleConfigurationResource{}
var _ resource.ResourceWithImportState = &BucketLifecycleConfigurationResource{}

func NewBucketLifecycleConfigurationResource() resource.Resource {
	return &BucketLifecycleConfigurationResource{}
}

// BucketLifecycleConfigurationResource defines the resource implementation.
type BucketLifecycleConfigurationResource struct {
	client *s3.Client
}

// BucketLifecycleConfigurationResourceModel describes the resource data model.
type BucketLifecycleConfigurationResourceModel struct {
	Bucket types.String `tfsdk:"bucket"`
	Rules  types.List   `tfsdk:"rule"`
}

type LifecycleConfigurationRule struct {
	ID                          types.String `tfsdk:"id"`
	Status                      types.String `tfsdk:"status"`
	Filter                      types.Object `tfsdk:"filter"`
	Expiration                  types.Object `tfsdk:"expiration"`
	NoncurrentVersionExpiration types.Object `tfsdk:"noncurrent_version_expiration"`
}

type LifecycleConfigurationRuleFilter struct {
	ObjectSizeLargerThan types.Int64  `tfsdk:"object_size_larger_than"`
	ObjectSizeLessThan   types.Int64  `tfsdk:"object_size_less_than"`
	Prefix               types.String `tfsdk:"prefix"`
	Tag                  types.Object `tfsdk:"tag"`
	And                  types.Object `tfsdk:"and"`
}

func (m LifecycleConfigurationRuleFilter) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"object_size_larger_than": types.Int64Type,
		"object_size_less_than":   types.Int64Type,
		"prefix":                  types.StringType,
		"tag": basetypes.ObjectType{
			AttrTypes: Tag{}.AttributeTypes(),
		},
		"and": basetypes.ObjectType{
			AttrTypes: LifecycleConfigurationRuleFilterAnd{}.AttributeTypes(),
		},
	}
}

type LifecycleConfigurationRuleFilterAnd struct {
	ObjectSizeLargerThan types.Int64  `tfsdk:"object_size_larger_than"`
	ObjectSizeLessThan   types.Int64  `tfsdk:"object_size_less_than"`
	Prefix               types.String `tfsdk:"prefix"`
	Tag                  types.Map    `tfsdk:"tags"`
}

func (m LifecycleConfigurationRuleFilterAnd) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"object_size_larger_than": types.Int64Type,
		"object_size_less_than":   types.Int64Type,
		"prefix":                  types.StringType,
		"tags": basetypes.MapType{
			ElemType: types.StringType,
		},
	}
}

type LifecycleConfigurationRuleExpiration struct {
	Date types.String `tfsdk:"date"`
	Days types.Int32  `tfsdk:"days"`
}

func (m LifecycleConfigurationRuleExpiration) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"date": types.StringType,
		"days": types.Int32Type,
	}
}

type LifecycleConfigurationRuleNoncurrentExpiration struct {
	NewerNoncurrentVersions types.Int32 `tfsdk:"newer_noncurrent_versions"`
	NoncurrentDays          types.Int32 `tfsdk:"noncurrent_days"`
}

func (m LifecycleConfigurationRuleNoncurrentExpiration) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"newer_noncurrent_versions": types.Int32Type,
		"noncurrent_days":           types.Int32Type,
	}
}

type Tag struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (m Tag) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

func (r *BucketLifecycleConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_lifecycle_configuration"
}

func (r *BucketLifecycleConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A bucket lifecycle configuration resource. Note that there can only be one lifecycle configuration per bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket for which to configure the lifecycle policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				MarkdownDescription: "The lifecycle rules to apply to the bucket.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The identifier of the rule.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"status": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The status of the rule.",
							Default:             stringdefault.StaticString("Enabled"),
							Validators: []validator.String{
								stringvalidator.OneOf(
									"Enabled",
									"Disabled",
								),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"filter": schema.SingleNestedBlock{
							MarkdownDescription: "A filter to select object that the rule applies to.",
							Validators: []validator.Object{
								objectvalidator.IsRequired(),
							},
							Attributes: map[string]schema.Attribute{
								"object_size_larger_than": schema.Int64Attribute{
									Optional:            true,
									MarkdownDescription: "The minimum object size in bytes.",
									Validators: []validator.Int64{
										int64validator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("object_size_less_than"),
											path.MatchRelative().AtParent().AtName("prefix"),
											path.MatchRelative().AtParent().AtName("tag"),
											path.MatchRelative().AtParent().AtName("and"),
										),
									},
								},
								"object_size_less_than": schema.Int64Attribute{
									Optional:            true,
									MarkdownDescription: "The maximum object size in bytes.",
								},
								"prefix": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "The prefix of the object key.",
								},
								"tag": schema.SingleNestedAttribute{
									Optional:            true,
									MarkdownDescription: "The tag of the object.",
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The key of the tag.",
										},
										"value": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The value of the tag.",
										},
									},
								},
							},
							Blocks: map[string]schema.Block{
								"and": schema.SingleNestedBlock{
									MarkdownDescription: "A logical AND filter.",
									Attributes: map[string]schema.Attribute{
										"object_size_larger_than": schema.Int64Attribute{
											Optional:            true,
											MarkdownDescription: "The minimum object size in bytes.",
										},
										"object_size_less_than": schema.Int64Attribute{
											Optional:            true,
											MarkdownDescription: "The maximum object size in bytes.",
										},
										"prefix": schema.StringAttribute{
											Optional:            true,
											MarkdownDescription: "The prefix of the object key.",
										},
										"tags": schema.MapAttribute{
											Optional:            true,
											MarkdownDescription: "The tags of the object.",
											ElementType:         types.StringType,
										},
									},
								},
							},
						},
						"expiration": schema.SingleNestedBlock{
							MarkdownDescription: "The expiration of the object.",
							Attributes: map[string]schema.Attribute{
								"date": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "The date of the expiration.",
									Validators: []validator.String{
										stringvalidator.ConflictsWith(
											path.MatchRelative().AtParent().AtName("days"),
										),
										isValidRFC3339{},
									},
								},
								"days": schema.Int32Attribute{
									Optional:            true,
									MarkdownDescription: "The number of days until expiration.",
								},
							},
						},
						"noncurrent_version_expiration": schema.SingleNestedBlock{
							MarkdownDescription: "The expiration of the noncurrent versions of the object.",
							Attributes: map[string]schema.Attribute{
								"newer_noncurrent_versions": schema.Int32Attribute{
									Optional:            true,
									MarkdownDescription: "The number of newer noncurrent versions.",
								},
								"noncurrent_days": schema.Int32Attribute{
									Optional:            true,
									MarkdownDescription: "The number of days until expiration of the noncurrent versions.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *BucketLifecycleConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = getClientFromProviderData(req.ProviderData)
}

var objectAsOptions = basetypes.ObjectAsOptions{
	UnhandledNullAsEmpty:    true,
	UnhandledUnknownAsEmpty: true,
}

func setRuleExpiration(ctx context.Context, rule *s3_types.LifecycleRule, ruleData *LifecycleConfigurationRule) (diags diag.Diagnostics) {
	if ruleData.Expiration.IsNull() {
		return
	}

	var expirationData LifecycleConfigurationRuleExpiration
	diags.Append(ruleData.Expiration.As(ctx, &expirationData, objectAsOptions)...)

	var date *time.Time
	if !expirationData.Date.IsNull() {
		date_, err := time.Parse(time.RFC3339, expirationData.Date.ValueString())
		if err != nil {
			diags.AddError("Failed to parse date", err.Error())
		}
		date = &date_
	}
	rule.Expiration = &s3_types.LifecycleExpiration{
		Date: date,
		Days: expirationData.Days.ValueInt32Pointer(),
	}
	return
}

func setRuleFilter(ctx context.Context, rule *s3_types.LifecycleRule, ruleData *LifecycleConfigurationRule) (diags diag.Diagnostics) {
	var filterData LifecycleConfigurationRuleFilter
	diags.Append(ruleData.Filter.As(ctx, &filterData, objectAsOptions)...)

	var filterTag *s3_types.Tag
	if !filterData.Tag.IsNull() {
		var filterTagData Tag
		diags.Append(filterData.Tag.As(ctx, &filterTagData, objectAsOptions)...)
		filterTag = &s3_types.Tag{
			Key:   filterTagData.Key.ValueStringPointer(),
			Value: filterTagData.Value.ValueStringPointer(),
		}
	}

	var filterAnd *s3_types.LifecycleRuleAndOperator
	if !filterData.And.IsNull() {
		var filterAndData LifecycleConfigurationRuleFilterAnd
		diags.Append(filterData.And.As(ctx, &filterAndData, objectAsOptions)...)

		filterAndTagsData := make(map[string]types.String)
		if !filterAndData.Tag.IsNull() {
			diags.Append(filterAndData.Tag.ElementsAs(ctx, &filterAndTagsData, false)...)
		}

		filterAndTags := []s3_types.Tag{}
		for key, value := range filterAndTagsData {
			key := key
			filterAndTags = append(filterAndTags, s3_types.Tag{
				Key:   &key,
				Value: value.ValueStringPointer(),
			})
		}

		filterAnd = &s3_types.LifecycleRuleAndOperator{
			ObjectSizeGreaterThan: filterAndData.ObjectSizeLargerThan.ValueInt64Pointer(),
			ObjectSizeLessThan:    filterAndData.ObjectSizeLessThan.ValueInt64Pointer(),
			Prefix:                filterAndData.Prefix.ValueStringPointer(),
			Tags:                  filterAndTags,
		}
	}

	rule.Filter = &s3_types.LifecycleRuleFilter{
		And:                   filterAnd,
		ObjectSizeGreaterThan: filterData.ObjectSizeLargerThan.ValueInt64Pointer(),
		ObjectSizeLessThan:    filterData.ObjectSizeLessThan.ValueInt64Pointer(),
		Prefix:                filterData.Prefix.ValueStringPointer(),
		Tag:                   filterTag,
	}
	return
}

func setRuleNoncurrentExpiration(ctx context.Context, rule *s3_types.LifecycleRule, ruleData *LifecycleConfigurationRule) (diags diag.Diagnostics) {
	if ruleData.NoncurrentVersionExpiration.IsNull() {
		return
	}

	var noncurrentExpirationData LifecycleConfigurationRuleNoncurrentExpiration
	diags.Append(ruleData.NoncurrentVersionExpiration.As(ctx, &noncurrentExpirationData, objectAsOptions)...)

	rule.NoncurrentVersionExpiration = &s3_types.NoncurrentVersionExpiration{
		NoncurrentDays:          noncurrentExpirationData.NoncurrentDays.ValueInt32Pointer(),
		NewerNoncurrentVersions: noncurrentExpirationData.NewerNoncurrentVersions.ValueInt32Pointer(),
	}
	return
}

func setLifecycleConfigurationValues(ctx context.Context, data *BucketLifecycleConfigurationResourceModel, output *s3.GetBucketLifecycleConfigurationOutput) (diags diag.Diagnostics) {
	var d diag.Diagnostics

	rulesData := []LifecycleConfigurationRule{}
	for _, rule := range output.Rules {
		ruleData := LifecycleConfigurationRule{
			ID:     types.StringPointerValue(rule.ID),
			Status: types.StringValue(string(rule.Status)),
		}

		if rule.Expiration != nil {
			value := LifecycleConfigurationRuleExpiration{
				Days: types.Int32PointerValue(rule.Expiration.Days),
			}
			if rule.Expiration.Date != nil {
				value.Date = types.StringValue(rule.Expiration.Date.Format(time.RFC3339))
			}
			ruleData.Expiration, d = types.ObjectValueFrom(ctx, value.AttributeTypes(), value)
			diags.Append(d...)
		} else {
			ruleData.Expiration = types.ObjectNull(LifecycleConfigurationRuleExpiration{}.AttributeTypes())
		}

		if rule.Filter != nil {
			filter := rule.Filter

			var tagData Tag
			tag := types.ObjectNull(tagData.AttributeTypes())
			if filter.Tag != nil {
				tagData = Tag{
					Key:   types.StringPointerValue(filter.Tag.Key),
					Value: types.StringPointerValue(filter.Tag.Value),
				}
				tag, d = types.ObjectValueFrom(ctx, tagData.AttributeTypes(), tagData)
				diags.Append(d...)
			}

			var andData LifecycleConfigurationRuleFilterAnd
			and := types.ObjectNull(andData.AttributeTypes())
			if filter.And != nil {
				andTagsData := make(map[string]attr.Value)
				for _, tag := range filter.And.Tags {
					andTagsData[*tag.Key] = types.StringPointerValue(tag.Value)
				}
				andTags, d := types.MapValue(types.StringType, andTagsData)
				diags.Append(d...)

				andData = LifecycleConfigurationRuleFilterAnd{
					ObjectSizeLargerThan: types.Int64PointerValue(filter.And.ObjectSizeGreaterThan),
					ObjectSizeLessThan:   types.Int64PointerValue(filter.And.ObjectSizeLessThan),
					Prefix:               types.StringPointerValue(filter.And.Prefix),
					Tag:                  andTags,
				}
				and, d = types.ObjectValueFrom(ctx, andData.AttributeTypes(), andData)
				diags.Append(d...)
			}

			value := LifecycleConfigurationRuleFilter{
				ObjectSizeLargerThan: types.Int64PointerValue(filter.ObjectSizeGreaterThan),
				ObjectSizeLessThan:   types.Int64PointerValue(filter.ObjectSizeLessThan),
				Prefix:               types.StringPointerValue(filter.Prefix),
				Tag:                  tag,
				And:                  and,
			}
			ruleData.Filter, d = types.ObjectValueFrom(ctx, value.AttributeTypes(), value)
			diags.Append(d...)
		}

		if rule.NoncurrentVersionExpiration != nil {
			value := LifecycleConfigurationRuleNoncurrentExpiration{
				NewerNoncurrentVersions: types.Int32PointerValue(rule.NoncurrentVersionExpiration.NewerNoncurrentVersions),
				NoncurrentDays:          types.Int32PointerValue(rule.NoncurrentVersionExpiration.NoncurrentDays),
			}
			ruleData.NoncurrentVersionExpiration, d = types.ObjectValueFrom(ctx, value.AttributeTypes(), value)
			diags.Append(d...)
		} else {
			ruleData.NoncurrentVersionExpiration = types.ObjectNull(LifecycleConfigurationRuleNoncurrentExpiration{}.AttributeTypes())
		}

		rulesData = append(rulesData, ruleData)
	}

	data.Rules, d = types.ListValueFrom(ctx, data.Rules.ElementType(ctx), rulesData)
	diags.Append(d...)
	return
}

func (r *BucketLifecycleConfigurationResource) put(ctx context.Context, data *BucketLifecycleConfigurationResourceModel) (diags diag.Diagnostics) {
	var rulesData []LifecycleConfigurationRule
	diags.Append(data.Rules.ElementsAs(ctx, &rulesData, false)...)

	rules := []s3_types.LifecycleRule{}
	for _, ruleData := range rulesData {
		rule := s3_types.LifecycleRule{
			ID:     ruleData.ID.ValueStringPointer(),
			Status: s3_types.ExpirationStatus(ruleData.Status.ValueString()),
		}

		diags.Append(setRuleExpiration(ctx, &rule, &ruleData)...)
		diags.Append(setRuleFilter(ctx, &rule, &ruleData)...)
		diags.Append(setRuleNoncurrentExpiration(ctx, &rule, &ruleData)...)
		rules = append(rules, rule)
	}

	_, err := r.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: data.Bucket.ValueStringPointer(),
		LifecycleConfiguration: &s3_types.BucketLifecycleConfiguration{
			Rules: rules,
		},
	})
	if err != nil {
		diags.AddError("Unable to create bucket lifecycle configuration", err.Error())
	}
	return
}

func (r *BucketLifecycleConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketLifecycleConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketLifecycleConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	output, err := r.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: data.Bucket.ValueStringPointer(),
	})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read bucket lifecycle configuration", err.Error())
		return
	}

	resp.Diagnostics.Append(setLifecycleConfigurationValues(ctx, &data, output)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketLifecycleConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketLifecycleConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteBucketLifecycle(ctx, &s3.DeleteBucketLifecycleInput{
		Bucket: data.Bucket.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete bucket lifecycle", err.Error())
	}
}

func (r *BucketLifecycleConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
