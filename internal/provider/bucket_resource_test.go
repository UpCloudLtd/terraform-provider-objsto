package provider

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// From Kubernetes random suffixes.
const letterBytes = "bcdfghjklmnpqrstvwxz2456789"

func randomSuffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func withSuffix(name string) string {
	nameAndCli := fmt.Sprintf("%s-terraform", name)
	if cliPath := os.Getenv("TF_ACC_TERRAFORM_PATH"); cliPath != "" {
		nameAndCli = fmt.Sprintf("%s-%s", name, filepath.Base(cliPath))
	}

	job := os.Getenv("GITHUB_JOB")
	runNumber := os.Getenv("GITHUB_RUN_NUMBER")
	runAttempt := os.Getenv("GITHUB_RUN_ATTEMPT")
	if runNumber != "" && runAttempt != "" {
		return fmt.Sprintf("%s-github-%s-%s.%s", nameAndCli, job, runNumber, runAttempt)
	}

	randStr := randomSuffix(8)

	if os.Getenv("CI") != "" {
		return fmt.Sprintf("%s-ci-%s", nameAndCli, randStr)
	}

	return fmt.Sprintf("%s-%s", nameAndCli, randStr)
}

func TestAccBucketResource_crud(t *testing.T) {
	bucket_name := withSuffix("bucket-crud")
	variables := map[string]config.Variable{
		"bucket_name": config.StringVariable(bucket_name),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/bucket_crud.tf"),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("objsto_bucket.this", "bucket", bucket_name),
				),
			},
			{
				ConfigFile:                           config.StaticFile("testdata/bucket_crud.tf"),
				ConfigVariables:                      variables,
				ResourceName:                         "objsto_bucket.this",
				ImportState:                          true,
				ImportStateId:                        bucket_name,
				ImportStateVerifyIdentifierAttribute: "bucket",
				ImportStateVerify:                    true,
			},
			{
				ConfigFile:        config.StaticFile("testdata/bucket_crud.tf"),
				ConfigVariables:   variables,
				ResourceName:      "objsto_object.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigFile:      config.StaticFile("testdata/bucket_crud_error.tf"),
				ConfigVariables: variables,
				ExpectError:     regexp.MustCompile("BucketNotEmpty"),
			},
			{
				ConfigFile: config.StaticFile("testdata/bucket_crud.tf"),
				ConfigVariables: map[string]config.Variable{
					"bucket_name":    config.StringVariable(bucket_name),
					"object_message": config.StringVariable("Hello from last test step!"),
				},
			},
		},
	})
}
