# Basic CI/CD with Argo CD

## Introduction

A basic GitOps CI/CD pipeline separates **continuous integration** from
**continuous deployment**:

- **CI** builds and tests the application, produces an immutable image, and
  writes the new image reference into a configuration repository.
- **CD** is performed by Argo CD, which watches the configuration repository and
  reconciles the cluster to match it.

The image tag is the commit SHA, so every deployment is traceable to an exact
source commit.

**Why it exists**

- Keeps application and configuration ownership separate.
- Prevents CI from holding cluster credentials; CI only writes to Git.
- Standard application delivery to one or more long-lived environments.

## Implementation

### 1. Create the two repositories

Webapp repository:

```
webapp/
├── .github/workflows/ci.yaml
├── Dockerfile
├── package.json
└── src/
```

Config repository:

```
webapp-config/
└── charts/webapp/
    ├── Chart.yaml
    ├── values.yaml
    ├── values-dev.yaml
    └── templates/
        ├── deployment.yaml
        └── service.yaml
```

### 2. Create the Argo CD Application

Create `applications/webapp-dev.yaml` in the config repository (or apply it
directly with `kubectl`):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: webapp-dev
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/<your-org>/webapp-config.git
    targetRevision: main
    path: charts/webapp
    helm:
      releaseName: webapp
      valueFiles:
        - values.yaml
        - values-dev.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: webapp-dev
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

Apply it:

```bash
kubectl apply -f applications/webapp-dev.yaml
argocd app get webapp-dev
argocd app sync webapp-dev
argocd app wait webapp-dev --health
```

At this point the app deploys whatever tag is in `values.yaml` (initially
`latest`). The pipeline will start changing it to commit SHAs.

### 3. Build the CI pipeline

Create `.github/workflows/ci.yaml` in the **webapp** repository:

```yaml
name: ci

on:
  push:
    branches: [main]

permissions:
  contents: read
  packages: write

env:
  IMAGE: ghcr.io/<your-org>/webapp
  CONFIG_REPO: <your-org>/webapp-config
  CHART_VALUES: charts/webapp/values.yaml

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - run: npm ci
      - run: npm test

  build-and-push:
    needs: build-and-test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push image tagged with commit SHA
        run: |
          docker build -t "$IMAGE:${GITHUB_SHA}" .
          docker push "$IMAGE:${GITHUB_SHA}"

  update-config:
    needs: build-and-push
    runs-on: ubuntu-latest
    steps:
      - name: Checkout config repository
        uses: actions/checkout@v4
        with:
          repository: ${{ env.CONFIG_REPO }}
          ref: main
          token: ${{ secrets.CONFIG_REPO_TOKEN }}
          path: config
      - name: Update image tag
        working-directory: config
        run: |
          yq -i '.image.repository = strenv(IMAGE) | .image.tag = strenv(GITHUB_SHA)' "$CHART_VALUES"
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add "$CHART_VALUES"
          git commit -m "chore(dev): webapp ${GITHUB_SHA}"
          git push
```

Verify the running image:

```bash
kubectl -n webapp-dev get deploy webapp \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
```

The tag must match the webapp commit SHA that triggered the pipeline.

