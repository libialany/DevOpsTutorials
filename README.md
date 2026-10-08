## Containers with Image SigninG



## Setup
**On screen:** terminal. Type the commands.
```bash
docker build -t localhost:5000/demo:v1 .
docker push localhost:5000/demo:v1
```

## Sign and verify

```bash
cosign generate-key-pair
cosign sign --key cosign.key --tlog-upload=false $DIGEST
cosign verify --key cosign.pub --signing-config signing-config.json  $DIGEST

```
## Change the image 

```bash
# Dockerfile now says: echo "I am malware"
docker build -t localhost:5000/demo:v1 .
docker push localhost:5000/demo:v1
cosign verify --key cosign.pub --insecure-ignore-tlog=true $EVIL
```

###  Enforce it

```bash
cosign verify --key cosign.pub --insecure-ignore-tlog=true $DIGEST \
  && docker run --rm $DIGEST
```