---
inclusion: manual
---

# Crossplane Provider CI/CD Pipeline Guide

Instructions for setting up CI, vulnerability scanning, and automated releases for Upjet-based Crossplane providers. Derived from the provider-linear implementation.

## Prerequisites

- Go project with a working `go build ./cmd/provider/`
- Dockerfile with multi-stage build (Go builder → distroless runtime)
- `package/crossplane.yaml` with Crossplane provider package metadata
- Angular commit message convention (`feat`, `fix`, `chore`, etc.)

## Go Version Alignment

The Go version must match across three places. A mismatch causes Docker build failures (`go.mod requires go >= X.Y running go Z`).

- `go.mod` — the `go` directive
- `Dockerfile` — the `FROM golang:X.Y` base image
- `.github/workflows/*.yml` — the `GO_VERSION` env var passed to `actions/setup-go`

## GitHub Actions SHA Pinning

Every third-party action must be pinned to a full commit SHA, not a tag. Tags are mutable and can be hijacked (ref: TeamPCP Trivy supply chain attack, March 2026). Format:

```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
```

To find the SHA for a tag, check the repo's tags page or use the GitHub API. Always add the version as a trailing comment for readability.

### Current Latest SHAs (as of April 2026)

| Action | Version | SHA |
|---|---|---|
| `actions/checkout` | v6.0.2 | `de0fac2e4500dabe0009e67214ff5f5447ce83dd` |
| `actions/setup-go` | v6.4.0 | `4a3601121dd01d1626a1e23e37211e3254c1c06c` |
| `actions/upload-artifact` | v7.0.1 | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` |
| `aquasecurity/trivy-action` | v0.35.0 | `57a97c7e7821a5776cebc9bb87c984fa69cba8f1` |
| `github/codeql-action` | v4.35.2 | `95e58e9a2cdfd71adc6e0353d5c52f41a045d225` |
| `docker/setup-buildx-action` | v4.0.0 | `4d04d5d9486b7bd6fa91e7baf45bbb4f8b9deedd` |
| `docker/login-action` | v4.1.0 | `4907a6ddec9925e35a0a9e82d7399ccc52663121` |
| `docker/metadata-action` | v6.0.0 | `030e881283bb7a6894de51c315a6bfe6a94e05cf` |
| `docker/build-push-action` | v7.1.0 | `bcafcacb16a39f128d818304e6c9c0c18556b85f` |
| `cycjimmy/semantic-release-action` | v6.0.0 | `b12c8f6015dc215fe37bc154d4ad456dd3833c90` |
| `softprops/action-gh-release` | v3.0.0 | `b4309332981a82ec1c5618f44dd2e27cc8bfbfda` |

Verify these are still latest before using. SHAs become stale as new versions are released.

## CI Workflow (`.github/workflows/ci.yml`)

Triggers: push to main, PRs, weekly schedule, manual dispatch.

### Jobs

1. **test-unit** — Runs unit and property-based tests with race detection and coverage. Scope the `go test` command to only packages with resolved dependencies. Do NOT use `./...` if some packages import modules not yet in `go.mod` (e.g., Upjet/Crossplane stubs before code generation).

2. **test-integration** — Runs integration tests gated behind `//go:build integration` tag. Depends on test-unit passing.

3. **trivy-fs** — Trivy filesystem scan. Split into two steps:
   - SARIF report step (no `exit-code`, always produces the file for upload to GitHub Security tab)
   - Table-format severity gate step (`exit-code: "1"`, `severity: CRITICAL,HIGH`)

   Do NOT combine `exit-code` with `format: sarif` in a single step — if vulns are found, the step fails before the SARIF upload runs.

   Do NOT use `include-dev-deps` as an action input — it's a Trivy CLI flag, not a valid trivy-action input.

4. **trivy-image** — Builds the Docker image locally and scans it. Same two-step pattern as trivy-fs. No `actions/setup-go` needed here since Docker handles the Go build.

5. **sbom** — Generates a CycloneDX SBOM via Trivy and uploads as a 90-day artifact.

6. **codeql** — CodeQL static analysis for Go.

### Permissions

```yaml
permissions:
  contents: read
  security-events: write  # Required for SARIF upload to GitHub Security tab
```

### Trivy Vulnerability Fixes

When Trivy finds a vulnerable transitive dependency, bump it directly:

```bash
go get github.com/vulnerable/package@vX.Y.Z
go mod tidy
```

Then verify the build still works and tests pass.

## Release Workflow (`.github/workflows/release.yml`)

Triggers: `workflow_run` on CI completion (main branch only). Only proceeds if CI succeeded.

