package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = isValidRFC3339{}

type isValidRFC3339 struct{}

// Description describes the validation.
func (v isValidRFC3339) Description(_ context.Context) string {
	return "must be a valid RFC3339 timestamp"
}

// MarkdownDescription describes the validation in Markdown.
func (v isValidRFC3339) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v isValidRFC3339) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	_, err := time.Parse(time.RFC3339, request.ConfigValue.ValueString())
	if err != nil {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			request.Path,
			v.Description(ctx),
			request.ConfigValue.ValueString(),
		))
	}
}
