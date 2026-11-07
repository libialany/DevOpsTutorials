# 30. ApplicationSet - Git Directory Generator

## Introduction

The **git generator** scans a Git repository and generates Applications from what
it finds there. It has two modes:

- **Files generator** - one Application per JSON/YAML file.
- **Directory generator** - one Application per directory matching a glob.

This exercise uses the **directory generator**. The generator clones the
repository, finds directories matching patterns such as `apps/*`, and exposes the
directory path as parameters. The Application template uses that path as both the
source path and, in this exercise, the Application name and namespace.

**Why it exists**

- The repository layout becomes the source of truth for the application list.
- Adding a new app is as simple as adding a directory containing a Helm chart or
  manifests; no ApplicationSet edit is required.
- Removes the need to keep a list generator in sync with the repository.
- Enables self-service: a team adds a directory and the platform automatically
  deploys it.

### How the directory generator works

The generator performs these steps on every reconciliation:

1. Clones/fetches the repository at the requested revision.
2. Expands the `directories` patterns (`*` matches within one segment, `**`
   matches across segments).
3. Applies `exclude` patterns if present.
4. Renders the template once per matching directory.

Available parameters include:

| Parameter | Example for `apps/payments-api` | Notes |
|-----------|-------------------------------|-------|
| `path.path` | `apps/payments-api` | Repository-relative directory |
| `path.basename` | `payments-api` | Last path segment |
| `path.basenameNormalized` | `payments-api` | Safe for Kubernetes names |
| `path.segments` | `[apps, payments-api]` | List of segments |
| `path[0]`, `path[1]` | `apps`, `payments-api` | Indexed access |

Use `basenameNormalized` when directory names may contain characters that are
invalid in Kubernetes resource names (underscores, dots, uppercase letters).

## Implementation

### 1. Create the repository layout and app charts

#### `apps/nginx/Chart.yaml`

```yaml
apiVersion: v2
name: nginx
description: Minimal application chart used by the git directory generator lab
type: application
version: 0.1.0
appVersion: "1.27"
```

#### `apps/nginx/values.yaml`

```yaml
replicaCount: 1

image:
  repository: nginx
  tag: "1.27-alpine"
  pullPolicy: IfNotPresent

containerPort: 80

service:
  type: ClusterIP
  port: 80

env: []

resources: {}
```

#### `apps/nginx/templates/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  labels:
    app.kubernetes.io/name: {{ .Chart.Name }}
    app.kubernetes.io/instance: {{ .Release.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ .Chart.Name }}
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ .Chart.Name }}
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: tcp
              containerPort: {{ .Values.containerPort }}
              protocol: TCP
          {{- with .Values.env }}
          env:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}
```


Validate the charts before pushing:

```bash
helm lint apps/nginx
helm template nginx apps/nginx
```

Push the repository and note its URL and branch.

### 2. Create the ApplicationSet

Create `appsets/git-directory.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: git-directory-apps
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - git:
        repoURL: https://github.com/<your-org>/<your-repo>.git
        revision: HEAD
        directories:
          - path: apps/*
        # Add exclusions with:
        #   - path: apps/experimental/*
        #     exclude: true
        # The directory generator cannot test file contents, so keep the
        # repository layout predictable instead.
  template:
    metadata:
      name: '{{.path.basename}}'
      labels:
        managed-by: applicationset
        generator: git-directory
      finalizers:
        - resources-finalizer.argocd.argoproj.io
    spec:
      project: default
      source:
        repoURL: https://github.com/<your-org>/<your-repo>.git
        targetRevision: HEAD
        path: '{{.path}}'
        helm:
          releaseName: '{{.path.basenameNormalized}}'
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{.path.basename}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

A real exclusion looks like:

```yaml
directories:
  - path: apps/*
  - path: apps/experimental/*
    exclude: true
```

The repository URL in the generator and in the template must match. Use a
read-only deploy key or a repository that Argo CD can clone anonymously.