### Jobs

1. **semantic-release** — Analyzes conventional commits to determine the next version. Outputs `new-release`, `version`, and `tag`. Only runs if CI passed (`github.event.workflow_run.conclusion == 'success'`).

2. **build-push** — Builds multi-arch image (amd64 + arm64) and pushes to GHCR. Only runs if semantic-release published a new version. Uses Docker Buildx with GHA cache.

3. **package** — Builds the Crossplane `.xpkg` package, pushes it to GHCR, generates an SBOM, and attaches both to the GitHub Release.

### Critical: Pull Before Package Build

The `package` job runs on a separate runner from `build-push`. The image is NOT available locally. You must explicitly pull it before `crossplane xpkg build`:

```yaml
- name: Pull runtime image
  run: docker pull ${{ env.IMAGE_NAME }}:${{ needs.semantic-release.outputs.version }}

- name: Build Crossplane package
  run: |
    cd package
    crossplane xpkg build \
      --package-root=. \
      --embed-runtime-image=${{ env.IMAGE_NAME }}:${{ needs.semantic-release.outputs.version }} \
      -o ../provider-$NAME-v${{ needs.semantic-release.outputs.version }}.xpkg
```

### Permissions

```yaml
permissions:
  contents: write       # Create tags and releases
  packages: write       # Push to GHCR
  issues: write         # semantic-release may comment on issues
  pull-requests: write  # semantic-release may comment on PRs
```

### GHCR Namespace

The `IMAGE_NAME` must use the GitHub repo owner's namespace, not an org you don't have push access to:

```yaml
# If repo is github.com/myuser/provider-foo:
IMAGE_NAME: ghcr.io/myuser/provider-foo

# NOT ghcr.io/some-org/provider-foo (unless the repo is in that org)
```

## Semantic Release Configuration (`.releaserc.json`)

```json
{
  "branches": ["main"],
  "plugins": [
    ["@semantic-release/commit-analyzer", {
      "preset": "angular",
      "releaseRules": [
        { "type": "feat", "release": "minor" },
        { "type": "fix", "release": "patch" },
        { "type": "perf", "release": "patch" },
        { "type": "refactor", "release": "patch" },
        { "breaking": true, "release": "major" }
      ]
    }],
    "@semantic-release/release-notes-generator",
    ["@semantic-release/github", {
      "successComment": false,
      "failTitle": false
    }]
  ]
}
```

Disabling `successComment` and `failTitle` prevents noisy bot comments on PRs and issues.

## Dockerfile Pattern

```dockerfile
FROM golang:1.26 AS builder
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build -ldflags "-s -w -X <module>/internal/version.Version=${VERSION}" \
    -o /usr/local/bin/<provider-name> ./cmd/provider/

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/local/bin/<provider-name> /usr/local/bin/<provider-name>
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["<provider-name>"]
```

Quote shell variables in `RUN` commands (`"${TARGETOS}"` not `${TARGETOS}`).

## Crossplane Package Metadata (`package/crossplane.yaml`)

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Provider
metadata:
  name: <provider-name>
  annotations:
    meta.crossplane.io/maintainer: <maintainer>
    meta.crossplane.io/source: <repo-url>
    meta.crossplane.io/description: <description>
spec:
  controller:
    image: ghcr.io/<owner>/<provider-name>:latest
  crossplane:
    version: ">=v1.14.0"
```

## Files to Create

| File | Purpose |
|---|---|
| `.github/workflows/ci.yml` | Tests, Trivy, CodeQL, SBOM |
| `.github/workflows/release.yml` | Semantic release, image build, xpkg publish |
| `.releaserc.json` | Semantic release config |
| `Dockerfile` | Multi-stage distroless build |
| `package/crossplane.yaml` | Crossplane provider package metadata |
| `.trivyignore` | Suppress known false positives (initially empty) |
| `.gitignore` | Exclude `bin/`, `provider`, `coverage.*`, `.scannerwork/` |

## Lessons Learned

1. Never combine `exit-code: "1"` with `format: sarif` in trivy-action — split into two steps.
2. `include-dev-deps` is a Trivy CLI flag, not a valid trivy-action input.
3. GHCR namespace must match the repo owner, not an arbitrary org.
4. The Crossplane package job must `docker pull` the image before `crossplane xpkg build --embed-runtime-image`.
5. Go version must be aligned across go.mod, Dockerfile, and CI env vars.
6. Scope `go test` to compilable packages if some packages have unresolved imports (pre-code-generation stubs).
7. Bump vulnerable transitive deps directly with `go get` rather than waiting for upstream.
