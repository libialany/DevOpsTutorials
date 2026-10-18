## ArgoCD with Helm

### Why Helm Charts with Argo CD and GitOps Matter

1. **Reuse instead of rewriting.** Public charts (like Bitnami's NGINX) package a Deployment, Service, and more, already tested by the community. You deploy with a few lines of YAML instead of maintaining dozens of manifests.

2. **Configuration lives in Git as values.** Instead of editing templates, you override only what you need (replicas, service type, image) in `values`. Every change is a Git commit, so you get history, review, and easy rollback.

3. **Argo CD makes Helm declarative and drift-proof.** Argo CD renders the chart itself (using `helm template`), compares it to the cluster, and with auto-sync and self-heal it keeps the cluster matching Git. No more manual `helm upgrade` commands, and no hidden state.

### Create the Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nginx-helm
  namespace: argocd
#helm show chart oci://registry-1.docker.io/bitnamicharts/nginx
spec:
  project: default
  source:
    repoURL: registry-1.docker.io/bitnamicharts   # OCI registry (no oci:// prefix)
    chart: nginx
    targetRevision: 18.2.0                        # chart version: pin it!
    helm:
      releaseName: nginx
      valuesObject:
        replicaCount: 2
        service:
          type: ClusterIP
  destination:
    server: https://kubernetes.default.svc
    namespace: nginx-demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

```bash
argocd repo add registry-1.docker.io/bitnamicharts --type helm --name bitnami-oci --enable-oci
```

### Apply

```bash
kubectl apply -f nginx-helm-app.yaml

argocd app get nginx-helm
kubectl get pods -n nginx-demo
kubectl get svc -n nginx-demo
```

### Test

```bash
kubectl port-forward svc/nginx -n nginx-demo 8081:80
curl http://localhost:8081
```

### Pass values thougth values

```yaml
spec:
  sources:
    - repoURL: registry-1.docker.io/bitnamicharts
      chart: nginx
      targetRevision: 18.2.0
      helm:
        releaseName: nginx
        valueFiles:
          - $values/nginx/values.yaml
    - repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
      targetRevision: main
      ref: values                    # makes this repo available as $values
```

`nginx/values.yaml` in your Git repo:

```yaml
replicaCount: 3
service:
  type: ClusterIP
```

```bash
kubectl scale deployment nginx -n nginx-demo --replicas=10
# Argo CD reverts it to the value in Git
```

