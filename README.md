# ArgoCD `ignoreDifferences`

## WHY IT MATTERS?

Many Kubernetes controllers and operators intentionally modify resources after they have been deployed. If Argo CD blindly reverted all these changes, it would constantly fight with these controllers, leading to reconciliation loops and unstable applications.

## Common Use Cases 

- Fields mutated by **admission webhooks** or sidecar injectors (e.g., Istio)
- **`caBundle`** fields injected by cert-manager into webhooks
- **Operator-managed** fields on CRDs
- Defaults added by the API server that Git doesn't declare

## HPA vs Auto-Sync: The Problem

A Deployment in Git says `replicas: 2`. An **HPA** (Horizontal Pod Autoscaler) scales it to 6 under load. ArgoCD sees the live value (6) differs from Git (2) and, with `selfHeal: true`, **scales it back to 2**. The HPA scales up again, ArgoCD reverts it again, and this loop repeats.


## The deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-app
spec:
  replicas: 2                  # fixed in Git: this is what causes the fight
  selector:
    matchLabels:
      app: demo-app
  template:
    metadata:
      labels:
        app: demo-app
    spec:
      containers:
        - name: nginx
          image: nginx:1.27
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 50m         # HPA needs requests to calculate CPU %
              memory: 64Mi
            limits:
              cpu: 100m
              memory: 128Mi
```

HPA Deployment.

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: demo-app
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: demo-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 20     # low threshold, easy to trigger
```

## Application WITHOUT `ignoreDifferences`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo-hpa
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

### Test HPA scales up

```bash
kubectl run load -n demo --image=busybox:1.36 --restart=Never -- \
  /bin/sh -c "while true; do wget -q -O- http://demo-app; done"

kubectl get hpa -n demo -w
```

### Add `ignoreDifferences`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo-hpa
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
      - RespectIgnoreDifferences=true     # also honor the ignore during sync
  ignoreDifferences:
    - group: apps
      kind: Deployment
      name: demo-app
      jsonPointers:
        - /spec/replicas
```

## Why `RespectIgnoreDifferences=true` Matters

By default, `ignoreDifferences` only affects the **diff display** (the Synced/OutOfSync status). When a sync actually runs (for example, triggered by another change), ArgoCD still applies the full manifest from Git, including `replicas: 2`. Adding `RespectIgnoreDifferences=true` makes the sync also skip the ignored fields, so replicas are not reset.

## Other Ways to Specify the Ignored Field

```yaml
ignoreDifferences:
  # JSON Pointer (exact path)
  - group: apps
    kind: Deployment
    jsonPointers:
      - /spec/replicas

  # JQ path expression (more flexible, e.g., filter containers)
  - group: apps
    kind: Deployment
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "nginx") | .resources

  # Managed fields manager (ignore anything changed by a given controller)
  - group: apps
    kind: Deployment
    managedFieldsManagers:
      - kube-controller-manager
```
