# Crossplane Provider for Slack — Spec

## Overview

This provider enables Kubernetes-native management of Slack workspace
resources via Crossplane. It is built as a **native Crossplane
provider** using [crossplane-tools](https://github.com/crossplane/crossplane-tools)
and the [Slack Web API](https://api.slack.com/methods) directly, since
no up-to-date Terraform provider exists for Upjet generation.

## Provider Configuration

### ProviderConfig

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: slack-credentials
      key: bot-token
```

The provider authenticates via a Slack Bot Token (`xoxb-*`) stored in
a Kubernetes Secret. The bot must be installed in the target workspace
with the required OAuth scopes for the resources being managed.

### Credential Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  bot-token: <SLACK_BOT_TOKEN>
```

### Required OAuth Scopes

| Scope | Used By |
|---|---|
| `channels:manage` | Conversation (public) |
| `channels:read` | Conversation (public) |
| `channels:join` | Conversation (public) |
| `groups:write` | Conversation (private) |
| `groups:read` | Conversation (private) |
| `bookmarks:write` | Bookmark |
| `bookmarks:read` | Bookmark |
| `usergroups:write` | UserGroup |
| `usergroups:read` | UserGroup |
| `users:read` | UserGroupMembers (resolve users) |
| `users:read.email` | UserGroupMembers (resolve by email) |
| `pins:write` | Pin |
| `pins:read` | Pin |

## API Group

All managed resources use the API group `slack.crossplane.io`.
Initial version: `v1alpha1`.

## Managed Resources

Derived from the Slack Web API's CRUD-capable resource families. Only
resources that support create, read, update, and/or delete via the API
are included.

### Conversations

| Crossplane Kind | Slack API Methods | Description |
|---|---|---|
| `Conversation` | `conversations.create`, `conversations.info`, `conversations.rename`, `conversations.setPurpose`, `conversations.setTopic`, `conversations.archive`, `conversations.unarchive` | Public or private channels |
| `ConversationBookmark` | `bookmarks.add`, `bookmarks.list`, `bookmarks.edit`, `bookmarks.remove` | Channel bookmarks (links, docs) |
| `ConversationPin` | `pins.add`, `pins.list`, `pins.remove` | Pinned messages in a channel |

### User Groups

| Crossplane Kind | Slack API Methods | Description |
|---|---|---|
| `UserGroup` | `usergroups.create`, `usergroups.list`, `usergroups.update`, `usergroups.disable`, `usergroups.enable` | User groups (handle-based mention groups) |
| `UserGroupMembers` | `usergroups.users.list`, `usergroups.users.update` | Membership of a user group |

## Example Managed Resources

### Conversation (Channel)

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: Conversation
metadata:
  name: platform-engineering
spec:
  forProvider:
    name: platform-engineering
    isPrivate: false
    topic: Platform Engineering team channel
    purpose: Coordination for platform infrastructure work
  providerConfigRef:
    name: default
```

### UserGroup

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: UserGroup
metadata:
  name: oncall-platform
spec:
  forProvider:
    name: On-Call Platform
    handle: oncall-platform
    description: Current on-call engineers for platform
  providerConfigRef:
    name: default
```

### UserGroupMembers

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: UserGroupMembers
metadata:
  name: oncall-platform-members
spec:
  forProvider:
    userGroupRef:
      name: oncall-platform
    userEmails:
      - alice@avodah.com
      - bob@avodah.com
  providerConfigRef:
    name: default
```

### ConversationBookmark

```yaml
apiVersion: slack.crossplane.io/v1alpha1
kind: ConversationBookmark
metadata:
  name: platform-runbook
spec:
  forProvider:
    conversationRef:
      name: platform-engineering
    title: Platform Runbook
    type: link
    link: https://docs.avodah.com/runbooks/platform
  providerConfigRef:
    name: default
```

## Project Structure

Native Crossplane provider layout:

```
crossplane-provider-slack/
├── apis/
│   ├── conversation/
│   │   └── v1alpha1/          # Conversation, ConversationBookmark, ConversationPin
│   ├── usergroup/
│   │   └── v1alpha1/          # UserGroup, UserGroupMembers
│   └── v1alpha1/
│       └── providerconfig.go  # ProviderConfig type
├── internal/
│   ├── clients/
│   │   └── slack/             # Slack Web API client (thin wrapper)
│   ├── controller/
│   │   ├── conversation/      # Conversation controller
│   │   ├── bookmark/          # ConversationBookmark controller
│   │   ├── pin/               # ConversationPin controller
│   │   ├── usergroup/         # UserGroup controller
│   │   └── usergroupmembers/  # UserGroupMembers controller
│   └── features/              # Feature flags
├── package/
│   ├── crds/                  # Generated CRD YAML
│   └── crossplane.yaml        # Provider package metadata
├── cmd/
│   └── provider/
│       └── main.go            # Provider binary entrypoint
├── Makefile
├── go.mod
└── README.md
```

## Rate Limiting

The Slack Web API enforces [rate limits](https://api.slack.com/docs/rate-limits)
per method tier:

| Tier | Requests/min | Methods |
|---|---|---|
| Tier 1 | ~1 | `admin.*` |
| Tier 2 | ~20 | `conversations.create`, `usergroups.create` |
| Tier 3 | ~50 | `conversations.info`, `bookmarks.list` |
| Tier 4 | ~100 | `users.list` |

The provider must:

- Respect `Retry-After` headers on 429 responses
- Use exponential backoff with jitter in controller reconciliation
- Set a conservative default poll interval (5 minutes) to avoid
  hitting limits during steady-state reconciliation
- Support configurable poll intervals via ProviderConfig

## Build and Publish

```bash
make generate        # Generate CRDs and deepcopy methods
make build           # Build provider binary
make docker-build    # Build container image
make docker-push     # Push to registry
make xpkg-build      # Build Crossplane package
make xpkg-push       # Push package to registry
```

## Integration with aws-eks-modules

The provider will be installed via the `crossplane` Flux module in
`aws-eks-modules`:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-slack
spec:
  package: ghcr.io/avodah-inc/crossplane-provider-slack:v0.1.0
  controllerConfigRef:
    name: provider-slack
```

## Security

- Bot token stored in Kubernetes Secret, referenced via ProviderConfig
- No credentials in CRDs or provider package
- Trivy scan must pass with no critical/high/medium issues
- Provider container image is distroless
- Bot token should follow least-privilege: only grant scopes needed
  for the resources being managed

## Validation

All generated CRDs validated with kubeconform:

```bash
kubeconform \
  -schema-location default \
  -schema-location "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
  -strict \
  -summary package/crds/
```

## References

- [Slack Web API Methods](https://api.slack.com/methods)
- [Slack Conversations API](https://api.slack.com/docs/conversations-api)
- [Slack Rate Limits](https://api.slack.com/docs/rate-limits)
- [Crossplane Provider Development](https://docs.crossplane.io/latest/guides/write-a-composition-function-in-go/)
- [crossplane-tools](https://github.com/crossplane/crossplane-tools)
