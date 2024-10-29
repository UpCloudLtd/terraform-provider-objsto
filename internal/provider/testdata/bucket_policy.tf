variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "public_read_access" {
  type    = bool
  default = true
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
