# 31. ApplicationSet - Matrix Generator

## Introduction

The **matrix generator** combines two or more child generators and produces the
**cartesian product** of their results. If the first generator yields 3
applications (from Git directories) and the second yields 4 clusters, the matrix
generator yields 12 parameter sets, and therefore 12 Applications.

Conceptually:

```
matrix = generatorA x generatorB
```

Each generated parameter set contains the union of the parameters from both
generators. Template fields can therefore reference values from either side, for
example `{{.path.basename}}` from a git generator and `{{.name}}` /
`{{.server}}` from a cluster generator.

**Why it exists**

- Deploy every app to every cluster without writing N x M manifests.
- Express fleet-wide rollout patterns (all platform add-ons to all clusters).

**When to use it**

- You have a set of applications defined in Git and a set of target clusters
  defined in Argo CD.
- You want to layer selection: for example only `env=prod` clusters and only
  `apps/*` directories.

**When not to use it**

- The two lists are the same length and should be paired one-to-one (app A to
  cluster A, app B to cluster B). A matrix produces all combinations; use a list
  generator with explicit pairs, or a merge generator.

### Matrix vs merge generator

- **Matrix** produces the cartesian product (N x M).
- **Merge** joins results by a common key, producing N rows enriched with data
  from a second generator (a "left join"). Use merge when cluster metadata should
  be attached to a specific app rather than multiplied across all apps.

## Implementation

### 1. Register clusters with labels

```bash
argocd cluster add kind-dev  --name dev  --label env=dev  -y
argocd cluster add kind-prod --name prod --label env=prod -y
argocd cluster list
```

For this exercise, target all registered clusters (including `in-cluster` if
present). If you want only the remote clusters, label them with a common label
and select it.

### 2. Create the repository layout

```
charts-repo/
└── apps/
    ├── nginx/
    │   ├── Chart.yaml
    │   ├── values.yaml
    │   └── templates/
    ├── redis/
    │   ├── Chart.yaml
    │   ├── values.yaml
    │   └── templates/
    └── postgresql/
        ├── Chart.yaml
        ├── values.yaml
        └── templates/
```

Each directory is a Helm chart. Push the repository.

### 3. Create the matrix ApplicationSet

Create `appsets/matrix.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: matrix-apps
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - matrix:
        generators:
          - clusters:
              selector:
                matchLabels:
                  env: dev
          - git:
              repoURL: https://github.com/<your-org>/<your-repo>.git
              revision: HEAD
              directories:
                - path: apps/*
  template:
    metadata:
      name: '{{.path.basename}}-{{.name}}'
      labels:
        managed-by: applicationset
        generator: matrix
        app: '{{.path.basename}}'
        cluster: '{{.name}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/<your-org>/<your-repo>.git
        targetRevision: HEAD
        path: '{{.path}}'
        helm:
          releaseName: '{{.path.basenameNormalized}}'
      destination:
        server: '{{.server}}'
        namespace: '{{.path.basename}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

The child generator order is not important for the result, but it is conventional
to put clusters first and the Git directory generator second.

```bash
kubectl apply -f appsets/matrix.yaml
kubectl -n argocd get applications
argocd app sync nginx-dev
argocd app wait nginx-dev --health
```

