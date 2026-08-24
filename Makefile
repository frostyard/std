.PHONY: all clean fmt lint lint-version-check vet verify test test-cover coverage-check test-coverage-check tidy check bump help

# Go commands
GO := go
GOFMT := gofmt
# Pinned golangci-lint release, read from mise.toml — the single source of
# every tool pin (core ADR-0043): `mise install` provisions it locally, in CI
# (jdx/mise-action), and on Snowcat workers, verified against mise.lock.
# Bump it there in a dedicated commit; never edit this line.
GOLANGCI_LINT_VERSION := $(strip $(shell sed -n 's/^golangci-lint = "\(.*\)"/\1/p' mise.toml))
# The Go release this module is built with, from go.mod's toolchain line —
# the only Go pin (mise reads the same line). golangci-lint must be built
# with a Go at least this new, or its embedded gofmt and typechecker disagree
# with the toolchain (observed 2026-08-24: 2.13.1/go1.27 vs go1.26 gofmt).
GO_TOOLCHAIN := $(strip $(shell sed -n 's/^toolchain go\(.*\)/\1/p' go.mod))
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
# The _examples/ programs, enumerated at run time by scripts/example-dirs.sh.
# Go package patterns ignore directories starting with "_", so `./...` never
# reaches them and the analyzers must be given the directories explicitly.
# The CI Lint and Verify jobs run the same script, so adding
# _examples/<program>/ is covered here and in CI without editing either file.
EXAMPLE_DIRS_CMD := ./scripts/example-dirs.sh

all: fmt lint test

## fmt: Format Go source files
fmt:
	$(GOFMT) -w $(GOFILES)

## lint: Run linter over the module and the _examples/ programs (requires golangci-lint; warns if the installed version differs from GOLANGCI_LINT_VERSION)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		installed="$$(golangci-lint version --short 2>/dev/null)"; \
		if [ -n "$$installed" ] && [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "warning: golangci-lint $$installed installed, CI pins $(GOLANGCI_LINT_VERSION); results may differ"; \
		fi; \
		golangci-lint run || exit 1; \
		dirs="$$($(EXAMPLE_DIRS_CMD))" || exit 1; \
		golangci-lint run $$dirs || exit 1; \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) not installed; provision every pinned tool with:"; \
		echo "mise install"; \
		exit 1; \
	fi

## vet: Run go vet over the module and the _examples/ programs
vet:
	$(GO) vet ./...
	@dirs="$$($(EXAMPLE_DIRS_CMD))" || exit 1; \
		echo "$(GO) vet $$dirs" | tr '\n' ' '; echo; \
		$(GO) vet $$dirs || exit 1

## test: Run tests, assert instrumented example-subprocess reporter coverage, and write the in-process reporter profile to coverage.out
test:
	$(GO) test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=github.com/frostyard/std/reporter ./...

## test-cover: Run tests with coverage (writes coverage.out plus an HTML report)
test-cover:
	$(GO) test -coverprofile=coverage.out -covermode=atomic -coverpkg=github.com/frostyard/std/reporter ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## coverage-check: Enforce the 95.0% total statement-coverage floor on coverage.out (scripts/check-coverage.sh)
coverage-check:
	./scripts/check-coverage.sh coverage.out 95.0

## test-coverage-check: Self-test scripts/check-coverage.sh against fixture profiles
test-coverage-check:
	./scripts/test-coverage-check.sh

## tidy: Tidy go modules
tidy:
	$(GO) mod tidy

## clean: Remove generated artifacts
clean:
	rm -f coverage.out coverage.html
	$(GO) clean

## lint-version-check: Fail unless the installed golangci-lint is the mise.toml pin and was built with a Go no older than go.mod's toolchain
lint-version-check:
	@test -n "$(GOLANGCI_LINT_VERSION)" || { echo "mise.toml pins no golangci-lint"; exit 1; }
	@installed="$$(golangci-lint version --short 2>/dev/null)" || { echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required (not installed; run: mise install)"; exit 1; }; \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then echo "expected golangci-lint $(GOLANGCI_LINT_VERSION), found $$installed (run: mise install)"; exit 1; fi; \
	built="$$(golangci-lint version 2>/dev/null | sed -n 's/.*built with go\([0-9.]*\).*/\1/p')"; \
	if [ -n "$$built" ] && [ "$$(printf '%s\n%s\n' "$(GO_TOOLCHAIN)" "$$built" | sort -V | head -1)" != "$(GO_TOOLCHAIN)" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) was built with go$$built, older than go.mod's toolchain go$(GO_TOOLCHAIN): bump golangci-lint first (core ADR-0043)"; exit 1; fi

## verify: Credential-free, non-mutating gate (what a read-only reviewer runs): tidy diff, gofmt -l, lint at the exact pin, vet, tests
verify:
	@echo "==> verify: go.mod is tidy"
	$(GO) mod tidy -diff
	@echo "==> verify: gofmt"
	@unformatted="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then echo "$$unformatted"; echo "gofmt: files need formatting (run make fmt)"; exit 1; fi
	@echo "==> verify: golangci-lint $(GOLANGCI_LINT_VERSION) (built with go >= $(GO_TOOLCHAIN))"
	@$(MAKE) --no-print-directory lint-version-check
	@$(MAKE) --no-print-directory lint vet
	@echo "==> verify: tests"
	$(GO) test ./...

## check: Run fmt, lint, vet, test, and the coverage floor
check: fmt lint vet test test-coverage-check coverage-check

## bump: Tag and push next version (requires clean tree and svu)
bump:
	@$(MAKE) check
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working directory not clean. Commit or stash first."; \
		exit 1; \
	fi
	@version=$$(svu next); \
		git tag -a $$version -m "Version $$version"; \
		echo "Tagged $$version"; \
		git push origin $$version

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
