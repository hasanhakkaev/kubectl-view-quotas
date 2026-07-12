# kubectl-view-quotas

## What This Is

A `kubectl` plugin (distributed via krew) that lists Kubernetes ResourceQuota objects in one or all namespaces and prints a colored table of used/hard/usage-percentage per resource.

## Core Value

At-a-glance visibility of quota consumption — a cluster user should immediately see which quotas are near or at their limit.

## Requirements

### Validated

- ✓ List ResourceQuotas in a namespace or all namespaces (`-A`) — v0.0.1
- ✓ Colored usage bar/percentage per resource — v0.0.1
- ✓ Krew distribution with automated manifest updates — v0.0.1

### Active

- [ ] Object quotas with `hard: 0` (forbidden resources) are displayed, not silently skipped

### Out of Scope

- JSON/YAML output — no user demand yet
- Watch mode — plugin is a one-shot reporting tool

## Context

Go CLI using cobra + k8s.io/client-go, table rendering via uitable + cfmt. Commit 70bfb42 introduced "skip resources with hard=0", which breaks object quotas whose limits are intentionally 0 (e.g. `persistentvolumeclaims: 0` to forbid PVCs) — those quotas now render as empty tables.

## Constraints

- **Tech stack**: Go, cobra, client-go — established, no new deps for small fixes
- **Lint**: golangci-lint (gofumpt, gosec, revive, …) must pass; pre-commit enforces it

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Skip hard=0 resources (70bfb42) | Avoid `<none>` noise | ⚠️ Revisit — hides legitimate zero-quotas |

---
*Last updated: 2026-07-12 after bootstrap for quick task 1*
