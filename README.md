# kubectl-view-quotas

A `kubectl` plugin to view resource quotas with color-coded usage bars.

## Quick Start

```
kubectl krew install view-quotas
kubectl view-quotas
```

## Usage

```
# Current namespace
kubectl view-quotas

# Specific namespace
kubectl view-quotas -n <namespace>

# All namespaces
kubectl view-quotas -A
```

## Example Output

```
Name:      compute-resource-quota
Namespace: my-namespace

╭─────────────────┬────────┬──────┬───────────────────╮
│ RESOURCE        │ USED   │ HARD │ USAGE             │
├─────────────────┼────────┼──────┼───────────────────┤
│ requests.cpu    │ 5350m  │ 30   │ █▊          17.8% │
│ requests.memory │ 6292Mi │ 50Gi │ █▏          12.3% │
╰─────────────────┴────────┴──────┴───────────────────╯

Name:      object-resource-quota
Namespace: my-namespace

╭────────────────────────┬──────┬──────┬───────────────────╮
│ RESOURCE               │ USED │ HARD │ USAGE             │
├────────────────────────┼──────┼──────┼───────────────────┤
│ configmaps             │ 5    │ 10   │ █████       50.0% │
│ persistentvolumeclaims │ 4    │ 5    │ ████████    80.0% │
│ pods                   │ 15   │ 30   │ █████       50.0% │
│ secrets                │ 7    │ 10   │ ███████     70.0% │
│ services               │ 14   │ 20   │ ███████     70.0% │
╰────────────────────────┴──────┴──────┴───────────────────╯
```

Usage bars are color-coded: green (< 75%), yellow (75–90%), red (90–100%), magenta (> 100%).
