# ArgoCD AppProject

## Create the appProject

An **AppProject** limits which repos an Application can pull from, which clusters and namespaces it can deploy to, and which resource kinds it can create.

Example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: demo-project
  namespace: argocd
spec:
  description: Project restricted to the demo namespace

  # Only allow this source repo
  sourceRepos:
    - https://github.com/YOUR_ORG/my-app-manifests.git

  # Only allow deploying to the demo namespace of the local cluster
  destinations:
    - server: https://kubernetes.default.svc
      namespace: demo

  # Namespaced resources allowed (empty = allow all kinds inside demo)
  namespaceResourceWhitelist:
    - group: "apps"
      kind: Deployment
    - group: ""
      kind: Service
    - group: ""
      kind: ConfigMap

  # Block cluster-wide resources (Namespace, ClusterRole, CRDs, etc.)
  clusterResourceWhitelist: []
```

## 2. Important: `CreateNamespace=true` needs a small change

Because `clusterResourceWhitelist` is empty, ArgoCD **cannot create the `demo` namespace** (Namespace is a cluster-scoped resource). You have two options.

Create the Namespace kind at cluster level:

```yaml
  clusterResourceWhitelist:
    - group: ""
      kind: Namespace
```

## 3. Use the project in your Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo-app
  namespace: argocd
spec:
  project: demo-project ## Here
..
```

## 4. Test That It Blocks Other Namespaces

Change the Application destination to a different namespace:

```yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: production     # change and watch
```

ArgoCD will refuse because:


```
  destinations:
    - server: https://kubernetes.default.svc
      namespace: demo
```

## Notes

- **The `default` project allows everything.** Avoid using it in production, and create restricted projects like this one.
- **Order matters with App-of-Apps:** the AppProject must exist before the Application that uses it. Use sync waves to be safe:

