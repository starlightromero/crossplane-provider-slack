# Implementation Plan: crossplane-provider-slack

## Overview

Incremental implementation of a family-scoped native Crossplane provider for Slack, written in Go. The plan starts with project scaffolding and shared infrastructure (ProviderConfig, Slack client), then implements each managed resource with its controller, followed by property-based tests, packaging, and security validation. Each task builds on previous tasks — no orphaned code.

## Tasks

- [x] 1. Scaffold project structure and Go module
  - [x] 1.1 Initialize the Go module and directory layout
    - Scaffold from `crossplane/provider-template` using `make provider.prepare`
    - Create the repository directory structure per the design: `apis/`, `internal/clients/slack/`, `internal/controller/`, `internal/features/`, `cmd/`, `package/`
    - Initialize `go.mod` with module path and add core dependencies: `crossplane-runtime`, `controller-runtime`, `k8s.io/apimachinery`
    - Create `apis/generate.go` with `//go:generate` angryjet directives
    - _Requirements: 11.1, 11.2_

  - [x] 1.2 Set up Makefile with family-scoped build targets
    - Configure Makefile using the Crossplane "build" Make submodule
    - Add targets: `generate`, `build` (three binaries), `docker-build` (three images), `docker-push`, `xpkg-build` (three packages), `xpkg-push`
    - Configure distroless base image for all container builds
    - Configure non-root user in Dockerfiles
    - Set registry URLs to `ghcr.io/avodah-inc/` for all three packages
    - _Requirements: 10.1, 10.3, 11.3, 11.4, 11.5, 11.6, 11.7, 11.11_

  - [x] 1.3 Create feature flags module
    - Implement `internal/features/features.go` with feature flag constants
    - _Requirements: 15.1_

- [x] 2. Implement ProviderConfig types and family provider
  - [x] 2.1 Define ProviderConfig API types
    - Create `apis/v1alpha1/providerconfig_types.go` with `ProviderConfig`, `ProviderConfigSpec`, `ProviderConfigStatus`, `ProviderConfigUsage`, and list types
    - Embed `xpv1.ProviderConfigSpec` and `xpv1.ProviderConfigStatus`
    - Add `PollInterval` field with `+kubebuilder:default="5m"` and kubebuilder markers
    - Create `apis/v1alpha1/groupversion_info.go` with `SchemeBuilder` for `slack.crossplane.io`
    - Run `make generate` to produce `zz_generated.deepcopy.go` and getter/setter methods
    - _Requirements: 1.1, 1.6, 1.9, 9.1, 9.2, 9.3, 15.4_

  - [x] 2.2 Implement ProviderConfig controller
    - Create `internal/controller/providerconfig/providerconfig.go` implementing credential validation
    - Read bot token from referenced Secret; set `Ready=False` with reason `SecretNotFound` if Secret is missing
    - Validate `xoxb-` prefix; set `Ready=False` with reason `InvalidCredentials` if invalid
    - Set `Ready=True` when valid token is loaded
    - Create `internal/controller/providerconfig/setup.go` with controller registration
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 1.7_

  - [x] 2.3 Create family provider entrypoint
    - Implement `cmd/family-provider/main.go` registering only the ProviderConfig controller
    - Wire up controller-runtime manager with scheme registration
    - _Requirements: 12.3, 13.2_

  - [x] 2.4 Write property test for bot token validation
    - **Property 1: Bot token validation accepts only xoxb- prefixed strings**
    - Use `pgregory.net/rapid` to generate arbitrary strings and verify the validator accepts iff the string starts with `xoxb-`
    - **Validates: Requirements 1.4, 1.5**

  - [x] 2.5 Write property test for credential exclusion from serialization
    - **Property 6: Serialized managed resources never contain credential values**
    - Generate arbitrary ProviderConfig objects and bot token strings, serialize to JSON, assert token is absent
    - **Validates: Requirements 1.7, 10.4**

- [x] 3. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. Implement Slack API client with rate limiting
  - [x] 4.1 Create Slack client interface and HTTP implementation
    - Define `ClientAPI` interface in `internal/clients/slack/client.go` with all methods from the design
    - Implement `Client` struct with `httpClient`, `token`, `baseURL` fields
    - Implement `Do` method: build request with `Authorization: Bearer <token>`, 30s timeout
    - Implement JSON response parsing with `"ok"` field check
    - _Requirements: 2.1, 2.4, 2.6_

  - [x] 4.2 Implement rate limiting and error handling
    - Create `internal/clients/slack/ratelimit.go` with rate-limit retry logic
    - On HTTP 429 with `Retry-After`: parse header, wait, retry
    - On HTTP 429 without `Retry-After`: exponential backoff with jitter (1s base, capped at 60s), max 3 retries
    - Create `internal/clients/slack/errors.go` with `SlackError` struct, `Error()`, and `IsRetriable()` methods
    - Map retriable error codes: `internal_error`, `fatal_error`, `request_timeout`
    - Return retriable errors for network errors and timeouts so reconciler requeues
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6_

  - [x] 4.3 Write property test for Authorization header inclusion
    - **Property 2: Every Slack API request includes the Authorization header**
    - Generate arbitrary tokens and API method names, verify every request has `Authorization: Bearer <token>`
    - **Validates: Requirements 2.1**

  - [x] 4.4 Write property test for Retry-After compliance
    - **Property 3: Rate-limited responses with Retry-After are retried after the specified duration**
    - Generate arbitrary Retry-After values, verify client waits at least N seconds before retry
    - **Validates: Requirements 2.2**

  - [x] 4.5 Write property test for exponential backoff bounds
    - **Property 4: Exponential backoff with jitter produces bounded delays**
    - Generate arbitrary retry attempt numbers, verify backoff is within `[0, min(2^N * 1s, 60s))`
    - **Validates: Requirements 2.3**

  - [x] 4.6 Write property test for Slack error parsing
    - **Property 5: Slack API error responses are parsed into structured errors**
    - Generate arbitrary error code strings, construct `{"ok": false, "error": "<code>"}` responses, verify `SlackError.Code` matches
    - **Validates: Requirements 2.4**

- [x] 5. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Implement Conversation managed resource
  - [x] 6.1 Define Conversation API types
    - Create `apis/conversation/v1alpha1/types.go` with `Conversation`, `ConversationSpec`, `ConversationParameters`, `ConversationStatus`, `ConversationObservation`, and list types
    - Embed `xpv1.ResourceSpec` and `xpv1.ResourceStatus`
    - Add kubebuilder markers for validation: `name` (required, 1-80 chars), `isPrivate` (default false), `topic` (max 250), `purpose` (max 250)
    - Add kubebuilder print columns: READY, SYNCED, EXTERNAL-NAME, AGE
    - Create `apis/conversation/v1alpha1/groupversion_info.go`
    - Run `make generate`
    - _Requirements: 3.1, 3.2, 3.3, 3.12, 9.5, 14.1, 15.4_

  - [x] 6.2 Implement Conversation Slack client methods
    - Create `internal/clients/slack/conversations.go` with methods: `CreateConversation`, `GetConversationInfo`, `RenameConversation`, `SetConversationTopic`, `SetConversationPurpose`, `ArchiveConversation`
    - _Requirements: 3.4, 3.5, 3.6, 3.7, 3.8, 3.9_

  - [x] 6.3 Implement Conversation controller
    - Create `internal/controller/conversation/conversation.go` with `connector` (ExternalConnecter) and `external` (ExternalClient)
    - Implement `Connect`: track ProviderConfig usage, get Secret, validate token, return ExternalClient
    - Implement `Observe`: call `conversations.info`, compare name/topic/purpose, set `ResourceUpToDate` accordingly, populate `status.atProvider`
    - Implement `Create`: call `conversations.create`, set external-name annotation to returned channel ID
    - Implement `Update`: call `conversations.rename` / `setTopic` / `setPurpose` only for changed fields
    - Implement `Delete`: call `conversations.archive`
    - Map Slack errors: `name_taken` → `Synced=False/NameConflict`, `channel_not_found` → `Synced=False/NotFound`
    - Create `internal/controller/conversation/setup.go` with `Setup` and `SetupGated` functions using `managed.NewReconciler`
    - _Requirements: 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 3.10, 3.11, 8.1, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8, 8.9, 15.1, 15.2, 15.3, 15.6_

  - [x] 6.4 Write property test for Create stores external-name (Conversation)
    - **Property 7: Create stores the Slack-returned ID as external-name**
    - Generate arbitrary valid ConversationParameters and mock channel IDs, verify Create sets `crossplane.io/external-name` to the returned ID
    - **Validates: Requirements 3.4, 8.3**

  - [x] 6.5 Write property test for Observe state drift detection (Conversation)
    - **Property 8: Observe correctly detects state drift between desired and remote**
    - Generate arbitrary desired spec and remote state combinations for Conversation, verify `ResourceUpToDate` is true iff all fields match
    - **Validates: Requirements 3.5**

  - [x] 6.6 Write property test for Conversation Update dispatch
    - **Property 9: Conversation Update dispatches the correct API call for each changed field**
    - Generate arbitrary desired/observed name, topic, purpose combinations, verify only the correct API methods are called for changed fields
    - **Validates: Requirements 3.6, 3.7, 3.8**

  - [x] 6.7 Write unit tests for Conversation controller
    - Test Observe, Create, Update, Delete with mock Slack client
    - Test error conditions: `name_taken`, `channel_not_found`, network errors
    - _Requirements: 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 3.10, 3.11_

