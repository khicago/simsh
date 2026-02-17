GO ?= go
GOLANGCI_LINT ?= golangci-lint
PNPM ?= pnpm

export CODEX_HOME := $(PWD)/.codex

codex-locale:
	@echo "CODEX_HOME=$(CODEX_HOME)"
	@echo "Running codex with CODEX_HOME=$(CODEX_HOME)"
	codex -m gpt-5.3-codex -c model_reasoning_effort="xhigh" -c model_reasoning_summary_format=experimental --search --dangerously-bypass-approvals-and-sandbox

codex-locale-resume:
	@echo "CODEX_HOME=$(CODEX_HOME)"
	@echo "Running codex with CODEX_HOME=$(CODEX_HOME)"
	codex -m gpt-5.3-codex -c model_reasoning_effort="xhigh" -c model_reasoning_summary_format=experimental --search --dangerously-bypass-approvals-and-sandbox resume 

