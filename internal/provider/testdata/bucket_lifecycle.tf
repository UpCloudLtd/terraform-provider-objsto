variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_bucket_lifecycle_configuration" "this" {
  bucket = objsto_bucket.this.bucket

  rules = [
    {
      id = "Expire non-current versions after 7 days"
      filter = {
        prefix = "",
      }
      noncurrent_version_expiration = {
        noncurrent_days = 7
      }
    },
    {
      id = "Expire all objects with status=completed tag"
      filter = {
        tag = {
          key   = "status"
          value = "completed"
        },
      }
      expiration = {
        date = "2024-01-01T00:00:00Z"
      }
    },
  ]
}