- [x] 7. Implement ConversationBookmark managed resource
  - [x] 7.1 Define ConversationBookmark API types
    - Create `apis/bookmark/v1alpha1/types.go` with `ConversationBookmark`, spec, parameters, status, observation, and list types
    - Add `conversationId`, `conversationRef`, `conversationSelector` fields for cross-resource references
    - Add `title` (required), `type` (required, enum: link), `link` (required, format: uri)
    - Create `apis/bookmark/v1alpha1/referencers.go` for Conversation cross-resource references
    - Create `apis/bookmark/v1alpha1/groupversion_info.go`
    - Run `make generate`
    - _Requirements: 4.1, 4.2, 4.3, 4.9, 9.5, 14.1_

  - [x] 7.2 Implement Bookmark Slack client methods
    - Create `internal/clients/slack/bookmarks.go` with methods: `AddBookmark`, `ListBookmarks`, `EditBookmark`, `RemoveBookmark`
    - _Requirements: 4.4, 4.5, 4.6, 4.7_

  - [x] 7.3 Implement ConversationBookmark controller
    - Create `internal/controller/bookmark/bookmark.go` with connector and external client
    - Implement `Observe`: call `bookmarks.list`, find by external-name, compare title/link
    - Implement `Create`: call `bookmarks.add`, set external-name to returned bookmark ID
    - Implement `Update`: call `bookmarks.edit` for changed title or link
    - Implement `Delete`: call `bookmarks.remove`
    - Map Slack errors: `channel_not_found`/`is_archived` → `Synced=False/ChannelUnavailable`
    - Create `internal/controller/bookmark/setup.go`
    - _Requirements: 4.4, 4.5, 4.6, 4.7, 4.8, 8.3, 8.8, 8.9, 15.1, 15.2, 15.3_

  - [x] 7.4 Write property test for Bookmark Update dispatch
    - **Property 10: Bookmark Update dispatches edit for changed title or link**
    - Generate arbitrary desired/observed title and link combinations, verify `bookmarks.edit` is called with updated fields
    - **Validates: Requirements 4.6**

  - [x] 7.5 Write unit tests for ConversationBookmark controller
    - Test Observe, Create, Update, Delete with mock Slack client
    - Test error conditions: `channel_not_found`, `is_archived`
    - _Requirements: 4.4, 4.5, 4.6, 4.7, 4.8_

- [x] 8. Implement ConversationPin managed resource
  - [x] 8.1 Define ConversationPin API types
    - Create `apis/pin/v1alpha1/types.go` with `ConversationPin`, spec, parameters, status, observation, and list types
    - Add `conversationId`, `conversationRef`, `conversationSelector` fields
    - Add `messageTimestamp` (required, pattern: `^\d+\.\d+$`)
    - Create `apis/pin/v1alpha1/referencers.go` for Conversation cross-resource references
    - Create `apis/pin/v1alpha1/groupversion_info.go`
    - Run `make generate`
    - _Requirements: 5.1, 5.2, 5.3, 5.9, 9.5, 14.1_

  - [x] 8.2 Implement Pin Slack client methods
    - Create `internal/clients/slack/pins.go` with methods: `AddPin`, `ListPins`, `RemovePin`
    - _Requirements: 5.4, 5.5, 5.6_

  - [x] 8.3 Implement ConversationPin controller
    - Create `internal/controller/pin/pin.go` with connector and external client
    - Implement `Observe`: call `pins.list`, check if message timestamp is in pinned items
    - Implement `Create`: call `pins.add` with channel ID and message timestamp
    - Implement `Delete`: call `pins.remove`
    - Map Slack errors: `message_not_found` → `Synced=False/MessageNotFound`, `channel_not_found`/`is_archived` → `Synced=False/ChannelUnavailable`
    - Create `internal/controller/pin/setup.go`
    - _Requirements: 5.4, 5.5, 5.6, 5.7, 5.8, 8.3, 8.8, 8.9, 15.1, 15.2, 15.3_

  - [x] 8.4 Write unit tests for ConversationPin controller
    - Test Observe, Create, Delete with mock Slack client
    - Test error conditions: `message_not_found`, `channel_not_found`, `is_archived`
    - _Requirements: 5.4, 5.5, 5.6, 5.7, 5.8_

