# Set up image scanning in the pipeline

## simple web server 

```
FROM nginx:alpine

RUN RUN apk update && apk upgrade --no-cache && echo '<h1>Hello from private AWS ECR!</h1>' \
    > /usr/share/nginx/html/index.html

EXPOSE 80
```

build the image


```
docker build -t my-app:v1 .
```

login the ecr repository

```
aws ecr get-login-password \
  --region "$AWS_REGION" \
  | docker login \
      --username AWS \
      --password-stdin "$ECR_REGISTRY"
```
 
push the image to ecr repository

```
docker tag my-app:v1 "$IMAGE_URI"
docker push "$IMAGE_URI"
```

## Configure the scan on the image 

[complete file](.github/workflows/snyk.yml)

```
      # 5. Set up Snyk CLI
      - name: Set up Snyk CLI
        uses: snyk/actions/setup@master

        # 6. Scan container image
      - name: Snyk Container Scan
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        run: |
          snyk container test \
            "${ECR_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}" \
            --severity-threshold=high \
            --project-name="${IMAGE_NAME}"
```
