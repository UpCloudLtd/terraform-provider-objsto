---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "objsto_object Resource - objsto"
subcategory: ""
description: |-
  A object resource that represents an object stored in a bucket.
---

# objsto_object (Resource)

A object resource that represents an object stored in a bucket.

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required Attributes

- `bucket` (String) The name of the bucket where to store the object.
- `content` (String) The content of the object.
- `key` (String) The key of the object.

### Read-Only

- `id` (String) The id of the object. The id is in `{bucket}/{key}` format.
- `url` (String) The URL of the object.
