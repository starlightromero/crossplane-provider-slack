# crossplane-provider-slack

A native [Crossplane](https://crossplane.io) provider for managing Slack workspace resources via the Slack Web API. Built as a **family-scoped provider** following Crossplane v2 best practices — install only the resource families you need.

## Overview

This provider enables Kubernetes-native, declarative management of Slack resources. It uses the Slack Web API directly (no Terraform/Upjet dependency) and produces three independently installable Crossplane packages:

| Package | Resources | Description |
|---------|-----------|-------------|
| `provider-family-slack` | `ProviderConfig` | Shared credential management and provider settings |
| `provider-slack-conversation` | `Conversation`, `ConversationBookmark`, `ConversationPin` | Channel lifecycle, bookmarks, and pinned messages |
| `provider-slack-usergroup` | `UserGroup`, `UserGroupMembers` | User group management and membership |

Each family member declares a dependency on `provider-family-slack`, which is automatically installed when any member is deployed.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                         │
│                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────┐ │
│  │ Family Provider   │  │ Conversation Pod │  │ UserGroup │ │
│  │ Pod               │  │                  │  │ Pod       │ │
│  │                   │  │ • Conversation   │  │           │ │
│  │ • ProviderConfig  │  │ • Bookmark       │  │ • Group   │ │
│  │   Controller      │  │ • Pin            │  │ • Members │ │
│  └────────┬──────────┘  └────────┬─────────┘  └─────┬─────┘ │
│           │                      │                   │       │
└───────────┼──────────────────────┼───────────────────┼───────┘
            │                      │                   │
            ▼                      ▼                   ▼
     ┌─────────────┐       ┌─────────────┐     ┌───────────┐
     │ K8s Secret  │       │ Slack API   │     │ Slack API │
     │ (bot token) │       │ channels    │     │ usergroups│
     └─────────────┘       └─────────────┘     └───────────┘
```

## Prerequisites

- Crossplane v2.0.0+
- A Slack bot token (`xoxb-...`) with appropriate scopes:
  - `channels:manage`, `channels:read` — for Conversation resources
  - `bookmarks:write`, `bookmarks:read` — for ConversationBookmark resources
  - `pins:write`, `pins:read` — for ConversationPin resources
  - `usergroups:write`, `usergroups:read` — for UserGroup resources
  - `users:read`, `users:read.email` — for UserGroupMembers email resolution

## Installation

Install the conversation family member (automatically installs the family provider):

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-slack-conversation
spec:
  package: ghcr.io/avodah-inc/provider-slack-conversation:v0.1.0
```

Install the usergroup family member:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-slack-usergroup
spec:
  package: ghcr.io/avodah-inc/provider-slack-usergroup:v0.1.0
```

## Configuration

### 1. Create a Secret with your Slack bot token

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-creds
  namespace: crossplane-system
type: Opaque
stringData:
  token: xoxb-your-bot-token-here
```

### 2. Create a ProviderConfig

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: slack-creds
      namespace: crossplane-system
      key: token
  pollInterval: 5m
```

## Managed Resources

### Conversation

Manage Slack channels declaratively:

```yaml
apiVersion: conversation.slack.crossplane.io/v1alpha1
kind: Conversation
metadata:
  name: platform-alerts
spec:
  forProvider:
    name: platform-alerts
    isPrivate: false
    topic: "Production alerts and notifications"
    purpose: "Automated alerting channel managed by Crossplane"
  providerConfigRef:
    name: default
```

### ConversationBookmark

Pin links to channels:

```yaml
apiVersion: bookmark.slack.crossplane.io/v1alpha1
kind: ConversationBookmark
metadata:
  name: runbook-link
spec:
  forProvider:
    conversationRef:
      name: platform-alerts
    title: "Incident Runbook"
    type: link
    link: "https://wiki.example.com/runbooks/incidents"
  providerConfigRef:
    name: default
```

### ConversationPin

Pin messages in channels:

```yaml
apiVersion: pin.slack.crossplane.io/v1alpha1
kind: ConversationPin
metadata:
  name: welcome-pin
spec:
  forProvider:
    conversationRef:
      name: platform-alerts
    messageTimestamp: "1234567890.123456"
  providerConfigRef:
    name: default
```

### UserGroup

Manage Slack user groups (mention handles):

```yaml
apiVersion: usergroup.slack.crossplane.io/v1alpha1
kind: UserGroup
metadata:
  name: oncall-sre
spec:
  forProvider:
    name: "SRE On-Call"
    handle: oncall-sre
    description: "Current SRE on-call rotation"
  providerConfigRef:
    name: default
```

### UserGroupMembers

Manage group membership by email:

```yaml
apiVersion: usergroupmembers.slack.crossplane.io/v1alpha1
kind: UserGroupMembers
metadata:
  name: oncall-sre-members
spec:
  forProvider:
    userGroupRef:
      name: oncall-sre
    userEmails:
      - alice@example.com
      - bob@example.com
  providerConfigRef:
    name: default
```

## Safe Start & Activation

This provider declares the `safe-start` capability. On Crossplane v2, all ManagedResourceDefinitions start as **Inactive**. Activate the resources you need:

```yaml
apiVersion: pkg.crossplane.io/v1alpha1
kind: ManagedResourceActivationPolicy
metadata:
  name: activate-slack-conversations
spec:
  providerRef:
    name: provider-slack-conversation
  activate:
    - kind: Conversation
    - kind: ConversationBookmark
    - kind: ConversationPin
```

## Development

### Requirements

- Go 1.26+
- Docker (for image builds)
- [Crossplane CLI](https://docs.crossplane.io/latest/cli/) (for xpkg builds)

### Build

```bash
# Build all binaries
make build

# Run tests
make test

# Run code generation (after modifying API types)
make generate

# Build Docker images
make docker-build

# Build Crossplane packages
make xpkg-build
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all three provider binaries |
| `make test` | Run unit and property-based tests |
| `make generate` | Run code generation (angryjet, controller-gen) |
| `make docker-build` | Build all container images |
| `make docker-push` | Push all images to GHCR |
| `make xpkg-build` | Build Crossplane `.xpkg` packages |
| `make xpkg-push` | Push packages to GHCR |
| `make clean` | Remove build artifacts |
| `make help` | Show all targets |

### Project Structure

```
├── apis/                          # API type definitions
│   ├── v1alpha1/                  # ProviderConfig types
│   ├── conversation/v1alpha1/     # Conversation types
│   ├── bookmark/v1alpha1/         # ConversationBookmark types
│   ├── pin/v1alpha1/              # ConversationPin types
│   ├── usergroup/v1alpha1/        # UserGroup types
│   └── usergroupmembers/v1alpha1/ # UserGroupMembers types
├── cmd/
│   ├── family-provider/           # Family provider entrypoint
│   ├── provider-conversation/     # Conversation member entrypoint
│   └── provider-usergroup/        # UserGroup member entrypoint
├── internal/
│   ├── clients/slack/             # Slack Web API client
│   ├── controller/                # Reconciliation controllers
│   └── features/                  # Feature flags
├── package/
│   ├── family/                    # Family provider package metadata + CRDs
│   ├── conversation/              # Conversation package metadata + CRDs
│   └── usergroup/                 # UserGroup package metadata + CRDs
└── .github/workflows/             # CI/CD pipelines
```

### Testing

The provider uses property-based testing with [`pgregory.net/rapid`](https://github.com/flyingmutant/rapid) to verify correctness properties:

- Bot token validation (accepts only `xoxb-` prefixed strings)
- Authorization header inclusion on every request
- Rate-limit retry compliance
- Exponential backoff bounds
- Slack error parsing
- Credential exclusion from serialization
- External-name assignment on create
- State drift detection
- Update dispatch correctness
- Email resolution correctness
- Order-independent member set comparison
- Poll interval parsing

Run all tests:

```bash
go test -race ./...
```

## CI/CD

- **CI** (`.github/workflows/ci.yml`): Unit tests, integration tests, Trivy filesystem scan, Trivy image scan, SBOM generation, CodeQL analysis
- **Release** (`.github/workflows/release.yml`): Semantic release from conventional commits, multi-arch Docker builds, Crossplane package publishing

Releases are automated via [semantic-release](https://semantic-release.gitbook.io/) using Angular commit conventions:

| Commit prefix | Release type |
|---------------|-------------|
| `feat:` | Minor |
| `fix:` | Patch |
| `perf:` | Patch |
| `refactor:` | Patch |
| `feat!:` / `BREAKING CHANGE:` | Major |

## Security

- All container images use `gcr.io/distroless/static:nonroot` base
- Containers run as non-root (UID 65532)
- Bot tokens are never stored in CRD specs, status fields, or logs
- Trivy scans enforce zero CRITICAL/HIGH vulnerabilities
- CodeQL static analysis runs on every push
- All GitHub Actions are SHA-pinned to prevent supply chain attacks

## License

[GPL-3.0](LICENSE)
