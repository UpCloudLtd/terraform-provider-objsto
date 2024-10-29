resource "objsto_bucket" "example" {
  bucket = "example"
}

// Allow anonymous read access to the bucket
resource "objsto_bucket_policy" "example" {
  bucket = objsto_bucket.example.bucket
  policy = jsonencode({
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
          objsto_bucket.example.arn,
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
          "${objsto_bucket.example.arn}/*",
        ]
      },
    ],
  })
}
