# 28. ApplicationSet - Deploy Multiple Applications with the List Generator

## Introduction

An **ApplicationSet** is a Kubernetes custom resource and controller that
generates one or more Argo CD **Applications** from a template. Instead of
writing and maintaining one Application manifest per app, you write one
ApplicationSet and let the controller render the Applications.

The **list generator** is the simplest generator: you provide an explicit list
of key/value maps (`elements`) in the manifest. The ApplicationSet controller
iterates over the list and renders the template once per element. Every key in
an element becomes a template parameter.

**Why it exists**

- Removes copy/paste duplication when onboarding many similar applications.
- Makes bulk changes (sync policy, project, labels, destination) a single edit.
- Keeps generated Applications consistent and declaratively managed.
- Automatically creates and deletes Applications as the list changes.

**When to use it**

- A fixed, known set of applications (for example a platform bundle).
- Environments or tenants that differ only by name, namespace, cluster, or
  chart values.
- Bootstrapping a set of apps before moving to a dynamic generator (git,
  cluster, pull request).

**When not to use it**

- The set of apps already lives in Git as directories or files: use the git
  generator instead.
- The set depends on clusters: use the cluster generator.
- The set must react to pull requests: use the pull request generator.

## Implementation

### 1. Prepare the cluster

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Confirm the ApplicationSet controller is running:

```bash
kubectl -n argocd get deploy argocd-applicationset-controller
```

The examples deploy three public Helm charts into the local cluster. You can
replace them with charts from your own Helm repository.

### 2. Create the ApplicationSet

Create `appsets/list-generator.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: demo-apps
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - list:
        elements:
          - appName: nginx
            namespace: nginx
            cluster: https://kubernetes.default.svc
            repoURL: https://charts.bitnami.com/bitnami
            chart: nginx
            chartVersion: 18.2.4
            
  template:
    metadata:
      name: '{{.appName}}'
      labels:
        managed-by: applicationset
        generator: list
    spec:
      project: default
      source:
        repoURL: '{{.repoURL}}'
        chart: '{{.chart}}'
        targetRevision: '{{.chartVersion}}'
        helm:
          releaseName: '{{.appName}}'
      destination:
        server: '{{.cluster}}'
        namespace: '{{.namespace}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

Notes:

- `elements` can contain any keys. Each key is available in the template as
  `{{.key}}`.
- `goTemplate: true` enables Go templating. The syntax is `{{.appName}}`, not
  `{{appName}}`.
- `missingkey=error` makes the controller fail loudly when the template
  references a key that is missing from an element.
- `releaseName` prevents Helm release name collisions.
- `CreateNamespace=true` lets Argo CD create each destination namespace.
- Do not set both `chart` and `path`; a source is either a Helm repository chart
  or a path in a Git repository.

Adjust chart versions to versions that exist in the repository, or use a valid
semver range such as `targetRevision: "18.x.x"`.

### 3. Apply the ApplicationSet

```bash
kubectl apply -f appsets/list-generator.yaml
```

The controller creates one Application per list element:

```bash
kubectl -n argocd get applications
argocd app list
```

Expected names: `nginx`, `redis`, and `postgresql`.

### 4. Inspect a generated Application

```bash
argocd app get nginx
kubectl -n argocd get application nginx -o yaml
```

Check that the generated Application contains the values rendered from the list
element:

- `metadata.name` = `nginx`
- `spec.destination.namespace` = `nginx`
- `spec.source.chart` = `nginx`
- `spec.source.targetRevision` = `18.2.4`

### 5. Sync and verify

If you kept `syncPolicy.automated`, sync happens automatically. Otherwise:

```bash
argocd app sync nginx redis postgresql
argocd app wait nginx redis postgresql --health
kubectl get pods -n nginx
kubectl get pods -n redis
kubectl get pods -n postgresql
```

### 6. Change the set and observe reconciliation

Add a fourth element or change `chartVersion`, then re-apply:

```bash
kubectl apply -f appsets/list-generator.yaml
kubectl -n argocd get applications -w
```

If you remove an element, the controller deletes the corresponding Application.
With automated sync and `prune: true`, the Application's resources are removed
too.

### 7. Clean up

```bash
kubectl delete -f appsets/list-generator.yaml
```

Deleting the ApplicationSet deletes the Applications it owns (via owner
references) and, by default, their resources.

## Common pitfalls

- Using `{{appName}}` instead of `{{.appName}}` while `goTemplate: true` is set.
- Referencing a key in the template that an element does not define. Either
  define it for every element or remove `missingkey=error`.
- Forgetting `helm.releaseName` and hitting release-name collisions.
- Setting `helm.valueFiles` for a remote Helm repository chart. `valueFiles`
  paths are relative to a chart stored in Git; use `helm.values` or
  `helm.parameters` for repository charts.
- Expecting Argo CD to create the destination namespace without
  `CreateNamespace=true`.

## What to remember for CGOA

- List generator = static list of elements, each element rendered once.
- Parameters are arbitrary keys; use Go template syntax with `goTemplate: true`.
- One ApplicationSet produces N Applications named by the template.
- Generated Applications are normal Applications and can be inspected and
  synced exactly like hand-written ones.
