package provider

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func checkCORSHeaders(name, key, origin string, expectedHeaders map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		url := rs.Primary.Attributes[key]
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed create GET request %s: %w", url, err)
		}
		req.Header.Set("Origin", origin)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to GET %s: %w", url, err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("expected GET request to succeed, got %d status code. %s", resp.StatusCode, string(b))
		}

		errors := []string{}
		for header, expectedValue := range expectedHeaders {
			actualValue := resp.Header.Get(header)
			if actualValue != expectedValue {
				errors = append(errors, fmt.Sprintf(`- Expected "%s" header value to be "%s", got "%s"`, header, expectedValue, resp.Header.Get(header)))
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("actual headers do not match expected headers:\n%s", strings.Join(errors, "\n"))
		}

		return nil
	}
}

func testTargetIs(target string) bool {
	return strings.EqualFold(os.Getenv("TEST_TARGET"), target)
}

func TestAccBucketCORSConfigurationResource(t *testing.T) {
	if testTargetIs("Minio") {
		t.Skip("Skipping CORS configuration tests because target object storage is Minio which does not support configuring CORS settings for buckets.")
	}

	bucket_name := withSuffix("bucket-cors-configuration")

	variables := func(configureCORS bool) map[string]config.Variable {
		return map[string]config.Variable{
			"bucket_name":    config.StringVariable(bucket_name),
			"configure_cors": config.BoolVariable(configureCORS),
		}
	}

	origin := "https://acc-test.example.com"

	corsDisabledStep := resource.TestStep{
		ConfigFile:      config.StaticFile("testdata/bucket_policy.tf"),
		ConfigVariables: variables(false),
		Check: resource.ComposeAggregateTestCheckFunc(
			checkCORSHeaders("objsto_object.this", "url", origin, map[string]string{
				"Access-Control-Allow-Origin": "",
			}),
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			corsDisabledStep,
			{
				ConfigFile:      config.StaticFile("testdata/bucket_policy.tf"),
				ConfigVariables: variables(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCORSHeaders("objsto_object.this", "url", origin, map[string]string{
						"Access-Control-Allow-Origin": origin,
					}),
				),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_policy.tf"),
				ConfigVariables:                      variables(true),
				ResourceName:                         "objsto_bucket_cors_configuration.this[0]",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerifyIdentifierAttribute: "bucket",
				ImportStateVerify:                    true,
			},
			corsDisabledStep,
		},
	})
}
