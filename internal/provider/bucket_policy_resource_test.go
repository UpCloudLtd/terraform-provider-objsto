package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
