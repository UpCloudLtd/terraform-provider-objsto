# Changelog

All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## [Unreleased]

### Fixed

- objsto_bucket_policy: when comparing policy documents, treat wildcard `Principal` value (`"*"`) as equal to `{"AWS": ["*"]}`.
- objsto_bucket_policy: when comparing policy documents, treat string value in `Principal` field as equal to list containing that string as its only item, e.g. `{"AWS": "*"}` vs. `{"AWS": ["*"]}`.

## [0.2.1]

### Fixed

- objsto_bucket_policy: when comparing policy documents, treat string value in `Action` field as equal to list containing that string as its only item, e.g. `"s3:PutObject"` vs. `["s3:PutObject"]`.

## [0.2.0]

### Added

- objsto_bucket_cors_configuration resource for configuring CORS settings for buckets.

## [0.1.1]

### Fixed

- objsto_bucket_policy: when comparing policy documents, ignore statement action order because Minio returns actions in inconsistent order.

## [0.1.0]

### Added

- Minimal implementation of bucket, bucket policy, bucket lifecycle configuration, and object resources.

[Unreleased]: https://github.com/UpCloudLtd/terraform-provider-upcloud/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/UpCloudLtd/terraform-provider-upcloud/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/UpCloudLtd/terraform-provider-upcloud/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/UpCloudLtd/terraform-provider-upcloud/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/UpCloudLtd/terraform-provider-upcloud/releases/tag/v0.1.0
