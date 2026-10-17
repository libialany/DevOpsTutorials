## Applications in ArgoCD


Deploy an application with ArgoCD using private repositories, and how to auth to that repo is pass the credentials using this command:

```bash
argocd repo add https://github.com/YOUR_ORG/argocd-config.git \
  --username YOUR_USER --password YOUR_TOKEN
```

or:

```bash
argocd repo add https://github.com/YOUR_ORG/my-app-manifests.git \
  --username YOUR_USER --password YOUR_TOKEN
```

## Define the demo-app Application

```yaml
# apps/demo-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: main
    path: manifests
  destination:
    server: https://kubernetes.default.svc
    namespace: demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

```bash
kubectl apply -f root-app.yaml
```

ArgoCD syncs it automatically.
