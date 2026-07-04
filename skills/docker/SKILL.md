---
name: docker
description: Use when creating, reviewing, or editing Dockerfiles, docker-compose files, container image selections, local dependency stacks, or deployment container configuration.
---

# Docker

## Overview

Use this skill to keep Docker assets reproducible and reviewable. Container image choices are dependency decisions, so image tags must be explicit and stable.

## Image Tag Rule

Never use `latest` for any Docker image.

This applies to:

- `FROM ...` in Dockerfiles
- `image: ...` in Compose files
- CLI examples such as `docker run`, `docker pull`, and `docker build --build-arg`
- documentation, README snippets, CI examples, deployment templates, and scripts

## Required Versioning

Use a pinned, visible version tag or digest:

- Prefer exact upstream release tags, such as `postgres:16.3-alpine`, `redis:7.2.5-alpine`, or `docker.elastic.co/elasticsearch/elasticsearch:8.15.3`.
- If the project deliberately tracks a broader line, document the reason next to the image or in the owning deployment doc; do not silently use floating tags.
- A digest pin, such as `image:tag@sha256:...`, is acceptable when supply-chain reproducibility matters.
- If no stable version tag is available, stop and explain the risk instead of substituting `latest`.

## Review Checklist

- Search Docker assets before finishing: `rg -n '(:latest\\b|image:\\s*\\S+latest|FROM\\s+\\S+latest)'`.
- Check generated examples and docs, not just runtime files.
- When replacing `latest`, choose a current stable release tag from the image owner or the project's existing dependency policy.
- State image version changes in the final response because they affect reproducibility and upgrade cadence.

## Common Mistakes

- Do not use `latest` for local-only Compose stacks; local drift still wastes debugging time.
- Do not hide floating tags inside variables such as `IMAGE_TAG=latest`.
- Do not assume a major-only tag is pinned. Treat `redis:7` or `postgres:16` as floating unless the project explicitly accepts that update policy.
