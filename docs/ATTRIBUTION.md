# MarchProxy Attribution

This document lists all open-source dependencies and libraries used in MarchProxy, along with their licenses and purposes.

## Python Dependencies (Manager & API Server)

| Library | License | Purpose |
|---------|---------|---------|
| py4web | BSD 3-Clause | Web framework for manager server |
| pydal | BSD 3-Clause | Database abstraction layer |
| psycopg2-binary | LGPL v3 | PostgreSQL adapter |
| redis | BSD 3-Clause | Caching and session management |
| bcrypt | Apache 2.0 | Password hashing |
| PyJWT | MIT | JWT token handling |
| pyotp | MIT | Two-factor authentication (TOTP/HOTP) |
| qrcode | BSD 3-Clause | QR code generation for 2FA |
| python-dotenv | BSD 3-Clause | Environment variable management |
| pydantic | MIT | Data validation |
| httpx | BSD 3-Clause | HTTP client library |
| uvicorn | BSD 3-Clause | ASGI application server |
| gunicorn | MIT | Python WSGI application server |
| python-saml | MIT | SAML 2.0 authentication |
| python-jose | MIT | JSON Web Signature/Encryption |
| authlib | BSD 3-Clause | OAuth/OIDC authentication |
| fastapi | MIT | Modern Python web framework |
| jinja2 | BSD 3-Clause | Template engine |
| aiofiles | Apache 2.0 | Async file I/O |
| python-multipart | MIT | Multipart form data parsing |
| cryptography | Apache 2.0 & BSD 3-Clause | Cryptographic operations |
| certifi | Mozilla Public License 2.0 | CA certificate bundle |
| prometheus-client | Apache 2.0 | Prometheus metrics |
| python-json-logger | BSD 3-Clause | JSON logging |
| sentry-sdk | BSD 2-Clause | Error tracking and monitoring |
| click | BSD 3-Clause | CLI framework |
| PyYAML | MIT | YAML parsing |
| pendulum | MIT | Date/time handling |
| validators | MIT | Data validation library |
| sqlalchemy | MIT | SQL toolkit and ORM |
| alembic | MIT | Database schema migrations |
| asyncpg | Apache 2.0 | PostgreSQL async driver |
| passlib | BSD 3-Clause | Password hashing library |
| pydantic-settings | MIT | Settings management |
| email-validator | CC0 (Public Domain) | Email validation |
| opentelemetry-api | Apache 2.0 | Observability API |
| opentelemetry-sdk | Apache 2.0 | Observability SDK |
| opentelemetry-instrumentation-fastapi | Apache 2.0 | FastAPI instrumentation |
| grpcio | Apache 2.0 | gRPC protocol implementation |
| grpcio-tools | Apache 2.0 | gRPC code generation tools |

## JavaScript/Node.js Dependencies (Web UI)

| Library | License | Purpose |
|---------|---------|---------|
| react | MIT | JavaScript UI framework |
| react-dom | MIT | React DOM rendering |
| react-router-dom | MIT | Client-side routing |
| react-hook-form | MIT | Form state management |
| react-query | MIT | Server state management |
| react-simple-maps | MIT | React mapping component |
| recharts | MIT | Charting library |
| reactflow | MIT | Node-based UI library |
| @mui/material | MIT | Material Design components |
| @mui/icons-material | MIT | Material Design icons |
| @mui/x-data-grid | MIT | Advanced data grid |
| @mui/x-date-pickers | MIT | Date/time picker components |
| @emotion/react | MIT | CSS-in-JS styling |
| @emotion/styled | MIT | Styled components |
| @monaco-editor/react | MIT | Monaco editor component |
| axios | MIT | HTTP client |
| zustand | MIT | State management |
| d3-geo | BSD 3-Clause | D3 geospatial utilities |
| date-fns | MIT | Date utility library |
| vite | MIT | Build tool and dev server |
| typescript | Apache 2.0 | TypeScript compiler |
| eslint | MIT | JavaScript linter |
| @typescript-eslint/eslint-plugin | BSD 2-Clause | TypeScript linting |
| @typescript-eslint/parser | BSD 2-Clause | TypeScript parser |
| @vitejs/plugin-react | MIT | React plugin for Vite |
| eslint-plugin-react-hooks | MIT | React Hooks linting |
| eslint-plugin-react-refresh | MIT | React Refresh linting |
| terser | BSD 2-Clause | JavaScript minifier |
| @playwright/test | Apache 2.0 | End-to-end testing |
| express | MIT | Express.js server framework |

