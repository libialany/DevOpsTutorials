# App of Apps - Helm Chart Approach

## Introduction

The **Helm chart approach to App of Apps** packages the child Application
manifests as Helm templates. The root Application points at a Helm chart. When
the chart is rendered, it produces Argo CD Application resources. Each rendered
Application then deploys its own workload.

Instead of copying one YAML file per child app, you define a list in
`values.yaml` and a single template that ranges over the list. This keeps the
child app definition


### How it works

1. The root Application is created with `kubectl apply`.
2. The root Application's source is a Helm chart in Git.
3. Argo CD renders the chart and applies the resulting Application CRs.
4. Each child Application has `syncPolicy.automated`, so it immediately syncs its
   own workload.
5. Changing `values.yaml` changes the child Applications on the next root sync.

## Implementation

### 1. Create the app-of-apps Helm chart

```
app-of-apps/
├── Chart.yaml
├── values.yaml
└── templates/
    └── applications.yaml
```

`Chart.yaml`:

```yaml
apiVersion: v2
name: app-of-apps
description: Root chart that renders Argo CD Applications
type: application
version: 0.1.0
appVersion: "1.0"
```

`values.yaml`:

```yaml
defaults:
  project: default
  targetRevision: HEAD
  server: https://kubernetes.default.svc
  autoSync: true

apps:
  - name: guestbook
    namespace: guestbook
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    path: guestbook
  - name: helm-guestbook
    namespace: helm-guestbook
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    path: helm-guestbook
  - name: podinfo
    namespace: podinfo
    repoURL: https://github.com/stefanprodan/podinfo.git
    targetRevision: master
    path: charts/podinfo
```

`templates/applications.yaml`:

```yaml
{{- $defaults := .Values.defaults }}
{{- range .Values.apps }}
{{- $autoSync := $defaults.autoSync }}
{{- if hasKey . "autoSync" }}
{{- $autoSync = .autoSync }}
{{- end }}
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ .name }}
  namespace: argocd
  labels:
    managed-by: app-of-apps
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: {{ default $defaults.project .project }}
  source:
    repoURL: {{ .repoURL }}
    targetRevision: {{ default $defaults.targetRevision .targetRevision }}
    {{- if .chart }}
    chart: {{ .chart }}
    {{- else }}
    path: {{ .path }}
    {{- end }}
    {{- if .helm }}
    helm:
      {{- toYaml .helm | nindent 6 }}
    {{- end }}
  destination:
    server: {{ default $defaults.server .server }}
    namespace: {{ .namespace }}
  syncPolicy:
    {{- if $autoSync }}
    automated:
      prune: true
      selfHeal: true
    {{- end }}
    syncOptions:
      - CreateNamespace=true
{{- end }}
```

Template notes:

- `range` renders one Application per entry in `.Values.apps`.
- `default` lets a per-app value override the chart-wide default.
- `toYaml | nindent` is how structured Helm values are embedded safely.
- The child Applications use `automated` sync with `prune` and `selfHeal`.
- The `resources-finalizer.argocd.argoproj.io` finalizer ensures workloads are
  removed when a child Application is deleted.
- The Application CRs live in the `argocd` namespace, while each workload targets
  its own namespace.

Render the chart locally before committing:

```bash
helm template app-of-apps ./app-of-apps
```

### 2. Create the root Application

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
    repoURL: https://github.com/<your-org>/<your-repo>.git
    targetRevision: main
    path: app-of-apps
    helm:
      releaseName: app-of-apps
      valueFiles:
        - values.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

If the chart is stored in a Helm repository instead of Git, use `chart: app-of-apps`
with the repository URL and a chart version, instead of `path` and
`valueFiles`.


### 3. Add a child app through values

Add an entry to `values.yaml`:

```yaml
  - name: redis
    namespace: redis
    repoURL: https://charts.bitnami.com/bitnami
    chart: redis
    targetRevision: 20.6.2
    helm:
      releaseName: redis
```

The template's `if .chart / else .path` branch supports both a Helm repository
chart and a chart stored in Git, so the same app catalog can mix source types.

Commit, push, and let the root app sync (or run `argocd app sync root`). The new
child Application is rendered and, because it is automated, syncs itself.