- [x] 9. Create conversation family member entrypoint
  - Implement `cmd/provider-conversation/main.go` registering Conversation, ConversationBookmark, and ConversationPin controllers
  - Wire up controller-runtime manager with all conversation API scheme registrations
  - _Requirements: 12.4, 13.3_

- [x] 10. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. Implement UserGroup managed resource
  - [x] 11.1 Define UserGroup API types
    - Create `apis/usergroup/v1alpha1/types.go` with `UserGroup`, spec, parameters, status, observation, and list types
    - Add `name` (required), `handle` (required, pattern: `^[a-z0-9][a-z0-9._-]*$`), `description` (optional)
    - Create `apis/usergroup/v1alpha1/groupversion_info.go`
    - Run `make generate`
    - _Requirements: 6.1, 6.2, 6.3, 6.9, 9.5, 14.1_

  - [x] 11.2 Implement UserGroup Slack client methods
    - Create `internal/clients/slack/usergroups.go` with methods: `CreateUserGroup`, `ListUserGroups`, `UpdateUserGroup`, `DisableUserGroup`
    - _Requirements: 6.4, 6.5, 6.6, 6.7_

  - [x] 11.3 Implement UserGroup controller
    - Create `internal/controller/usergroup/usergroup.go` with connector and external client
    - Implement `Observe`: call `usergroups.list`, find by external-name, compare name/handle/description
    - Implement `Create`: call `usergroups.create`, set external-name to returned usergroup ID
    - Implement `Update`: call `usergroups.update` for changed name, handle, or description
    - Implement `Delete`: call `usergroups.disable`
    - Map Slack errors: `name_already_exists` → `Synced=False/NameConflict`
    - Create `internal/controller/usergroup/setup.go`
    - _Requirements: 6.4, 6.5, 6.6, 6.7, 6.8, 8.3, 8.8, 8.9, 15.1, 15.2, 15.3_

  - [x] 11.4 Write property test for UserGroup Update dispatch
    - **Property 11: UserGroup Update dispatches update for changed name, handle, or description**
    - Generate arbitrary desired/observed name, handle, description combinations, verify `usergroups.update` is called with updated fields
    - **Validates: Requirements 6.6**

  - [x] 11.5 Write unit tests for UserGroup controller
    - Test Observe, Create, Update, Delete with mock Slack client
    - Test error conditions: `name_already_exists`
    - _Requirements: 6.4, 6.5, 6.6, 6.7, 6.8_

- [x] 12. Implement UserGroupMembers managed resource
  - [x] 12.1 Define UserGroupMembers API types
    - Create `apis/usergroupmembers/v1alpha1/types.go` with `UserGroupMembers`, spec, parameters, status, observation, and list types
    - Add `userGroupId`, `userGroupRef`, `userGroupSelector` fields for cross-resource references
    - Add `userEmails` (required, minItems: 1)
    - Create `apis/usergroupmembers/v1alpha1/referencers.go` for UserGroup cross-resource references
    - Create `apis/usergroupmembers/v1alpha1/groupversion_info.go`
    - Run `make generate`
    - _Requirements: 7.1, 7.2, 7.3, 7.9, 9.5, 14.1_

  - [x] 12.2 Implement Users Slack client methods
    - Create `internal/clients/slack/users.go` with method: `LookupUserByEmail`
    - Add `ListUserGroupMembers` and `UpdateUserGroupMembers` methods to `internal/clients/slack/usergroups.go`
    - _Requirements: 7.4, 7.6, 7.7, 7.8_

  - [x] 12.3 Implement UserGroupMembers controller
    - Create `internal/controller/usergroupmembers/usergroupmembers.go` with connector and external client
    - Implement `Observe`: call `usergroups.users.list`, resolve desired emails to user IDs, compare sets (order-independent)
    - Implement `Create`: resolve emails via `users.lookupByEmail`, call `usergroups.users.update` with resolved IDs
    - Implement `Update`: same as Create (full membership replacement)
    - Implement `Delete`: call `usergroups.users.update` with empty user list
    - Handle unresolvable emails: set `Synced=False` with reason `UserNotFound`, include email in status message
    - Populate `status.atProvider.resolvedUserIds` and `memberCount`
    - Create `internal/controller/usergroupmembers/setup.go`
    - _Requirements: 7.4, 7.5, 7.6, 7.7, 7.8, 7.9, 8.3, 8.8, 8.9, 15.1, 15.2, 15.3_

  - [x] 12.4 Write property test for email resolution correctness
    - **Property 12: Email resolution produces correct user ID list for membership update**
    - Generate arbitrary email-to-userID mappings, verify controller resolves all emails and calls `usergroups.users.update` with exactly the resolved set
    - **Validates: Requirements 7.4, 7.6**

  - [x] 12.5 Write property test for unresolvable email reporting
    - **Property 13: Unresolvable emails are reported with UserNotFound**
    - Generate email lists with at least one unresolvable email, verify `Synced=False` with reason `UserNotFound` and the email appears in the status message
    - **Validates: Requirements 7.5**

  - [x] 12.6 Write property test for order-independent member set comparison
    - **Property 14: Member set comparison is order-independent**
    - Generate two permutations of the same user ID set, verify Observe reports them as equal
    - **Validates: Requirements 7.7**

  - [x] 12.7 Write unit tests for UserGroupMembers controller
    - Test Observe, Create, Update, Delete with mock Slack client
    - Test error conditions: unresolvable emails, empty member list
    - _Requirements: 7.4, 7.5, 7.6, 7.7, 7.8_

