resource "objsto_bucket" "example" {
  bucket = "example"
}

resource "objsto_bucket_cors_configuration" "this" {
  bucket = objsto_bucket.example.bucket

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD", "DELETE", "PUT", "POST"]
    allowed_origins = ["*"]
    expose_headers  = ["x-amz-server-side-encryption"]
    max_age_seconds = 3000
  }
}
