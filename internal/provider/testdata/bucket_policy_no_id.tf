variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "allow_get_object" {
  type    = bool
  default = false
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_bucket_policy" "this" {
  bucket = objsto_bucket.this.bucket
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Principal = "*"
        Effect = var.allow_get_object ? "Allow" : "Deny"
        Action = [
          "s3:GetObject",
        ]
        Resource = [
          "${objsto_bucket.this.arn}/*",
        ]
      },
  ] })
}