- [x] 13. Create usergroup family member entrypoint
  - Implement `cmd/provider-usergroup/main.go` registering UserGroup and UserGroupMembers controllers
  - Wire up controller-runtime manager with all usergroup API scheme registrations
  - _Requirements: 12.5, 13.4_

- [x] 14. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 15. Implement poll interval configuration and remaining property tests
  - [x] 15.1 Wire configurable poll interval into reconcilers
    - Read `spec.pollInterval` from the referenced ProviderConfig in each controller's `Connect` method
    - Pass the parsed duration to `managed.NewReconciler` via `WithPollInterval`
    - Default to 5 minutes when not set
    - _Requirements: 8.1, 8.2_

  - [x] 15.2 Write property test for configurable poll interval
    - **Property 15: Configurable poll interval is respected**
    - Generate arbitrary valid Go duration strings, verify the reconciler uses the parsed duration
    - **Validates: Requirements 8.2**

- [x] 16. Package metadata and CRD generation
  - [x] 16.1 Create Crossplane package metadata files
    - Create `package/family/crossplane.yaml` for `provider-family-slack` with `safe-start` capability
    - Create `package/conversation/crossplane.yaml` for `provider-slack-conversation` with dependency on `provider-family-slack >= v0.1.0` and `safe-start` capability
    - Create `package/usergroup/crossplane.yaml` for `provider-slack-usergroup` with dependency on `provider-family-slack >= v0.1.0` and `safe-start` capability
    - _Requirements: 1.8, 9.6, 11.8, 11.9, 11.10, 13.5, 13.6_

  - [x] 16.2 Generate and validate CRDs
    - Run `make generate` to produce all CRD YAML files in `package/crds/`
    - Place ProviderConfig CRD in `package/family/crds/`
    - Place Conversation, ConversationBookmark, ConversationPin CRDs in `package/conversation/crds/`
    - Place UserGroup, UserGroupMembers CRDs in `package/usergroup/crds/`
    - Validate all CRDs with kubeconform in `--strict` mode — zero errors
    - Verify OpenAPI v3 validation schemas are present for all `spec.forProvider` and `status.atProvider` fields
    - _Requirements: 9.1, 9.4, 9.5, 11.8, 11.9, 11.10_

- [x] 17. Container image build and security scanning
  - [x] 17.1 Build container images and run security scans
    - Run `make docker-build` to build all three distroless container images
    - Verify images run as non-root user
    - Run `trivy fs . --include-dev-deps` — zero critical/high/medium vulnerabilities
    - _Requirements: 10.1, 10.2, 10.3_

  - [x] 17.2 Write credential leak test
    - Verify bot token never appears in logs, Kubernetes events, or serialized objects across all controllers
    - _Requirements: 10.4_

- [x] 18. Build Crossplane packages
  - Run `make xpkg-build` to produce `.xpkg` files for all three family members
  - Verify `provider-family-slack.xpkg` contains only ProviderConfig CRD
  - Verify `provider-slack-conversation.xpkg` contains Conversation, ConversationBookmark, ConversationPin CRDs
  - Verify `provider-slack-usergroup.xpkg` contains UserGroup, UserGroupMembers CRDs
  - _Requirements: 11.7, 11.8, 11.9, 11.10_

- [x] 19. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after major milestones
- Property tests use `pgregory.net/rapid` with minimum 100 iterations per property
- All 15 correctness properties from the design are covered as property test sub-tasks
- The implementation language is Go throughout, matching the design document
- All controllers follow the `crossplane-runtime` pattern: `managed.NewReconciler` + `ExternalConnecter` + `ExternalClient`
