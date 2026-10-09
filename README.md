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