## Go Dependencies (Proxy & API Server)

| Library | License | Purpose |
|---------|---------|---------|
| google.golang.org/grpc | Apache 2.0 | gRPC implementation |
| google.golang.org/protobuf | BSD 3-Clause | Protocol Buffers |
| github.com/andybalholm/brotli | MIT | Brotli compression |
| github.com/go-redis/redis/v8 | BSD 2-Clause | Redis client |
| github.com/golang-jwt/jwt/v4 | MIT | JWT handling |
| github.com/golang-jwt/jwt/v5 | MIT | JWT handling (v5) |
| github.com/gorilla/mux | BSD 3-Clause | HTTP router |
| github.com/klauspost/compress | BSD 3-Clause | Data compression |
| github.com/prometheus/client_golang | Apache 2.0 | Prometheus client |
| github.com/prometheus/client_model | Apache 2.0 | Prometheus data model |
| github.com/quic-go/quic-go | MIT | QUIC protocol |
| github.com/sirupsen/logrus | MIT | Structured logging |
| github.com/spf13/cobra | Apache 2.0 | CLI framework |
| github.com/spf13/viper | MIT | Configuration management |
| go.opentelemetry.io/otel | Apache 2.0 | OpenTelemetry SDK |
| go.opentelemetry.io/otel/exporters/stdout/stdouttrace | Apache 2.0 | Telemetry exporter |
| go.opentelemetry.io/otel/sdk | Apache 2.0 | Telemetry SDK |
| go.opentelemetry.io/otel/trace | Apache 2.0 | Trace API |
| golang.org/x/net | BSD 3-Clause | Networking extensions |
| golang.org/x/sys | BSD 3-Clause | System-level primitives |
| golang.org/x/time | BSD 3-Clause | Time utilities |
| golang.org/x/crypto | BSD 3-Clause | Cryptographic packages |
| golang.org/x/sync | BSD 3-Clause | Synchronization primitives |
| golang.org/x/text | BSD 3-Clause | Text handling |
| golang.org/x/mod | BSD 3-Clause | Go module utilities |
| golang.org/x/tools | BSD 3-Clause | Go tools |
| github.com/envoyproxy/go-control-plane | Apache 2.0 | Envoy control plane |
| github.com/envoyproxy/protoc-gen-validate | Apache 2.0 | Protocol validation |
| github.com/cncf/xds | Apache 2.0 | xDS protocol |
| go.uber.org/mock | Apache 2.0 | Mocking framework |
| gopkg.in/yaml.v3 | MIT, Apache 2.0 | YAML parsing |
| gopkg.in/ini.v1 | Apache 2.0 | INI file parsing |

## Project Credits

**MarchProxy** is built and maintained by PenguinTech. This project represents a comprehensive two-container application suite for managing egress traffic in data center environments.

The development of MarchProxy relies on the exceptional work of the open-source community. All dependencies listed above are used in accordance with their respective licenses. We are grateful to all maintainers and contributors of these critical libraries.

For more information about MarchProxy, visit [www.penguintech.io](https://www.penguintech.io)

## License Compliance

MarchProxy is licensed under the Limited AGPL3 with a commercial fair use preamble. All dependencies are compatible with this license. For detailed license information, see the [LICENSE](../LICENSE) file in the project root.

---

Generated: December 2024
Project: MarchProxy v1.x
