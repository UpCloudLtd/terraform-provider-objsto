package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

func checkGetUrl(name, key string, expectedStatus int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		url := rs.Primary.Attributes[key]
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to GET %s: %w", url, err)
		}

		if resp.StatusCode != expectedStatus {
			return fmt.Errorf(`expected GET %s status code to be %d, got %d`, url, expectedStatus, resp.StatusCode)
		}
		return nil
	}
}

func TestNormalizePolicyDocument(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Change ID to Id",
			input:    `{"ID":"PublicRead"}`,
			expected: `{"Id":"PublicRead"}`,
		},
		{
			name:     "Removes null Id",
			input:    `{"Id":"null"}`,
			expected: `{}`,
		},
		{
			name:     "Sorts statement actions",
			input:    `{"Statement":[{"Action":["s3:ListBucket","s3:GetBucketLocation"]}]}`,
			expected: `{"Statement":[{"Action":["s3:GetBucketLocation","s3:ListBucket"]}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, diags := normalizePolicyDocument(test.input)
			assert.False(t, diags.HasError())
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestAccBucketPolicyResource(t *testing.T) {
	bucket_name := withSuffix("bucket-policy")
	variables := func(public_read_access bool) map[string]config.Variable {
		return map[string]config.Variable{
			"bucket_name":        config.StringVariable(bucket_name),
			"public_read_access": config.BoolVariable(public_read_access),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/bucket_policy.tf"),
				ConfigVariables: variables(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGetUrl("objsto_object.this", "url", 403),
				),
			},
			{
				ConfigFile:      config.StaticFile("testdata/bucket_policy.tf"),
				ConfigVariables: variables(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGetUrl("objsto_object.this", "url", 200),
				),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_policy.tf"),
				ConfigVariables:                      variables(true),
				ResourceName:                         "objsto_bucket_policy.this[0]",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerifyIdentifierAttribute: "bucket",
				ImportStateVerify:                    true,
			},
		},
	})
}

func TestAccBucketPolicyResource_Normalization(t *testing.T) {
	tests := []struct {
		configName string
		testImport bool
	}{
		{
			configName: "bucket_policy_str_action",
			testImport: false,
		},
		{
			configName: "bucket_policy_no_id",
			testImport: true,
		},
	}

	for _, test := range tests {
		t.Run(test.configName, func(t *testing.T) {
			configPath := fmt.Sprintf("testdata/%s.tf", test.configName)

			bucket_name := withSuffix("bucket-policy-no-id")
			variables := func(allow_get_object bool) map[string]config.Variable {
				return map[string]config.Variable{
					"bucket_name":      config.StringVariable(bucket_name),
					"allow_get_object": config.BoolVariable(allow_get_object),
				}
			}

			steps := []resource.TestStep{
				{
					ConfigFile:      config.StaticFile(configPath),
					ConfigVariables: variables(false),
					Check:           resource.ComposeAggregateTestCheckFunc(),
				},
				{
					ConfigFile:      config.StaticFile(configPath),
					ConfigVariables: variables(true),
					Check:           resource.ComposeAggregateTestCheckFunc(),
				},
			}

			if test.testImport {
				steps = append(steps, resource.TestStep{
					ConfigFile:                           config.StaticFile(configPath),
					ConfigVariables:                      variables(true),
					ResourceName:                         "objsto_bucket_policy.this",
					ImportState:                          true,
					ImportStateId:                        bucket_name,
					ImportStateVerifyIdentifierAttribute: "bucket",
					ImportStateVerify:                    true,
				})
			}

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    steps,
			})
		})
	}
}
