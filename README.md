# GitOps with Flux CD and Kustomize: Dev + Staging on Two Local kind Clusters

This tutorial builds a complete multi-environment GitOps setup from scratch on your laptop:

- **2 kind clusters**: `dev` and `staging`.
- **1 Flux configuration repo** (`flux-infra`): tells each cluster *what to watch*
- **1 application repo** (`podinfo-demo`): holds the Kubernetes manifests, with a Kustomize **base** and one **overlay per environment**
- **A demo app**: [podinfo](https://github.com/stefanprodan/podinfo), a tiny web app made for Kubernetes demos (one container, one port, a nice UI that shows its version, color and message)

> **Why podinfo instead of instavote?** Instavote has 5 services (vote, redis, worker, db, result). That is great later, but podinfo lets you focus on Flux and Kustomize first. The structure below works identically for instavote: just replace the manifests in `deploy/`.

---

## 1. The big picture

```
┌─────────────────────┐        ┌──────────────────────┐
│  flux-infra (repo)  │        │ podinfo-demo (repo)  │
│                     │        │                      │
│ clusters/           │        │ deploy/podinfo/      │
│   dev/      ────────┼──┐     │   base/              │
│   staging/  ────────┼─┐│     │   overlays/dev/      │
└─────────────────────┘ ││     │   overlays/staging/  │
                        ││     └──────────▲───────────┘
   Flux (dev cluster)   ││                │ GitRepository + Kustomization
   reads clusters/dev ◄─┘│                │ (defined in flux-infra)
   Flux (staging)        │                │
   reads clusters/staging◄┘───────────────┘
```

Two repositories, two different jobs:

| Repo | Contains | Who reads it |
|---|---|---|
| `flux-infra` | Flux's own components (`flux-system/`) plus **pointers** to apps: a `GitRepository` ("where is the code?") and a `Flux Kustomization` ("which folder, how often?") | Flux, when bootstrapped with `--path=clusters/<env>` |
| `podinfo-demo` | The app's manifests: `base/` plus `overlays/dev` and `overlays/staging` | Flux, via the pointers above |

> **Two things are called "Kustomization".** Don't mix them up:
> - `kustomize.config.k8s.io/v1beta1` `Kustomization` is the **`kustomization.yaml`** file used by the Kustomize tool (it lists resources, patches, images, ...).
> - `kustomize.toolkit.fluxcd.io/v1` `Kustomization` is a **Flux CRD** that says "apply the Kustomize output of this path from this Git source to this cluster".
>
> Flux's Kustomization *runs* Kustomize's `kustomization.yaml`.

---

## 2. Prerequisites

Install these tools (Docker must be running):

```bash
# kind, kubectl, flux, git, docker
kind version
kubectl version --client
flux --version
git --version
```

Install hints:

```bash
# macOS / Linux with Homebrew
brew install kind kubectl fluxcd/tap/flux
# Flux alternative
curl -s https://fluxcd.io/install.sh | sudo bash
```

You also need a **GitHub account** (this tutorial uses GitHub; GitLab works with `flux bootstrap gitlab`).

### Create a GitHub Personal Access Token (PAT)

Flux uses it **once**, during bootstrap, to create the repo and register the deploy key.

- Classic token: scope `repo`
- Fine-grained token: Administration (RW) and Contents (RW) on the repo

```bash
export GITHUB_USER=<your-github-username>
export GITHUB_TOKEN=<your-personal-access-token>
```

> Do not commit the token. Flux does not store it in the cluster; it stores only the deploy key.

---

## 3. Create the two kind clusters

```bash
kind create cluster --name dev
kind create cluster --name staging
```

kind prefixes context names with `kind-`:

```bash
kubectl config get-contexts
# kind-cluster-1
# kind-cluster-2
```

Always be explicit about which cluster you talk to, to avoid applying things to the wrong one:

```bash
kubectl --context kind-cluster-1 get nodes
kubectl --context kind-cluster-2 get nodes
```

Run the Flux pre-flight check against each cluster:

```bash
flux check --pre --context kind-cluster-1
flux check --pre --context kind-cluster-2
```

---

## 4. Create the application repo (`podinfo-demo`)

Create an empty **public** repo on GitHub named `podinfo-demo` (public avoids needing credentials for the app repo; Section 9 shows the private variant), then:

```bash
git clone https://github.com/$GITHUB_USER/podinfo-demo.git
cd podinfo-demo
mkdir -p deploy/podinfo/base deploy/podinfo/overlays/dev deploy/podinfo/overlays/staging
```

### 4.1 The base: what is common to every environment

`deploy/podinfo/base/namespace.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: podinfo
```

`deploy/podinfo/base/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: podinfo
  labels:
    app: podinfo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: podinfo
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
        - name: podinfo
          image: ghcr.io/stefanprodan/podinfo:6.5.0
          ports:
            - name: http
              containerPort: 9898
          envFrom:
            - configMapRef:
                name: podinfo-config
          readinessProbe:
            httpGet:
              path: /readyz
              port: http
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
          resources:
            requests:
              cpu: 25m
              memory: 32Mi
            limits:
              memory: 64Mi
```

`deploy/podinfo/base/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: podinfo
spec:
  selector:
    app: podinfo
  ports:
    - name: http
      port: 9898
      targetPort: http
```

`deploy/podinfo/base/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: podinfo
resources:
  - namespace.yaml
  - deployment.yaml
  - service.yaml

configMapGenerator:
  - name: podinfo-config
    literals:
      - PODINFO_UI_COLOR=#34577c
      - PODINFO_UI_MESSAGE=Hello from BASE
```

### 4.2 Preview what Kustomize renders

```bash
kubectl kustomize deploy/podinfo/base
# or: kustomize build deploy/podinfo/base
```

Notice the ConfigMap is called `podinfo-config-<random hash>`. When its content changes, the hash changes, the Deployment's reference changes, and Kubernetes performs a **rolling restart automatically**. This is a major benefit of generators.

### 4.3 The dev overlay

`deploy/podinfo/overlays/dev/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
images:
  - name: ghcr.io/stefanprodan/podinfo
    newTag: 6.5.4
replicas:
  - name: podinfo
    count: 1
configMapGenerator:
  - name: podinfo-config
    behavior: merge
    literals:
      - PODINFO_UI_COLOR=#2e8b57
      - PODINFO_UI_MESSAGE=Hello from DEV
labels:
  - pairs:
      environment: dev
    includeSelectors: false
```

### 4.4 The staging overlay

`deploy/podinfo/overlays/staging/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

images:
  - name: ghcr.io/stefanprodan/podinfo
    newTag: 6.5.0
replicas:
  - name: podinfo
    count: 2
configMapGenerator:
  - name: podinfo-config
    behavior: merge
    literals:
      - PODINFO_UI_COLOR=#c9731a
      - PODINFO_UI_MESSAGE=Hello from STAGING
labels:
  - pairs:
      environment: staging
    includeSelectors: false
patches:
  - target:
      kind: Deployment
      name: podinfo
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/0/resources/limits/memory
        value: 128Mi
```

### 4.5 Preview each environment and compare

```bash
kubectl kustomize deploy/podinfo/overlays/dev
kubectl kustomize deploy/podinfo/overlays/staging

# see exactly what differs
diff <(kubectl kustomize deploy/podinfo/overlays/dev) \
     <(kubectl kustomize deploy/podinfo/overlays/staging)
```

### 4.6 Kustomize cheat sheet (the key primitives)

| Field | Purpose |
|---|---|
| `resources` | Files, folders or remote bases to include. Overlays reference `../../base` here |
| `namespace` | Force a namespace on all resources |
| `namePrefix` / `nameSuffix` | Add prefix/suffix to all resource names |
| `labels` / `commonAnnotations` | Add labels/annotations to everything |
| `images` | Change image name, tag or digest without editing the manifest |
| `replicas` | Change replica counts |
| `configMapGenerator` / `secretGenerator` | Create ConfigMaps/Secrets from literals or files. Content hash in name triggers rollouts. `behavior: create \| merge \| replace` |
| `patches` | Strategic-merge or JSON6902 patches targeted by kind/name/labels |
| `components` | Reusable optional features you can mix into overlays |

Commit and push:

```bash
git add .
git commit -m "Add podinfo base and dev/staging overlays"
git push origin main
```

---

## 5. Bootstrap Flux on the DEV cluster

"Bootstrapping" does three things in one command:

1. **Installs the Flux (GitOps Toolkit) components** into the cluster namespace `flux-system`: source-controller, kustomize-controller, helm-controller, notification-controller, plus CRDs, RBAC and NetworkPolicies.
2. **Creates or uses a Git repository** (`flux-infra`) and commits the Flux manifests into it, at the path you choose.
3. **Creates a read-only deploy key** so Flux can pull from that repo, and stores the private half in a Secret in the cluster. Flux then manages itself from Git ("self-managing").

```bash
flux bootstrap github \
  --context=kind-cluster-1 \
  --owner=$GITHUB_USER \
  --repository=flux-infra \
  --branch=main \
  --path=clusters/dev \
  --personal \
  --log-level=debug \
  --network-policy=false \
  --components=source-controller,kustomize-controller

```

What the flags mean:

| Flag | Meaning |
|---|---|
| `--context` | Which cluster to install into |
| `--owner` | GitHub user (or org) |
| `--repository` | Created if it doesn't exist |
| `--path` | **The folder this cluster watches.** Dev watches `clusters/dev` only |
| `--personal` | Owner is a user account, not an organization |

> By default the deploy key is **read-only**. Add `--read-write-key` only if you plan to use image automation, which pushes commits back.

Verify:

```bash
flux check --context kind-cluster-1
kubectl --context kind-cluster-1 -n flux-system get pods
flux --context kind-cluster-1 get sources git
flux --context kind-cluster-1 get kustomizations
```

Now pull the new repo locally:

```bash
cd ..
git clone https://github.com/$GITHUB_USER/flux-infra.git
cd flux-infra
tree clusters
# clusters/
# └── dev
#     └── flux-system
#         ├── gotk-components.yaml   <- Flux controllers + CRDs
#         ├── gotk-sync.yaml         <- GitRepository + Kustomization that sync this very repo
#         └── kustomization.yaml
```

---

## 6. Tell the DEV cluster about the app

Flux needs two objects per app: **where** (GitRepository) and **what/how** (Kustomization).

```bash
cd clusters/dev

# WHERE: the app repo
flux create source git podinfo \
  --url=https://github.com/$GITHUB_USER/podinfo-demo \
  --branch=main \
  --interval=1m \
  --export > podinfo-gitrepository.yaml

# WHAT: the dev overlay folder
flux create kustomization podinfo-dev \
  --source=GitRepository/podinfo \
  --path="./deploy/podinfo/overlays/dev" \
  --prune=true \
  --interval=5m \
  --wait=true \
  --export > podinfo-dev-kustomization.yaml
```

> **`flux create ... --export` vs `flux export`**
> - `flux create ... --export` **generates** YAML without touching any cluster. Use it to author files before they exist.
> - `flux export source git <name>` / `flux export kustomization <name>` **dumps live objects** from a cluster as YAML. Use it to capture something that was created manually (e.g. `flux export source git podinfo >> podinfo-gitrepository.yaml`).
>
> Both end with the same files in `flux-infra/clusters/dev`.

The generated files look like this:

`clusters/dev/podinfo-gitrepository.yaml`

```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 1m0s
  ref:
    branch: main
  url: https://github.com/<you>/podinfo-demo
```

`clusters/dev/podinfo-dev-kustomization.yaml`

```yaml
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: podinfo-dev
  namespace: flux-system
spec:
  interval: 5m0s
  path: ./deploy/podinfo/overlays/dev
  prune: true
  wait: true
  sourceRef:
    kind: GitRepository
    name: podinfo
```

Key fields:

- `interval`: how often Flux re-checks/re-applies.
- `prune: true`: if you delete a manifest from Git, Flux deletes it from the cluster.
- `wait: true`: Flux waits for resources to become Ready before marking success.

Push (Flux itself watches this repo, so you never run `kubectl apply`):

```bash
cd ../..
git add -A
git commit -m "dev: deploy podinfo"
git push

# Optional: skip the wait
flux --context kind-cluster-1 reconcile source git flux-system
```

Verify:

```bash
flux --context kind-cluster-1 get kustomizations --watch
kubectl --context kind-cluster-1 -n podinfo get deploy,svc,pods
kubectl --context kind-cluster-1 -n podinfo port-forward svc/podinfo 9898:9898
# open http://localhost:9898  -> "Hello from DEV", green, 1 replica
```

---

## 7. Bootstrap Flux on the STAGING cluster

Requirements recap: a new Kubernetes cluster (`kind-cluster-2`), the existing `flux-infra` repo, and `GITHUB_USER` / `GITHUB_TOKEN` exported.

Same repo, same branch, **different `--path`**:

```bash
flux bootstrap github \
  --context=kind-cluster-2 \
  --owner=$GITHUB_USER \
  --repository=flux-infra \
  --branch=main \
  --path=clusters/staging \
  --personal
```

Because the repo already exists, Flux reuses it and creates a separate deploy key for this cluster.

Pull the changes Flux just committed, then add the staging app definitions:

```bash
cd flux-infra
git pull
mkdir -p clusters/staging && cd clusters/staging

flux create source git podinfo \
  --url=https://github.com/$GITHUB_USER/podinfo-demo \
  --branch=main \
  --interval=1m \
  --export > podinfo-gitrepository.yaml

flux create kustomization podinfo-staging \
  --source=GitRepository/podinfo \
  --path="./deploy/podinfo/overlays/staging" \
  --prune=true \
  --interval=5m \
  --wait=true \
  --export > podinfo-staging-kustomization.yaml

cd ../..
git add -A
git commit -m "staging: deploy podinfo"
git push
```

Verify:

```bash
flux --context kind-cluster-2 get kustomizations
kubectl --context kind-cluster-2 -n podinfo get deploy,pods     # 2 replicas
kubectl --context kind-cluster-2 -n podinfo port-forward svc/podinfo 9899:9898
# open http://localhost:9899 -> "Hello from STAGING", orange, v6.5.0
```

Final layout of `flux-infra`:

```
flux-infra/
└── clusters/
    ├── dev/
    │   ├── flux-system/                      # generated by bootstrap
    │   ├── podinfo-gitrepository.yaml
    │   └── podinfo-dev-kustomization.yaml
    └── staging/
        ├── flux-system/                      # generated by bootstrap
        ├── podinfo-gitrepository.yaml
        └── podinfo-staging-kustomization.yaml
```

---

## 8. Day-2: the GitOps workflow in action

### Change something in dev

In `podinfo-demo`, edit `overlays/dev/kustomization.yaml`:

```yaml
      - PODINFO_UI_MESSAGE=Dev says hi again
```

```bash
git commit -am "dev: new message" && git push
```

Within about 1 minute Flux detects the new commit, renders the overlay, sees a **new ConfigMap hash**, and rolls the pods. Staging is untouched.

### Promote to staging

```bash
# in overlays/staging/kustomization.yaml
images:
  - name: ghcr.io/stefanprodan/podinfo
    newTag: 6.5.4
```

```bash
git commit -am "staging: promote podinfo 6.5.4" && git push
```

Promotion is just a pull request that copies a tested value from `dev` to `staging`. Git history is your audit log, and `git revert` is your rollback.

### Handy commands

```bash
flux get all -A --context kind-cluster-1        # status of everything
flux logs --level=error --context kind-cluster-1
flux reconcile kustomization podinfo-dev --with-source --context kind-cluster-1
flux suspend kustomization podinfo-dev --context kind-cluster-1   # pause (for debugging)
flux resume  kustomization podinfo-dev --context kind-cluster-1
flux diff kustomization podinfo-dev --path ./deploy/podinfo/overlays/dev --context kind-cluster-1
```

### Prove self-healing

```bash
kubectl --context kind-cluster-1 -n podinfo scale deploy podinfo --replicas=5
# wait for the next reconcile (or run `flux reconcile ...`): Flux sets it back to 1
```

---

## 9. Optional: private app repo

If `podinfo-demo` is private, create a read-only deploy key / credentials secret per cluster:

```bash
flux create secret git podinfo-auth \
  --url=ssh://git@github.com/$GITHUB_USER/podinfo-demo \
  --context kind-cluster-1
# prints a public key -> add it as a read-only Deploy Key in the podinfo-demo repo settings
```

Then reference it in the GitRepository:

```yaml
spec:
  url: ssh://git@github.com/<you>/podinfo-demo
  secretRef:
    name: podinfo-auth
```

---

## 10. Troubleshooting

| Symptom | Check |
|---|---|
| `flux bootstrap` fails with 403/404 | Token scope, `GITHUB_USER` set, correct `--personal` vs org |
| Kustomization shows `kustomization path not found` | The `--path` must match the folder in the **app repo** exactly and be pushed |
| `kustomize build failed` | Run `kubectl kustomize <path>` locally to reproduce |
| Stuck on "Reconciliation in progress" with `wait: true` | A pod is not Ready: `kubectl -n podinfo describe pod` |
| Nothing updates after push | `flux reconcile source git podinfo`, then `flux get sources git` |
| Wrong cluster modified | Always pass `--context kind-cluster-1` / `kind-cluster-2` |

Debug order that solves 90% of cases:

```bash
flux get sources git -A
flux get kustomizations -A
kubectl -n flux-system describe kustomization podinfo-dev
flux logs --kind=Kustomization --name=podinfo-dev
```

---

## 11. Clean up

```bash
kind delete cluster --name dev
kind delete cluster --name staging
# optionally delete the two GitHub repos and revoke the token
```

---

## 12. Where to go next

- Swap podinfo for **instavote** (put each service under `base/`, same overlay approach).
- Add `dependsOn` between Flux Kustomizations (e.g. infra before apps).
- Use **image automation** (`--read-write-key`) to auto-bump tags in the dev overlay.
- Encrypt secrets in Git with **SOPS + age** (Flux supports `decryption:` natively).
- Add a `production` cluster by repeating Section 7 with `--path=clusters/production`.

