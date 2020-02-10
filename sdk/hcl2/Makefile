PROJECT_NAME     := Pulumi HCL2 SDK
LANGHOST_PKG     := github.com/pulumi/pulumi/sdk/hcl2/pulumi-language-hcl2
VERSION          := $(shell ../../scripts/get-version HEAD)
PROJECT_PKGS     := $(shell go list ./pulumi/... ./pulumi-language-hcl2/... | grep -v /vendor/ | grep -v templates)

TESTPARALLELISM := 10

include ../../build/common.mk

build::
	go install -ldflags "-X github.com/pulumi/pulumi/pkg/version.Version=${VERSION}" ${LANGHOST_PKG}

install_plugin::
	GOBIN=$(PULUMI_BIN) go install -ldflags "-X github.com/pulumi/pulumi/pkg/version.Version=${VERSION}" ${LANGHOST_PKG}

install:: install_plugin

lint::
	golangci-lint run

test_fast::
	go test -count=1 -cover -parallel ${TESTPARALLELISM} ${PROJECT_PKGS}

dist::
	go install -ldflags "-X github.com/pulumi/pulumi/pkg/version.Version=${VERSION}" ${LANGHOST_PKG}
