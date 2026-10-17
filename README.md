# ArgoCD Tracking Tag, SHA, and HEAD

## Why it matters 

`targetRevision` tells ArgoCD **which point in Git** to deploy. The same Application can track a moving branch, a fixed tag, or an exact commit.

IN CASE OF HEAD: **Fast feedback in dev.** Every merge is deployed automatically, so developers see their changes live in minutes without extra steps. **Best for** dev and test environments.

IN CASE OF Tag: Best for staging and production releases.

IN CASE OF SHA: **Fully immutable.** A commit SHA can never change, so what you deploy is exactly what you reviewed and tested.
**Best for** production with strict change control, or debugging a specific version.

## 1. Create the points

In the **my-app-manifests** repo:

```bash
## TAG 
git tag v1.0.0
git push origin v1.0.0

# Commit SHA
git rev-parse HEAD
```

## 2. Apply 

### A) HEAD (branch tracking)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
....
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: HEAD          # or a branch name like main
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### B) Tag

```yaml
apiVersion: argoproj.io/v1alpha1
............
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: v1.0.0        # git tag
    path: manifests
...........
  syncPolicy:
...........
    syncOptions:
      - CreateNamespace=true
```

### C) SHA

```yaml
apiVersion: argoproj.io/v1alpha1
..............
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: 8f3c2a1d5e7b9c4a6f0e1d2c3b4a5968778695a4   # full commit SHA
...............
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

