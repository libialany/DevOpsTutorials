# App of Apps - Directory of Apps Approach

## Introduction

**App of Apps** is a pattern where one Argo CD Application (the **root app**)
deploys other Argo CD Applications (the **child apps**), and those child apps in
turn deploy workloads. The root app is only responsible for the Application
resources; it does not deploy the workloads directly.

In the **directory of apps** approach, the child Application manifests are plain
YAML files stored in a directory tree in Git. The root app points at that
directory and syncs it. Argo CD applies every Application manifest it finds.

**Why it exists**

- Bootstraps a whole platform from a single Application.
- Gives a single place to see and control a group of applications.

**When to use it**

- You want a root app that installs a fixed set of Applications from plain
  manifests.
- Child apps have different sources and should remain independent.

### Root app vs child app

| | Root app | Child app |
|---|----------|-----------|
| Source | Directory of Application manifests | Workload manifests or Helm chart |
| Destination namespace | `argocd` | Workload namespace |
| Deploys | Application CRs | Deployments, Services, etc. |
| Sync policy | Usually manual or automated | Set per child (manual in this exercise) |

Deleting the root app does not automatically delete child apps unless Git no
longer contains them and the root app syncs with prune, or you delete the child
apps explicitly. This is an important operational difference from
ApplicationSets, where generated Applications are owned by the ApplicationSet.

## Implementation

### 1. Create the directory structure

In a Git repository:

```
argocd-apps/
├── root.yaml
└── apps/
    ├── dev/
    │   ├── frontend.yaml
    │   └── backend.yaml
    └── prod/
        └── frontend.yaml
```

The root app source path is `apps`, so every Application manifest under `apps`
and its subdirectories is applied. Subdirectories require
`directory.recurse: true`.

### 2. Create the child Application manifests

`apps/dev/frontend.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: frontend-dev
  namespace: argocd
  labels:
    env: dev
    tier: frontend
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: frontend-dev
  syncPolicy:
    syncOptions:
      - CreateNamespace=true
```

`apps/dev/backend.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: backend-dev
  namespace: argocd
  labels:
    env: dev
    tier: backend
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: helm-guestbook
    helm:
      valueFiles:
        - values.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: backend-dev
  syncPolicy:
    syncOptions:
      - CreateNamespace=true
```

Notice that the child apps have **no `automated` sync policy**. They are synced
manually, as required by the exercise. They may still be refreshed automatically,
but no deployment happens until someone syncs them.

Each child app should have `CreateNamespace=true` if the destination namespace
does not exist.

### 3. Create the root Application

`root.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: root
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/<your-org>/argocd-apps.git
    targetRevision: main
    path: apps
    directory:
      recurse: true
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    syncOptions:
      - CreateNamespace=true
```

Key points:

- `path: apps` points at the directory of child Applications.
- `directory.recurse: true` includes `apps/dev`, `apps/prod`, and deeper.
- The destination is the Argo CD namespace because that is where child
  Application CRs live.
- The root app has no `automated` policy, so you explicitly sync it.
- Do not add `prune: true` until you understand which Application CRs would be
  removed when a file disappears.


## Comparison with ApplicationSet

| Property | App of Apps (directory) | ApplicationSet |
|----------|-------------------------|----------------|
| Child app manifests | Stored in Git as files | Rendered from a template |
| Adding an app | Add a file and sync root | Add generator input (list, directory, PR) |
| Ownership | Root app applies CRs; no generator owns them | ApplicationSet owns generated Applications |
| Removal | Git delete + prune sync | Automatic when generator stops producing the app |
| Best for | Hand-crafted, heterogeneous apps | Repetitive or dynamic apps |

