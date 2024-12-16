variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "public_read_access" {
  type    = bool
  default = true
}

variable "configure_cors" {
  type    = bool
  default = false
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_object" "this" {
  bucket = objsto_bucket.this.bucket
  key    = "hello.json"
  content = jsonencode({
    message = "Hello objsto!"
  })
}

resource "objsto_bucket_policy" "this" {
  count = var.public_read_access ? 1 : 0

  bucket = objsto_bucket.this.bucket
  policy = jsonencode({
    Id      = "PublicRead",
    Version = "2012-10-17",
    Statement = [
      {
        Principal = {
          "AWS" = ["*"]
        }
        Effect = "Allow"
        Action = [
          "s3:GetBucketLocation",
          "s3:ListBucket",
        ]
        Resource = [
          objsto_bucket.this.arn,
        ]
      },
      {
        Principal = {
          "AWS" = ["*"]
        }
        Effect = "Allow"
        Action = [
          "s3:GetObject",
        ]
        Resource = [
          "${objsto_bucket.this.arn}/*",
        ]
      },
    ],
  })
}

resource "objsto_bucket_cors_configuration" "this" {
  count = var.configure_cors ? 1 : 0

  bucket = objsto_bucket.this.bucket

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD", "DELETE", "PUT", "POST"]
    allowed_origins = ["*"]
    expose_headers  = ["x-amz-server-side-encryption"]
    max_age_seconds = 3000
  }
}
