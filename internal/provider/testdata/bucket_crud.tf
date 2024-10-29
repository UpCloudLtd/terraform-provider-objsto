variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "object_message" {
  type    = string
  default = "Hello objsto!"
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_object" "this" {
  bucket = objsto_bucket.this.bucket
  key    = "hello.json"
  content = jsonencode({
    message = var.object_message
  })
}
