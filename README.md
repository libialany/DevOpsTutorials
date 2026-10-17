## ArgoCD AppProject Roles: Demo with a CI Token

## Why it matters

A **project role** is a set of permissions scoped to one AppProject. You attach policies to the role, then generate a **JWT token** for it. A CI pipeline can use that token instead of the admin password.

## 1. create a rol

in argocd
```
kubectl edit configmap argocd-cm -n argocd
```
and then add:

```
data:
  accounts.developer: apiKey, logins
```

create a password:

```bash
argocd account update-password \
  --account developer
```
assign permissions to the user:

```
kubectl edit configmap argocd-rbac-cm -n argocd
```
in the terminal:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-rbac-cm
  namespace: argocd
data:
  policy.csv: |
    p, proj:app-project:developer, applications, get, app-project/*, allow
```
finally you need to create a token. You must be logged in as admin:

```bash
argocd login localhost:8080 --username admin --password <PASSWORD> --insecure

argocd proj role create-token app-project developer --expires-in 24h
```

## 2. Create a Project


```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: app-project
  namespace: argocd
    ...........................
  roles:
  - name: developer
    description: Sync privileges for app-project
    policies:
    - p, proj:app-project:developer, applications, sync, app-project/*, allow
    - p, proj:app-project:developer, applications, get, app-project/*, allow
```

Format explained:

```
p, <subject>, <resource>, <action>, <object>, <effect>
```

So `developer` can **view** and **sync** apps in `app-project`, and nothing else. It cannot delete, update, or create apps, and it has no access to other projects.


## 3. Create an Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo
  namespace: argocd
spec:
  project: app-project
  source:
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: main
    path: manifests
  destination:
    server: https://kubernetes.default.svc
    namespace: demo
  syncPolicy:
    syncOptions:
      - CreateNamespace=true
```
## 4. Use the token

```bash
# Allowed: get
argocd app get app2 --grpc-web --server localhost:8080 --insecure --auth-token $TOKEN_VALUE

# Allowed: sync
argocd app sync app2 --grpc-web --server localhost:8080 --insecure --auth-token $TOKEN_VALUE
```


