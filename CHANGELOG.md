# Changelog

All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## [Unreleased]

### Fixed

- objsto_bucket_policy: when comparing policy documents, ignore statement action order because Minio returns actions in inconsistent order.

## [0.1.0]

### Added

- Minimal implementation of bucket, bucket policy, bucket lifecycle configuration, and object resources.

[Unreleased]: https://github.com/UpCloudLtd/terraform-provider-upcloud/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/UpCloudLtd/terraform-provider-upcloud/releases/tag/v0.1.0
