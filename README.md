# ApplicationSet - Pull Request Generator

## Introduction

The **pull request generator** asks an SCM provider (GitHub, GitLab, Gitea,
Bitbucket Server, Azure DevOps) for open pull requests and generates an
Application for each matching PR. The Application usually deploys a preview
environment in a unique namespace, built from the PR's branch or commit.

This is the "ephemeral preview environment" pattern: every pull request gets its
own running copy of the application, and the environment disappears when the PR
is merged or closed.

**Why it exists**

- Reviewers can test a change in a real cluster before merge.
- Environments are created and destroyed automatically, avoiding manual
  namespace and release cleanup.

### How the pieces fit together

1. A developer opens a pull request with the label `preview`.
2. A CI pipeline builds and tests the code, pushes an image tagged with the
   commit SHA, writes that tag into the Helm chart, and adds the `preview` label.
3. The pull request generator lists open PRs carrying the `preview` label.
4. For each eligible PR, Argo CD creates an Application named `webapp-pr-<n>`
   that deploys the chart from the PR head commit into namespace `pr-<n>`.
5. When the PR is merged or closed, the PR disappears from the generator, the
   Application is deleted, and its finalizer removes the resources.

## Implementation

### 1. Prepare the webapp repository

Layout:

```
webapp/
├── .github/workflows/preview.yaml
├── app/                
    ├── webapp.go
    └── Dockerfile
├── helm/                
    ├── values.yaml
    ├── Chart.yaml
    └── templates/
        ├── deployment.yaml
        └── service.yaml
```

### 2. Enable SCM providers in Argo CD

The ApplicationSet controller must be allowed to talk to SCM APIs. Check the
controller arguments:

```bash
kubectl -n argocd get deploy argocd-applicationset-controller \
  -o jsonpath='{.spec.template.spec.containers[0].args}'
```
Ensure `--enable-scm-providers` is present. If it is missing, add it with:

### 3. Create the SCM token Secret

Create a GitHub token with read access to the repository (and `repo` scope for a
private repository), then:

```bash
kubectl -n argocd create secret generic github-token \
  --from-literal=token="$GITHUB_TOKEN"
```

The token needs to list pull requests. For public repositories a read-only
token is enough; do not reuse a token with broader permissions than necessary.

### 4. Create the ApplicationSet

Create `appsets/pull-request.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: webapp-preview
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - pullRequest:
        github:
          owner: <your-org>
          repo: webapp
          tokenRef:
            secretName: github-token
            key: token
          labels:
            - preview
        requeueAfterSeconds: 60
  template:
    metadata:
      name: 'webapp-pr-{{.number}}'
      labels:
        managed-by: applicationset
        generator: pull-request
        preview: "true"
      annotations:
        preview.example.com/pr: '{{.number}}'
        preview.example.com/branch: '{{.branch}}'
        preview.example.com/sha: '{{.head_sha}}'
      finalizers:
        - resources-finalizer.argocd.argoproj.io
    spec:
      project: default
      source:
        repoURL: https://github.com/<your-org>/webapp.git
        targetRevision: '{{.head_sha}}'
        path: charts/webapp
        helm:
          releaseName: 'webapp-pr-{{.number}}'
      destination:
        server: https://kubernetes.default.svc
        namespace: 'pr-{{.number}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

```bash
kubectl apply -f appsets/pull-request.yaml
argocd appset get webapp-preview
```

Create `.github/workflows/preview.yaml` in the webapp repository:

```yaml
name: preview

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: write
  packages: write
  pull-requests: write

env:
  IMAGE: ghcr.io/<your-org>/webapp
  CHART_VALUES: charts/webapp/values.yaml

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - run: npm ci
      - run: npm test

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push image with commit SHA tag
        run: |
          docker build -t "$IMAGE:${GITHUB_SHA}" .
          docker push "$IMAGE:${GITHUB_SHA}"

  update-chart:
    needs: build-and-push
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.head_ref }}
          token: ${{ secrets.GITHUB_TOKEN }}
      - name: Update image tag in Helm chart
        run: |
          yq -i '.image.tag = strenv(GITHUB_SHA)' "$CHART_VALUES"
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add "$CHART_VALUES"
          git commit -m "chore: update preview image to ${GITHUB_SHA} [skip ci]"
          git push

  add-preview-label:
    needs: update-chart
    runs-on: ubuntu-latest
    steps:
      - name: Add preview label to the pull request
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
        run: gh pr edit "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --add-label preview
```
