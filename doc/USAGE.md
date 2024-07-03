
## Usage
The following assumes you have the plugin installed via

```shell
kubectl krew install kubectl-view-quotas
```

### View quotas for your current namespace

```shell
kubectl view-quotas
```

### View quotas in another namespace

```shell
kubectl view-quotas -n namespace_name
```

### View all quotas

```shell
kubectl view-quotas -A 
```