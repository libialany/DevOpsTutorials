# Trivy Security Scanning & Custom Image Patching Tutorial


## Critical Strategies for Ubuntu 24.04 & Alpine 3.20.3

### 1. Fast Scan Commands (Before Building)

```bash
# Scan local image (fast - uses cache)
trivy image --skip-update --ignore-unfixed ubuntu:24.04
trivy image --skip-update --ignore-unfixed alpine:3.20.3

# Scan with severity filter (CRITICAL/HIGH only to save time)
trivy image --severity HIGH,CRITICAL --ignore-unfixed ubuntu:24.04

# Generate report (JSON for CI/CD)
trivy image --format json --output report.json ubuntu:24.04
```

### 2. OpenSSL Vulnerability Patching Tricks

**Trick #1: Layered patching in Dockerfile**
```dockerfile
# Ubuntu 24.04
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    openssl=3.0.13-0ubuntu3.1 \  # Pin specific secure version
    && rm -rf /var/lib/apt/lists/*

# Alpine 3.20.3  
FROM alpine:3.20.3

RUN apk add --no-cache openssl=3.2.2-r0  # Check Alpine repos for patched version
```

**Trick #2: Multi-stage patching**
```dockerfile
# Builder stage - scan and patch
FROM ubuntu:24.04 AS builder
RUN apt-get update && apt-get install -y openssl=3.0.13-0ubuntu3.1

# Final stage - minimal patched image
FROM ubuntu:24.04
COPY --from=builder /usr/bin/openssl /usr/bin/openssl
COPY --from=builder /usr/lib/x86_64-linux-gnu/libssl.so.3 /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libcrypto.so.3 /usr/lib/x86_64-linux-gnu/
```

**Trick #3: Post-scan remediation**
```bash
# Scan and auto-fix (Alpine only - uses apk upgrade)
trivy image --severity HIGH,CRITICAL --auto-fix alpine:3.20.3

# For Ubuntu, scan first to get exact package versions
trivy image --format table --output vulnerabilities.txt ubuntu:24.04
# Then manually pin versions in Dockerfile
```

### 3. DevOps Time-Savers

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

### 4. Docker Compose Integration

```yaml
version: '3.8'
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.ubuntu  # Use patched base
    # Pre-build validation
    init: true
    security_opt:
      - no-new-privileges:true

  # Separate service for scanning
  trivy-scan:
    image: aquasec/trivy:latest
    volumes:
      - .:/app
    command: trivy image --severity HIGH,CRITICAL --exit-code 1 app:latest
```

### 5. Verification Commands

```bash
# Verify patched OpenSSL version
docker run --rm ubuntu:24.04 openssl version -a
docker run --rm alpine:3.20.3 openssl version -a

# Re-scan after patching
trivy image --ignore-unfixed --severity HIGH,CRITICAL your-image:latest

# Check for remaining vulnerabilities
trivy image --format table your-image:latest | grep -i openssl
```

### Key OpenSSL Versions to Target

| Distribution | Target Version | Command to Verify |
|-------------|---------------|-------------------|
| Ubuntu 24.04 | 3.0.13-0ubuntu3.1+ | `apt list --installed \| grep openssl` |
| Alpine 3.20.3 | 3.2.2-r0+ | `apk list --installed \| grep openssl` |

## Pro Tips

1. **Always scan before building** - Prevents wasting time on broken images
2. **Pin exact versions** - Don't use `latest` tags in production
3. **Use `--ignore-unfixed`** - Ignores vulnerabilities with no available patch
4. **Scan during CI/CD** - Add `--exit-code 1` to fail builds on critical vulns
5. **Minimize attack surface** - Remove package managers from final images
6. **Alpine advantage** - Smaller surface area, easier to patch

## Quick Reference

```bash
# One-liner: Scan, patch, rebuild loop
trivy image --severity CRITICAL --exit-code 1 ubuntu:24.04 || \
(echo "Critical found, patching..." && docker build --build-arg OPENSSL_VER=3.0.13-0ubuntu3.1 -t patched .)

# Check all images in docker-compose
for img in $(grep -o "image:.*" docker-compose.yml | cut -d: -f2-); do 
  trivy image --severity HIGH,CRITICAL $img; 
done
```
