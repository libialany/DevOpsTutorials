## ArgoCD AppProject Roles: Demo with a CI Token

## Why it matters

A **project role** is a set of permissions scoped to one AppProject. You attach policies to the role, then generate a **JWT token** for it. A CI pipeline can use that token instead of the admin password.

## 1. create a rol

```yaml
roles:
- name: ci-role
  description: Sync privileges for project-with-role
  policies:
  - p, proj:project-with-role:ci-role, applications, sync, project-with-role/*, allow
  - p, proj:project-with-role:ci-role, applications, get, project-with-role/*, allow
```

Format:

```
p, <subject>, <resource>, <action>, <object>, <effect>
```

So `ci-role` can **view** and **sync** apps in `project-with-role`, and nothing else. It cannot delete, update, or create apps, and it has no access to other projects.

## 2. Create a Project


```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: project-with-role
  namespace: argocd
    ...........................
  roles:
  - name: ci-role
    description: Sync privileges for project-with-role
    policies:
    - p, proj:project-with-role:ci-role, applications, sync, project-with-role/*, allow
    - p, proj:project-with-role:ci-role, applications, get, project-with-role/*, allow
```

## 3. Create an Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo
  namespace: argocd
spec:
  project: project-with-role #### HERE COMES THE PROJECT
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
## Generate the Token

You must be logged in as admin:

```bash
argocd login localhost:8080 --username admin --password <PASSWORD> --insecure

argocd proj role create-token project-with-role ci-role --expires-in 24h

```

## 4. Use the token

```bash
# Allowed: get
argocd app get demo --grpc-web --server localhost:8080 --insecure --auth-token $TOKEN_VALUE

# Allowed: sync
argocd app sync demo --grpc-web --server localhost:8080 --insecure --auth-token $TOKEN_VALUE
```


