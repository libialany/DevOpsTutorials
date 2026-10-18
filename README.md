# ArgoCD PostSync Hook: Post-Deployment

A **PostSync hook** is a Kubernetes Job that Argo CD runs **after** all resources are synced and **Healthy**. If the Job fails, the sync is marked as failed, so you know the deployment is broken.

Since you don't have an app, we'll use a simple NGINX app plus a check Job that curls it.

## Repo Structure

```
my-app-manifests/
└── postsync-demo/
    ├── deployment.yaml
    ├── service.yaml
    └── postsync-check.yaml
```

## The Application services

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: nginx
          image: nginx:does-not-exist  #nginx:1.27
          ports:
            - containerPort: 80
---

apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  selector:
    app: web
  ports:
    - port: 80
      targetPort: 80
```

## Health Check Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: web-healthcheck
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 2
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: check
          image: curlimages/curl:8.10.1
          command: ["sh", "-c"]
          args:
            - |
              echo "Checking web service..."
              curl -sf --retry 5 --retry-delay 3 --retry-connrefused http://web/ \
                && echo "Health check PASSED" \
                || (echo "Health check FAILED" && exit 1)
```

## The Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: postsync-demo
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR_ORG/my-app-manifests.git
    targetRevision: main
    path: postsync-demo
  destination:
    server: https://kubernetes.default.svc
    namespace: postsync-demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

```bash
kubectl apply -f postsync-app.yaml
argocd app get postsync-demo
kubectl get pods -n postsync-demo -w
```


