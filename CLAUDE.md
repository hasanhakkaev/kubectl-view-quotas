# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make bin           # fmt + vet + build to bin/kubectl-view-quotas

# Test
make test          # runs tests with coverage (go test ./pkg/... ./cmd/...)

# Format & Vet
make fmt
make vet

# Lint (via pre-commit)
golangci-lint run
```

Run a single test:
```bash
go test ./pkg/plugin/... -run TestFunctionName -v
```

## Architecture

This is a `kubectl` plugin (installed via krew) that lists Kubernetes `ResourceQuota` objects with color-coded usage percentages.

**Entry point flow:**
1. `cmd/plugin/main.go` → `cli.InitAndExecute()`
2. `cmd/plugin/cli/root.go` — Cobra command setup; wires `genericclioptions.ConfigFlags` (standard kubectl flags) and the `--all-namespaces`/`-A` flag; calls `plugin.RunPlugin()`
3. `pkg/plugin/plugin.go` — Core logic: builds a k8s clientset from kubeconfig, lists `ResourceQuota` objects in the target namespace(s), and prints a colored table via `uitable` + `cfmt`

**Color coding** in `chooseColour()`: usage percentage maps to a gradient from white (0–10%) through green shades, then yellow/orange/red at higher thresholds (100% = `#FF0000`).

## Linting

golangci-lint is configured in `.golangci.yaml`. Active linters include `gofumpt`, `goimports`, `gosec`, `revive`, `gocritic`, `errcheck`, and others. Tag naming rules: YAML fields use `goCamel`, JSON fields use `snake_case`. `ioutil.*` usage is forbidden (use `io` / `os` equivalents).

Pre-commit hook runs golangci-lint automatically (`.pre-commit-config.yaml`).

## Release

`.goreleaser.yml` handles cross-platform builds. `.krew.yaml` defines the krew plugin manifest for distribution via `kubectl krew`.
