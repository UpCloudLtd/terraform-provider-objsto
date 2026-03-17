resource "objsto_bucket" "example" {
  bucket = "example"
}

resource "objsto_bucket_versioning" "example" {
  bucket = objsto_bucket.example.bucket

  versioning_configuration {
    status = "Enabled"
  }
}
