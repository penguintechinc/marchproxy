# MarchProxy Application Standards

## Container Image Exceptions

The following container images deviate from the standard Debian 12 (bookworm) base image requirement. These exceptions are approved for specific hardware-acceleration use cases.

| Dockerfile | Image | Reason | Approved |
|------------|-------|--------|---------|
| `proxy-rtmp/Dockerfile.amd` | `rocm/dev-ubuntu-22.04:6.0` | AMD ROCm GPU runtime required for hardware video transcoding; no Debian equivalent | ✅ |
| `proxy-rtmp/Dockerfile.nvidia` | `nvidia/cuda:12.3.1-runtime-ubuntu22.04` | NVIDIA CUDA runtime required for GPU-accelerated RTMP transcoding; no Debian equivalent | ✅ |

All other containers MUST use Debian 12 (bookworm) based images. Ubuntu is only acceptable where the required GPU/CUDA runtime has no Debian equivalent.

---

## Standard Container Images

All MarchProxy services use the following approved base images unless an exception is documented above:

- **Python**: `python:3.13-slim-bookworm`
- **Go**: `golang:1.24-bookworm`
- **Node.js**: `node:18-bookworm-slim`
- **Nginx**: `nginx:stable-bookworm-slim`
- **Debian runtime**: `debian:bookworm-slim`
