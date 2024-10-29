variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "object_message" {
  type    = string
  default = "Hello objsto!"
}

// Deleting bucket should fail because of a BucketNotEmpty error

resource "objsto_object" "this" {
  bucket = var.bucket_name
  key    = "hello.json"
  content = jsonencode({
    message = var.object_message
  })
}
