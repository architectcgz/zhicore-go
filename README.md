# zhicore-go

Go migration workspace for the ZhiCore backend.

The current Java backend lives in `../zhicore-microservice`. This repository is the Go landing zone for gradual service replacement. The first goal is to make every target service location explicit, then migrate one service at a time while preserving the external API contracts used by the existing frontend and gateway.

## Service Layout

- `go.work`: local workspace that ties all service and library modules together.
- `services/zhicore-*`: independently buildable, testable, and deployable Go services.
- `services/zhicore-*/go.mod`: each service owns its own Go module.
- `services/zhicore-*/internal`: service-private application, domain, transport, and infrastructure code.
- `libs/contracts`: cross-service client and event contracts.
- `libs/kit`: small shared technical primitives such as response envelopes, auth, config, logging, and infrastructure clients.
- `deploy/`: Docker and Kubernetes deployment manifests.
- `docs/migration/`: migration map and staged replacement notes.

## Target Services

- `zhicore-gateway`
- `zhicore-user`
- `zhicore-content`
- `zhicore-comment`
- `zhicore-message`
- `zhicore-notification`
- `zhicore-search`
- `zhicore-ranking`
- `zhicore-admin`
- `zhicore-upload`
- `zhicore-id-generator`
- `zhicore-ops`

## Commands

```bash
make check
make test
```

`make check` verifies the scaffold and runs Go tests. At this stage most service directories are placeholders; implementation should proceed service by service.
