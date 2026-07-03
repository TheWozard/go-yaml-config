.PHONY: check build vet lint test vuln

# Read from ci.yml's golangci-lint-action step so there's one place to bump.
GOLANGCI_LINT_VERSION := $(shell grep -A2 'golangci-lint-action' .github/workflows/ci.yml | grep 'version:' | awk '{print $$2}')

check: build vet lint test vuln

build:
	go build ./...

vet:
	go vet ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

test:
	go test ./... -race

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
