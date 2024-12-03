default: fmt lint install generate

GIT_VERSION?=$(shell git describe --tags --always --dirty --match=v* 2> /dev/null)
VERSION=$(shell echo $(GIT_VERSION) | sed 's/^v//' | sed 's/-.*//')

PROVIDER_HOSTNAME=registry.upcloud.com
PROVIDER_NAMESPACE=upcloud
PROVIDER_TYPE=objsto
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_PATH=~/.terraform.d/plugins/$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(VERSION)/$(PROVIDER_TARGET)

build:
	@mkdir -p $(PROVIDER_PATH)
	go build \
		-tags release \
		-o $(PROVIDER_PATH)/terraform-provider-$(PROVIDER_TYPE)_v$(VERSION)

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

release-notes: CHANGELOG_HEADER = ^\#\# \[
release-notes: CHANGELOG_VERSION = $(subst v,,$(VERSION))
release-notes:
	@awk \
		'/${CHANGELOG_HEADER}${CHANGELOG_VERSION}/ { flag = 1; next } \
		/${CHANGELOG_HEADER}/ { if ( flag ) { exit; } } \
		flag { if ( n ) { print prev; } n++; prev = $$0 }' \
		CHANGELOG.md

.PHONY: fmt lint test testacc build install generate release-notes
