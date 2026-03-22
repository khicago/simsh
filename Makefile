GO ?= go
GOLANGCI_LINT ?= golangci-lint
PNPM ?= pnpm

export CODEX_HOME := $(PWD)/.codex
ROOT ?= $(CURDIR)
PORT ?= 18080
LISTEN ?= :18080
PROFILE ?= core-strict
CMD ?= env PATH

.PHONY: help test test-race lint check doc cli cli-c cli-serve simshd codex-locale codex-locale-resume

help:
	@echo "Common targets:"
	@echo "  make test        # go test ./..."
	@echo "  make test-race   # go test -race ./..."
	@echo "  make lint        # staticcheck ./... (if installed)"
	@echo "  make check       # test + lint"
	@echo "  make doc         # regenerate simsh.md"
	@echo "  make cli         # run interactive simsh-cli"
	@echo "  make cli-c CMD='ls -l /'    # run one command via simsh-cli"
	@echo "  make cli-serve PORT=18080   # run simsh-cli serve"
	@echo "  make simshd LISTEN=':18080' # run simshd service"
	@echo "  make codex-locale           # launch codex local profile"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

lint:
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed; skip lint"; \
	fi

check: test lint

doc:
	$(GO) run ./cmd/simsh-doc

cli:
	$(GO) run ./cmd/simsh-cli -profile $(PROFILE)

cli-c:
	$(GO) run ./cmd/simsh-cli -profile $(PROFILE) -c "$(CMD)"

cli-serve:
	$(GO) run ./cmd/simsh-cli serve -P $(PORT) -root "$(ROOT)" -profile $(PROFILE)

simshd:
	$(GO) run ./cmd/simshd -listen "$(LISTEN)" -root "$(ROOT)" -profile $(PROFILE)

# BAGAKIT:LONGRUN:LAUNCHER:START
ralphloop:
	bash .bagakit/long-run/ralphloop-runner.sh
.PHONY: ralphloop
# BAGAKIT:LONGRUN:LAUNCHER:END
