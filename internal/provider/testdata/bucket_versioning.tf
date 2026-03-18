variable "bucket_name" {
  type    = string
  default = "objsto-acc-test"
}

variable "bucket_versioning" {
  type    = string
  default = ""
}

variable "object_content" {
  type    = string
  default = "Hello objsto!"
}

resource "objsto_bucket" "this" {
  bucket = var.bucket_name
}

resource "objsto_bucket_versioning" "this" {
  count = var.bucket_versioning != "" ? 1 : 0

  bucket = objsto_bucket.this.bucket

  versioning_configuration {
    status = var.bucket_versioning
  }
}

resource "objsto_object" "this" {
  count = var.object_content != "" ? 1 : 0

  bucket  = objsto_bucket.this.bucket
  key     = "object.txt"
  content = var.object_content
}
