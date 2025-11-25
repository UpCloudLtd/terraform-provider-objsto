variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_bucket_lifecycle_configuration" "this" {
  bucket = objsto_bucket.this.bucket

  rule {
    id = "Expire non-current versions after 3 newer versions"

    filter {
      prefix = "test/"
    }

    noncurrent_version_expiration {
      noncurrent_days = 1
      newer_noncurrent_versions = 3
    }
  }

  rule {
    id = "Expire non-current versions after 7 days"

    filter {
      prefix = ""
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }

  rule {
    id = "Expire all objects with status=completed tag"

    filter {
      tag = {
        key   = "status"
        value = "completed"
      }
    }

    expiration {
      date = "2024-01-01T00:00:00Z"
    }
  }

  rule {
    id = "Expire all objects with status=pending and managed-by=team-devex after 30 days"

    filter {
      and {
        tags = {
          status     = "pending"
          managed-by = "team-devex"
        }
      }
    }

    expiration {
      days = 30
    }
  }
}
