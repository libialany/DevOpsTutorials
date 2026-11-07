# ApplicationSet - Cluster Generator

## Introduction

The **cluster generator** produces parameters from the clusters registered in
Argo CD. Instead of listing clusters by hand, the ApplicationSet controller
reads the cluster Secrets managed by Argo CD and renders one Application per
matching cluster.

The generator supports **label selectors**, so the same ApplicationSet can target
all clusters, a group of clusters (for example `env=dev`), or a single cluster.
Each generated Application receives the cluster name, the API server URL, and
the cluster's labels and annotations as template parameters.

**Why it exists**

- Deploys the same application to many clusters without duplicating manifests.
- Uses cluster metadata (labels) as the selection mechanism, matching GitOps
  promotion patterns such as dev, staging, and production.
- Automatically adds Applications when a new cluster is registered and removes
  them when it is deregistered.
- Separates "what to deploy" (the template) from "where to deploy" (cluster
  labels).

**When to use it**

- A platform component or add-on must run on every cluster.
- A workload must run in a subset of clusters chosen by labels.
- You manage a fleet of clusters and want centralized, consistent deployments.

## Implementation

### 1. Register remote clusters

In the Argo CD cluster (the cluster where Argo CD is installed):

```bash
argocd cluster add kind-cluster-2  --name cluster-2 --label env=dev -y
argocd cluster add kind-prod --name prod --label env=prod --label region=eu -y
argocd cluster list
```

To target **only remote clusters**, add a selector on the Secret label that the
local cluster does not have:

```yaml
selector:
  matchLabels:
    argocd.argoproj.io/secret-type: cluster
```

To give the local cluster labels, create a cluster Secret for it, or register it
explicitly on recent versions:

```bash
argocd cluster add <context-name> --in-cluster --name in-cluster --label env=local
```

### 2. Deploy to matching clusters only

Create `appsets/cluster-dev.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: podinfo-dev-only
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - clusters:
        selector:
          matchLabels:
            env: dev
        values:
          logLevel: debug
  template:
    metadata:
      name: 'podinfo-{{.name}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/stefanprodan/podinfo.git
        targetRevision: master
        path: charts/podinfo
        helm:
          releaseName: podinfo
          values: |
            logLevel: {{.values.logLevel}}
      destination:
        server: '{{.server}}'
        namespace: podinfo
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

```bash
kubectl apply -f appsets/cluster-dev.yaml
kubectl -n argocd get applications
```

Only the cluster with label `env=dev` is selected. Use `matchExpressions` for
more advanced selection:

```yaml
selector:
  matchExpressions:
    - key: env
      operator: In
      values: [dev, staging]
    - key: region
      operator: NotIn
      values: [us]
```

### 4. Verify the deployment

```bash
argocd app get podinfo-dev
argocd app sync podinfo-dev
argocd app wait podinfo-dev --health
```


From the Argo CD CLI you can inspect the target without switching:

```bash
argocd app get podinfo-dev --show-params
```
