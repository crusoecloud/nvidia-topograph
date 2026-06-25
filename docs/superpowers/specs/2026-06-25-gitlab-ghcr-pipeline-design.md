# GitLab CI → GHCR publish pipeline for the nvidia-topograph fork

Date: 2026-06-25
Status: Approved (pending spec review)

## Problem

Crusoe maintains a fork of `NVIDIA/topograph`, with GitLab
(`gitlab.com/crusoeenergy/island/external/nvidia-topograph`) as the CI/publishing
source of truth. The fork has rarely been published, so it has no `.gitlab-ci.yml`.
We need to (re)publish a **multi-arch** container image — including an `arm64`
variant — to GHCR at `ghcr.io/crusoecloud/nvidia-topograph`.

Security has already provisioned two GitLab CI/CD variables:
- `GHCR_TOKEN` — a GitHub PAT with permission to push packages to the existing repo.
- `GHCR_USERNAME` — the username the PAT was generated from.

The upstream GitHub Actions workflow (`.github/workflows/docker.yml`) does not run on
the GitLab mirror, so its multi-arch build must be reproduced in GitLab CI.

## Goals

- Build a single multi-arch manifest (`linux/amd64` + `linux/arm64`).
- Publish to `ghcr.io/crusoecloud/nvidia-topograph` using `GHCR_USERNAME`/`GHCR_TOKEN`.
- Keep publishing intentional (manual), matching the rarely-published nature of the fork.
- Catch a broken image build on every push/MR, before anyone publishes.

## Non-goals

- Replicating the Go unit-test / lint CI (those remain GitHub-side concerns for now).
- Auto-incrementing versions or writing git tags from CI. Versions are set by a human
  tagging the commit (`vX.Y.Z-crusoe.N`) before running the publish pipeline.
- Publishing the `Dockerfile.ib` InfiniBand image or Helm charts (out of scope here).

## Design

### 1. Build strategy — Go cross-compile, no QEMU

The `Dockerfile` already cross-compiles via `make build-${TARGETOS}-${TARGETARCH}`,
producing pure-static binaries (`CGO_ENABLED=0`, `-extldflags "-static"`).

Single one-line change so the toolchain stage always runs natively on the amd64 runner
and Go cross-compiles per target arch:

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.24.7 AS builder
```

`buildx` passes `TARGETOS`/`TARGETARCH` per target platform. The final `alpine:3` stage
is target-arch but only performs `COPY` (no `RUN`), so no emulation executes. Result:
both arches build fast on a standard `docker` runner — no QEMU, no scarce native arm64
runner.

### 2. Pipeline shape — two stages

`stages: [build, publish]`

- **`build_image`** (stage `build`) — validation only. Runs on push and merge_request
  events. Builds `--platform linux/amd64,linux/arm64` with **no `--push`** (export
  disabled) to verify the Dockerfile compiles for both arches. Fast feedback, no
  registry writes.

- **`publish_ghcr`** (stage `publish`) — builds the multi-arch image and `--push`es a
  single manifest to GHCR. Runs **only** when triggered manually on the default branch:
  `rules: $CI_PIPELINE_SOURCE == "web" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH`.
  Uses `resource_group: publish-ghcr` to serialize concurrent publishes.

### 3. Versioning & tags

The `publish_ghcr` job resolves the version from the manually-set git tag:

- `GIT_DEPTH: 0` on the job (full history) plus `git fetch --tags` as belt-and-suspenders,
  so tags are present in the CI checkout.
- `VERSION=$(git describe --tags)` (e.g. `v0.1.0-crusoe.2`).
- Push two tags: `:$VERSION` and `:latest`.

No `major`/`minor` moving tags — the `crusoe.N` suffix is prerelease-style and those
would be ambiguous.

Publish workflow:
1. `git tag v0.1.0-crusoe.N` on the commit to release; push to GitLab.
2. CI/CD → Pipelines → **Run Pipeline** on `main`.
3. Image is published as `ghcr.io/crusoecloud/nvidia-topograph:v0.1.0-crusoe.N` and `:latest`.

### 4. Runner / engine

Mirrors the slinky-containers setup that already works against this org's runners:

- `image: docker:28-cli`, service `docker:28-dind`.
- `DOCKER_DRIVER: overlay2`, `DOCKER_BUILDKIT: "1"`.
- `apk add --no-cache git` (needed for `git describe`).
- `docker buildx create --use` (docker-container driver) so multi-platform `--push` works.
- Login: `echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin`.
- `tags: [docker]` runner tag.

## Files touched

- `Dockerfile` — add `--platform=$BUILDPLATFORM` to the builder `FROM` line.
- `.gitlab-ci.yml` — new file (build + publish stages described above).

## Verification

- Push the branch → the `build_image` validation job builds both arches without pushing.
- Manually run a pipeline on `main` with a `crusoe.N` tag present → confirm the image and
  `:latest` appear at the GHCR package, and `docker buildx imagetools inspect` shows both
  `linux/amd64` and `linux/arm64` in the manifest list.
