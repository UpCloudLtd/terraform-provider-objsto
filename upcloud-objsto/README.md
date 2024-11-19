# UpCloud Managed Object Storage acceptance test target

This directory contains configuration for creating UpCloud Managed Object Storage acceptance test target.

```sh
terraform init
terraform apply
```

Values to be configured into workflow secrets are available as outputs.

```sh
terraform output -json
```