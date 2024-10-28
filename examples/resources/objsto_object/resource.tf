resource "objsto_bucket" "example" {
  bucket = "example"
}

resource "objsto_object" "example" {
  bucket = objsto_bucket.example.bucket
  key    = "hello.json"
  content = jsonencode({
    message = "Hello objsto!"
  })
}
