# Migration Map

This file maps the Java ZhiCore modules to their Go landing zones.

## Source Repository

Java source of truth: `../zhicore-microservice`

## Deployable Services

| Java module | Go service module |
| --- | --- |
| `zhicore-gateway` | `services/zhicore-gateway` |
| `zhicore-user` | `services/zhicore-user` |
| `zhicore-content` | `services/zhicore-content` |
| `zhicore-comment` | `services/zhicore-comment` |
| `zhicore-message` | `services/zhicore-message` |
| `zhicore-notification` | `services/zhicore-notification` |
| `zhicore-search` | `services/zhicore-search` |
| `zhicore-ranking` | `services/zhicore-ranking` |
| `zhicore-admin` | `services/zhicore-admin` |
| `zhicore-upload` | `services/zhicore-upload` |
| `zhicore-id-generator` | `services/zhicore-id-generator` |
| `zhicore-ops` | `services/zhicore-ops` |

## Shared Java Modules

| Java module | Go landing zone | Notes |
| --- | --- | --- |
| `zhicore-common` | `libs/kit` | Response envelope, errors, auth, config, persistence, observability, infrastructure primitives. |
| `zhicore-client` | `libs/contracts/clients` | Typed service clients and call contracts. |
| `zhicore-integration` | `libs/contracts/events` | Cross-service event payloads and messaging contracts. |

## Recommended Migration Order

1. `zhicore-id-generator`: smallest HTTP surface and useful for validating Go service deployment.
2. `zhicore-upload`: mostly proxy/integration logic and a bounded API surface.
3. `zhicore-search`: query-heavy service with Elasticsearch and RocketMQ consumers.
4. `zhicore-ranking`: Redis-heavy read and scheduled workload.
5. `zhicore-user`, `zhicore-comment`, `zhicore-content`: core write services with PostgreSQL, Redis, events, and cross-service calls.
6. `zhicore-message`, `zhicore-notification`: WebSocket and push/event fanout services.
7. `zhicore-admin`, `zhicore-ops`, `zhicore-gateway`: migrate after the core service contracts are stable.

## Compatibility Rules

- Preserve existing external API paths and response envelopes until callers are intentionally changed.
- Keep Java and Go services side by side during migration.
- Prefer replacing one service at a time behind the existing gateway or deployment routing.
- Record each migrated endpoint and its validation evidence before removing the Java equivalent.
