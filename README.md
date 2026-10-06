## Alpine vs Debian base image vulnerability comparison

### resources

[comparisons](https://safeguard.sh/resources/blog/alpine-vs-debian-base-image-vulnerability-comparison)

## Trivy Tips


```bash
# Skip DB updates (faster scans in CI)
trivy image --skip-update ubuntu:24.04

# Cache scanning (reuse results)
trivy image --cache-dir /tmp/trivy-cache ubuntu:24.04

# Scan only specific vulnerabilities (OpenSSL CVEs)
trivy image --vuln-type os --severity HIGH,CRITICAL --ignore-unfixed ubuntu:24.04

# Check image before docker-compose up
trivy image --exit-code 1 --severity CRITICAL ubuntu:24.04 && docker-compose up
```