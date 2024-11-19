package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketLifecycleConfigurationResource(t *testing.T) {
	bucket_name := withSuffix("bucket-lifecycle-configuration")
	variables := map[string]config.Variable{
		"bucket_name": config.StringVariable(bucket_name),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/bucket_lifecycle.tf"),
				ConfigVariables: variables,
				Check:           resource.ComposeAggregateTestCheckFunc(),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_lifecycle.tf"),
				ConfigVariables:                      variables,
				ResourceName:                         "objsto_bucket_lifecycle_configuration.this",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerifyIdentifierAttribute: "bucket",
				ImportStateVerify:                    true,
			},
		},
	})
}
