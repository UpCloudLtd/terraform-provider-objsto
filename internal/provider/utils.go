package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func passthroughUpdate[T any](ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data T
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func getClientFromProviderData(providerData any) (client *s3.Client, diags diag.Diagnostics) {
	if providerData == nil {
		return
	}

	client, ok := providerData.(*s3.Client)
	if !ok {
		diags.AddError(
			"Unexpected resource Configure type",
			fmt.Sprintf("Expected *s3.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
	}

	return
}

type valueOrEnvValidator struct {
	envKey string
}

var _ validator.String = valueOrEnvValidator{}

func NewValueOrEnvValidator(envKey string) valueOrEnvValidator {
	return valueOrEnvValidator{envKey: envKey}
}

func (v valueOrEnvValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be defined either in the configuration or with the %s environment variable", v.envKey)
}

func (v valueOrEnvValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v valueOrEnvValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	envValue := os.Getenv(v.envKey)
	if req.ConfigValue.IsNull() && envValue == "" {
		resp.Diagnostics.AddError(
			fmt.Sprintf("No value found for %s", req.Path.String()),
			fmt.Sprintf("Value must be defined either in the configuration or with the %s environment variable", v.envKey))
	}
}
