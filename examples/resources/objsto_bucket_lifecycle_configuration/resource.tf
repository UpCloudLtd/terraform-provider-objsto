resource "objsto_bucket" "example" {
  bucket = "example"
}

resource "objsto_bucket_lifecycle_configuration" "example" {
  bucket = objsto_bucket.example.bucket

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
}