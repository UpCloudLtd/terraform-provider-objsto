package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	envKeyEndpoint  string = "OBJSTO_ENDPOINT"
	envKeyRegion    string = "OBJSTO_REGION"
	envKeyAccessKey string = "OBJSTO_ACCESS_KEY"
	envKeySecretKey string = "OBJSTO_SECRET_KEY"
)

// Ensure ObjStoProvider satisfies various provider interfaces.
var _ provider.Provider = &ObjStoProvider{}

// ObjStoProvider defines the provider implementation.
type ObjStoProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ObjStoProviderModel describes the provider data model.
type ObjStoProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	Region    types.String `tfsdk:"region"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

func (p *ObjStoProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "objsto"
	resp.Version = p.version
}

func withStringDefault(val types.String, def string) string {
	if val.IsNull() {
		return def
	}
	return val.ValueString()
}

func withEnvDefault(val types.String, env string) string {
	return withStringDefault(val, os.Getenv(env))
}

func envAlternative(description, envKey string) string {
	return fmt.Sprintf("%s. Can also be configured with `%s` environment variable.", description, envKey)
}

func (p *ObjStoProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The `objsto` provider is used to manage S3 compatible object storage services such as [UpCloud Managed Object Storage](https://upcloud.com/products/object-storage).",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: envAlternative("S3 endpoint of the object storage service", envKeyEndpoint),
				Optional:            true,
				Validators: []validator.String{
					NewValueOrEnvValidator(envKeyEndpoint),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: envAlternative("Region of the object storage service", envKeyRegion),
				Optional:            true,
				Validators: []validator.String{
					NewValueOrEnvValidator(envKeyRegion),
				},
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: envAlternative("Access key for the object storage service", envKeyAccessKey),
				Optional:            true,
				Validators: []validator.String{
					NewValueOrEnvValidator(envKeyAccessKey),
				},
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: envAlternative("Secret key for the object storage service", envKeySecretKey),
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					NewValueOrEnvValidator(envKeySecretKey),
				},
			},
		},
	}
}

func (p *ObjStoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ObjStoProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := withEnvDefault(data.Endpoint, envKeyEndpoint)
	client := s3.New(s3.Options{
		BaseEndpoint:  &endpoint,
		ClientLogMode: aws.LogRetries | aws.LogRequestWithBody | aws.LogResponseWithBody,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			withEnvDefault(data.AccessKey, envKeyAccessKey),
			withEnvDefault(data.SecretKey, envKeySecretKey),
			"",
		)),
		Logger:       logger{ctx: ctx},
		UsePathStyle: true,
		Region:       withEnvDefault(data.Region, envKeyRegion),
	})

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ObjStoProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBucketResource,
		NewBucketLifecycleConfigurationResource,
		NewBucketPolicyResource,
		NewObjectResource,
	}
}

func (p *ObjStoProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ObjStoProvider{
			version: version,
		}
	}
}
