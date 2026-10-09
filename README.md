## Containers with Image SigninG



## Setup

```bash
docker build -f Dockerfile.good -t ghcr.io/$GH_USER/demo:v3 . && docker push ghcr.io/$GH_USER/demo:v3
DIGEST=$(docker buildx imagetools inspect ghcr.io/$GH_USER/demo:v3 --format '{{json .Manifest.Digest}}' | tr -d '"')
```

## Sign and verify

```bash
cosign generate-key-pair
cosign sign --key cosign.key ghcr.io/$GH_USER/demo@$DIGEST --new-bundle-format=false --use-signing-config=false
cosign verify --key cosign.pub ghcr.io/$GH_USER/demo@$DIGEST
```

## Change the image 

```bash
# Dockerfile now says: echo "I am malware"
docker build -f Dockerfile.bad -t ghcr.io/$GH_USER/demo:v4 . && docker push ghcr.io/$GH_USER/demo:v4
EVIL=$(docker buildx imagetools inspect ghcr.io/$GH_USER/demo:v4 --format '{{json .Manifest.Digest}}' | tr -d '"')
cosign verify --key cosign.pub --insecure-ignore-tlog=true $EVIL
```


# kyverno

Kyverno is a policy engine for Kubernetes that can be used to enforce security policies. One of the ways that Kyverno can be used is to require that all container images deployed to a Kubernetes cluster be signed.

[How to install](https://medium.com/@sddkal/use-cosign-and-kyverno-for-enforcing-image-signing-dff43bc959df)

## Example

1. pull secrets from namespaces:
```
# For Kyverno to fetch the signature
kubectl create secret docker-registry ghcr-creds -n kyverno \
  --docker-server=ghcr.io \
  --docker-username=$GH_USER \
  --docker-password=$READ_PAT

# For the kubelet to pull the image
kubectl create secret docker-registry ghcr-creds -n default \
  --docker-server=ghcr.io \
  --docker-username=$GH_USER \
  --docker-password=$READ_PAT
```

2. Create the sign images policies

[policy](./verify-ghcr-signature.yaml)

```
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-ghcr-signature
  ..............
```

and then 

```
k apply -f policy.yaml
```

3. test it.

```
kubectl run evil --image=ghcr.io/$GH_USER/demo:v<replace> # replace with v3 and v4
```