package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tftest "github.com/hashicorp/terraform-plugin-testing/terraform"
)

func getVariables(bucket, versioning string, content string) map[string]config.Variable {
	return map[string]config.Variable{
		"bucket_name":       config.StringVariable(bucket),
		"bucket_versioning": config.StringVariable(versioning),
		"object_content":    config.StringVariable(content),
	}
}

func deleteObjectsStep(bucket, versioning string) resource.TestStep {
	return resource.TestStep{
		ConfigFile:      config.StaticFile("testdata/bucket_versioning.tf"),
		ConfigVariables: getVariables(bucket, versioning, ""),
		PreConfig: func() {
			ctx := context.TODO()
			client := getClient(ctx, ObjStoProviderModel{})
			objVersions, err := client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
				Bucket: &bucket,
			})
			if err != nil {
				panic(fmt.Sprintf("failed to list object versions: %v", err))
			}

			for _, version := range objVersions.Versions {
				_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket:    &bucket,
					Key:       version.Key,
					VersionId: version.VersionId,
				})
				if err != nil {
					panic(fmt.Sprintf("failed to delete object version: %v", err))
				}
			}

			for _, marker := range objVersions.DeleteMarkers {
				_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket:    &bucket,
					Key:       marker.Key,
					VersionId: marker.VersionId,
				})
				if err != nil {
					panic(fmt.Sprintf("failed to delete object delete marker: %v", err))
				}
			}
		},
	}
}

func TestAccBucketVersioning_Suspended(t *testing.T) {
	bucket_name := withSuffix("bucket-versioning-suspended")
	variables := getVariables(bucket_name, "Suspended", "original")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("objsto_bucket_versioning.this.0", "versioning_configuration.status", "Suspended"),
					resource.TestCheckResourceAttr("objsto_object.this.0", "content", "original"),
				),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables:                      variables,
				ResourceName:                         "objsto_bucket_versioning.this[0]",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			deleteObjectsStep(bucket_name, "Suspended"),
		},
	})
}

func checkStringDoesChange(name, key string, expected *string) resource.TestCheckFunc {
	return func(s *tftest.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		actual := rs.Primary.Attributes[key]
		if *expected == "" {
			*expected = actual
		} else if actual == *expected {
			return fmt.Errorf(`expected %s to change from previous value "%s", but it did not`, key, *expected)
		}
		return nil
	}
}

func checkVersioningIsSuspended(bucket string) resource.TestCheckFunc {
	return func(_ *tftest.State) error {
		ctx := context.TODO()
		client := getClient(ctx, ObjStoProviderModel{})
		versioning, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: &bucket,
		})
		if err != nil {
			return fmt.Errorf("failed to get bucket versioning: %w", err)
		}
		if versioning.Status != "Suspended" {
			return fmt.Errorf("expected bucket versioning to be suspended, got %s", versioning.Status)
		}
		return nil
	}
}

func TestAccBucketVersioning_Enabled(t *testing.T) {
	bucket_name := withSuffix("bucket-versioning-enabled")

	var versionId string
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables: getVariables(bucket_name, "Enabled", "original"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("objsto_bucket_versioning.this.0", "versioning_configuration.status", "Enabled"),
					resource.TestCheckResourceAttr("objsto_object.this.0", "content", "original"),
					checkStringDoesChange("objsto_object.this.0", "version_id", &versionId),
				),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables:                      getVariables(bucket_name, "Enabled", "original"),
				ResourceName:                         "objsto_bucket_versioning.this[0]",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			{
				ConfigFile:      config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables: getVariables(bucket_name, "Enabled", "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("objsto_object.this.0", "content", "updated"),
					checkStringDoesChange("objsto_object.this.0", "version_id", &versionId),
				),
			},
			{
				// Removing versioning resource should suspend versioning.
				ConfigFile:      config.StaticFile("testdata/bucket_versioning.tf"),
				ConfigVariables: getVariables(bucket_name, "", "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkVersioningIsSuspended(bucket_name),
				),
			},
			deleteObjectsStep(bucket_name, ""),
		},
	})
}
