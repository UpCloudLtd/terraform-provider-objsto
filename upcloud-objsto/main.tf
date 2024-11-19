terraform {
  required_providers {
    upcloud = {
      source  = "UpCloudLtd/upcloud"
      version = "~> 5.15"
    }
  }
}

variable "prefix" {
  type    = string
  default = "persistent-tf-acc-test-objsto-provider-"
}

variable "region" {
  type    = string
  default = "europe-1"
}

resource "upcloud_managed_object_storage" "this" {
  name              = "${var.prefix}objstov2"
  region            = var.region
  configured_status = "started"

  network {
    name   = "public"
    family = "IPv4"
    type   = "public"
  }
}

resource "upcloud_managed_object_storage_user" "this" {
  username     = "acc-test"
  service_uuid = upcloud_managed_object_storage.this.id
}

resource "upcloud_managed_object_storage_user_access_key" "this" {
  username     = upcloud_managed_object_storage_user.this.username
  status       = "Active"
  service_uuid = upcloud_managed_object_storage.this.id
}

resource "upcloud_managed_object_storage_policy" "this" {
  name        = "FullAccess"
  description = "Allow full access to all S3 resources."
  document = urlencode(jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:*"
        Resource = "*"
      }
    ]
  }))
  service_uuid = upcloud_managed_object_storage.this.id
}

resource "upcloud_managed_object_storage_user_policy" "this" {
  name         = upcloud_managed_object_storage_policy.this.name
  username     = upcloud_managed_object_storage_user.this.username
  service_uuid = upcloud_managed_object_storage.this.id
}

locals {
  endpoints_list = tolist(upcloud_managed_object_storage.this.endpoint)
  endpoints      = length(local.endpoints_list) > 0 ? local.endpoints_list[index(local.endpoints_list.*.type, "public")] : null
}

output "UPCLOUD_REGION" {
  value = var.region
}

output "UPCLOUD_ENDPOINT" {
  value = "https://${local.endpoints.domain_name}"
}

output "UPCLOUD_ACCESS_KEY" {
    sensitive = true
  value = upcloud_managed_object_storage_user_access_key.this.access_key_id
}

output "UPCLOUD_SECRET_KEY" {
    sensitive = true
  value = upcloud_managed_object_storage_user_access_key.this.secret_access_key
}
