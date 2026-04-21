# Requirements Document

## Introduction

This document defines the requirements for a native Crossplane provider that enables Kubernetes-native management of Slack workspace resources. The provider uses the Slack Web API directly (no Terraform/Upjet dependency) and is structured as a **family-scoped provider** following Crossplane 2.0 best practices. The family consists of three packages: `provider-family-slack` (owns the shared ProviderConfig), `provider-slack-conversation` (Conversation, ConversationBookmark, ConversationPin), and `provider-slack-usergroup` (UserGroup, UserGroupMembers). All managed resources support both namespaced (default in Crossplane v2) and cluster-scoped modes, using the `.m` API group suffix convention for namespaced variants (e.g., `conversation.slack.m.crossplane.io`). The provider is scaffolded from the official `crossplane/provider-template` repository and uses `crossplane-runtime` reconciler patterns.

## Glossary

- **Family_Provider**: The top-level Crossplane package (`provider-family-slack`) that owns the shared ProviderConfig CRD and is automatically installed as a dependency when any family member package is installed.
- **Family_Member**: A scoped Crossplane package (e.g., `provider-slack-conversation`, `provider-slack-usergroup`) that manages a subset of managed resources and declares a dependency on the Family_Provider.
- **Provider**: A Crossplane provider binary that runs as a Kubernetes controller pod, reconciling Slack managed resources against the Slack Web API.
- **ProviderConfig**: A Crossplane CRD (`slack.crossplane.io/v1alpha1`) owned by the Family_Provider that holds credential references and provider-level settings such as poll interval.
- **Managed_Resource**: A Kubernetes custom resource representing a Slack workspace object, reconciled by a Provider controller.
- **ManagedResourceDefinition**: A Crossplane v2 object (MRD) that replaces raw CRDs for managed resources during provider installation, enabling activation control and reduced API server overhead.
- **Safe_Start**: A Crossplane v2 provider capability declared in package metadata that causes all MRDs to start as Inactive by default, requiring explicit activation via a ManagedResourceActivationPolicy.
- **ManagedResourceActivationPolicy**: A Crossplane v2 resource that activates specific MRDs, allowing platform engineers to enable only the managed resource kinds they need.
- **DeploymentRuntimeConfig**: The Crossplane v2 replacement for ControllerConfig, used to configure Pod-level settings (resource limits, environment variables, tolerations) for provider controller pods.
- **Namespaced_MR**: A managed resource scoped to a Kubernetes namespace (Crossplane v2 default), using the `.m` API group suffix convention (e.g., `conversation.slack.m.crossplane.io`).
- **Conversation_Controller**: The reconciliation controller responsible for observing, creating, updating, and deleting Conversation resources.
- **Bookmark_Controller**: The reconciliation controller responsible for observing, creating, updating, and deleting ConversationBookmark resources.
- **Pin_Controller**: The reconciliation controller responsible for observing, creating, updating, and deleting ConversationPin resources.
- **UserGroup_Controller**: The reconciliation controller responsible for observing, creating, updating, and deleting UserGroup resources.
- **UserGroupMembers_Controller**: The reconciliation controller responsible for observing, creating, updating, and deleting UserGroupMembers resources.
- **Slack_Client**: A thin Go wrapper around the Slack Web API HTTP endpoints used by all controllers.
- **Bot_Token**: A Slack bot token (prefixed `xoxb-`) used to authenticate API calls.
- **Reconciliation_Loop**: The periodic observe-compare-act cycle that a Crossplane controller executes for each managed resource.
- **External_Name**: The Slack-side identifier (e.g., channel ID, bookmark ID, usergroup ID) stored as the `crossplane.io/external-name` annotation on the managed resource after creation.
- **Poll_Interval**: The duration between successive reconciliation cycles for a managed resource.
- **CRD**: Custom Resource Definition — the Kubernetes schema for a custom resource kind.
- **Distroless_Image**: A minimal container image containing only the application binary and its runtime dependencies, with no shell or package manager.
- **ExternalClient**: The `crossplane-runtime` interface (`managed.ExternalClient`) that each controller implements with Observe, Create, Update, and Delete methods.
- **ExternalConnecter**: The `crossplane-runtime` interface (`managed.ExternalConnecter`) that creates ExternalClient instances from a ProviderConfig reference.
- **Composition_Function**: The Crossplane v2 replacement for native patch-and-transform composition, used to compose managed resources into higher-level abstractions.

## Requirements

### Requirement 1: Provider Configuration and Authentication

**User Story:** As a platform engineer, I want to configure the Crossplane provider with a Slack bot token stored in a Kubernetes Secret, so that the provider can authenticate against the Slack Web API securely.

#### Acceptance Criteria

1. THE Family_Provider SHALL register a ProviderConfig CRD at API group `slack.crossplane.io/v1alpha1` with kind `ProviderConfig`.
2. WHEN a ProviderConfig resource is created with a `spec.credentials.secretRef` referencing a Kubernetes Secret, THE Family_Provider SHALL read the bot token from the specified Secret key.
3. IF the referenced Kubernetes Secret does not exist, THEN THE Family_Provider SHALL set the ProviderConfig status condition to `Ready=False` with reason `SecretNotFound`.
4. IF the bot token value is empty or does not match the `xoxb-` prefix pattern, THEN THE Family_Provider SHALL set the ProviderConfig status condition to `Ready=False` with reason `InvalidCredentials`.
5. WHEN a valid bot token is loaded, THE Family_Provider SHALL set the ProviderConfig status condition to `Ready=True`.
6. THE ProviderConfig SHALL expose a `spec.pollInterval` field that accepts a duration string (default: `5m`) to configure the reconciliation poll interval for all managed resources referencing the ProviderConfig.
7. THE Family_Provider SHALL store no credentials in CRD specs, status fields, or provider package metadata.
8. WHEN a Family_Member is installed without the Family_Provider present, THE Family_Member package metadata SHALL declare a dependency on `provider-family-slack` so that Crossplane automatically installs the Family_Provider.
9. THE ProviderConfig CRD SHALL be owned exclusively by the Family_Provider package, and Family_Member packages SHALL reference the shared ProviderConfig without re-registering the CRD.

### Requirement 2: Slack API Client

**User Story:** As a provider developer, I want a reusable Slack Web API client wrapper, so that all controllers use consistent HTTP handling, authentication, and rate-limit compliance.

#### Acceptance Criteria

1. THE Slack_Client SHALL authenticate every Slack API request using the bot token from the referenced ProviderConfig.
2. WHEN the Slack API returns HTTP status 429, THE Slack_Client SHALL read the `Retry-After` response header and wait for the specified duration before retrying the request.
3. IF the Slack API returns HTTP status 429 and no `Retry-After` header is present, THEN THE Slack_Client SHALL apply exponential backoff with jitter starting at 1 second.
4. WHEN the Slack API returns a response with `"ok": false`, THE Slack_Client SHALL return a structured error containing the Slack error code and a human-readable message.
5. IF the Slack API is unreachable (network error or timeout), THEN THE Slack_Client SHALL return a retriable error so the Reconciliation_Loop can requeue the resource.
6. THE Slack_Client SHALL set a request timeout of 30 seconds for each individual Slack API call.

### Requirement 3: Conversation Managed Resource

**User Story:** As a platform engineer, I want to manage Slack channels (public and private) as Kubernetes resources, so that channel lifecycle is declarative and version-controlled.

#### Acceptance Criteria

1. THE Provider SHALL register a Conversation Managed_Resource at API group `slack.crossplane.io/v1alpha1` with kind `Conversation`, and a namespaced variant at `conversation.slack.m.crossplane.io/v1alpha1`.
2. THE Conversation Managed_Resource SHALL expose `spec.forProvider` fields: `name` (required, string), `isPrivate` (optional, boolean, default `false`), `topic` (optional, string), and `purpose` (optional, string).
3. THE Conversation Managed_Resource type SHALL embed `xpv1.ResourceSpec` in its spec and `xpv1.ResourceStatus` in its status to satisfy the `resource.Managed` interface.
4. WHEN a Conversation resource is created, THE Conversation_Controller SHALL call `conversations.create` with the specified name and `is_private` flag, and store the returned channel ID as the External_Name annotation.
5. WHEN the Conversation_Controller observes an existing Conversation resource, THE Conversation_Controller SHALL call `conversations.info` using the External_Name and compare the remote state with the desired spec.
6. WHEN the desired `name` differs from the observed remote name, THE Conversation_Controller SHALL call `conversations.rename` with the new name.
7. WHEN the desired `topic` differs from the observed remote topic, THE Conversation_Controller SHALL call `conversations.setTopic` with the new topic.
8. WHEN the desired `purpose` differs from the observed remote purpose, THE Conversation_Controller SHALL call `conversations.setPurpose` with the new purpose.
9. WHEN a Conversation resource is deleted, THE Conversation_Controller SHALL call `conversations.archive` to archive the channel.
10. IF the Slack API returns error `name_taken` during creation, THEN THE Conversation_Controller SHALL set the resource condition to `Synced=False` with reason `NameConflict`.
11. IF the Slack API returns error `channel_not_found` during observation, THEN THE Conversation_Controller SHALL set the resource condition to `Synced=False` with reason `NotFound`.
12. THE Conversation Managed_Resource SHALL expose `status.atProvider` fields: `id` (string), `isArchived` (boolean), `numMembers` (integer), and `created` (integer, Unix timestamp).

### Requirement 4: ConversationBookmark Managed Resource

**User Story:** As a platform engineer, I want to manage channel bookmarks as Kubernetes resources, so that important links and documents are pinned to channels declaratively.

#### Acceptance Criteria

1. THE Provider SHALL register a ConversationBookmark Managed_Resource at API group `slack.crossplane.io/v1alpha1` with kind `ConversationBookmark`, and a namespaced variant at `conversationbookmark.slack.m.crossplane.io/v1alpha1`.
2. THE ConversationBookmark Managed_Resource SHALL expose `spec.forProvider` fields: `conversationRef` or `conversationId` (required, reference to a Conversation or raw channel ID), `title` (required, string), `type` (required, string, one of `link`), and `link` (required, string, valid URL).
3. THE ConversationBookmark Managed_Resource type SHALL embed `xpv1.ResourceSpec` in its spec and `xpv1.ResourceStatus` in its status to satisfy the `resource.Managed` interface.
4. WHEN a ConversationBookmark resource is created, THE Bookmark_Controller SHALL call `bookmarks.add` with the channel ID, title, type, and link, and store the returned bookmark ID as the External_Name annotation.
5. WHEN the Bookmark_Controller observes an existing ConversationBookmark resource, THE Bookmark_Controller SHALL call `bookmarks.list` for the channel and locate the bookmark by External_Name to compare remote state with the desired spec.
6. WHEN the desired `title` or `link` differs from the observed remote values, THE Bookmark_Controller SHALL call `bookmarks.edit` with the updated fields.
7. WHEN a ConversationBookmark resource is deleted, THE Bookmark_Controller SHALL call `bookmarks.remove` to remove the bookmark.
8. IF the referenced channel does not exist or is archived, THEN THE Bookmark_Controller SHALL set the resource condition to `Synced=False` with reason `ChannelUnavailable`.
9. THE ConversationBookmark Managed_Resource SHALL expose `status.atProvider` fields: `id` (string), `channelId` (string), and `dateCreated` (integer, Unix timestamp).

### Requirement 5: ConversationPin Managed Resource

**User Story:** As a platform engineer, I want to manage pinned messages in Slack channels as Kubernetes resources, so that important messages remain pinned declaratively.

#### Acceptance Criteria

1. THE Provider SHALL register a ConversationPin Managed_Resource at API group `slack.crossplane.io/v1alpha1` with kind `ConversationPin`, and a namespaced variant at `conversationpin.slack.m.crossplane.io/v1alpha1`.
2. THE ConversationPin Managed_Resource SHALL expose `spec.forProvider` fields: `conversationRef` or `conversationId` (required, reference to a Conversation or raw channel ID) and `messageTimestamp` (required, string, the Slack message `ts` value).
3. THE ConversationPin Managed_Resource type SHALL embed `xpv1.ResourceSpec` in its spec and `xpv1.ResourceStatus` in its status to satisfy the `resource.Managed` interface.
4. WHEN a ConversationPin resource is created, THE Pin_Controller SHALL call `pins.add` with the channel ID and message timestamp.
5. WHEN the Pin_Controller observes an existing ConversationPin resource, THE Pin_Controller SHALL call `pins.list` for the channel and verify the message timestamp is present in the pinned items list.
6. WHEN a ConversationPin resource is deleted, THE Pin_Controller SHALL call `pins.remove` to unpin the message.
7. IF the referenced message does not exist, THEN THE Pin_Controller SHALL set the resource condition to `Synced=False` with reason `MessageNotFound`.
8. IF the referenced channel does not exist or is archived, THEN THE Pin_Controller SHALL set the resource condition to `Synced=False` with reason `ChannelUnavailable`.
9. THE ConversationPin Managed_Resource SHALL expose `status.atProvider` fields: `channelId` (string) and `pinnedAt` (integer, Unix timestamp).

### Requirement 6: UserGroup Managed Resource

**User Story:** As a platform engineer, I want to manage Slack user groups as Kubernetes resources, so that mention handles and group metadata are declarative and version-controlled.

#### Acceptance Criteria

1. THE Provider SHALL register a UserGroup Managed_Resource at API group `slack.crossplane.io/v1alpha1` with kind `UserGroup`, and a namespaced variant at `usergroup.slack.m.crossplane.io/v1alpha1`.
2. THE UserGroup Managed_Resource SHALL expose `spec.forProvider` fields: `name` (required, string), `handle` (required, string), and `description` (optional, string).
3. THE UserGroup Managed_Resource type SHALL embed `xpv1.ResourceSpec` in its spec and `xpv1.ResourceStatus` in its status to satisfy the `resource.Managed` interface.
4. WHEN a UserGroup resource is created, THE UserGroup_Controller SHALL call `usergroups.create` with the specified name, handle, and description, and store the returned usergroup ID as the External_Name annotation.
5. WHEN the UserGroup_Controller observes an existing UserGroup resource, THE UserGroup_Controller SHALL call `usergroups.list` and locate the usergroup by External_Name to compare remote state with the desired spec.
6. WHEN the desired `name`, `handle`, or `description` differs from the observed remote values, THE UserGroup_Controller SHALL call `usergroups.update` with the updated fields.
7. WHEN a UserGroup resource is deleted, THE UserGroup_Controller SHALL call `usergroups.disable` to disable the user group.
8. IF the Slack API returns error `name_already_exists` during creation, THEN THE UserGroup_Controller SHALL set the resource condition to `Synced=False` with reason `NameConflict`.
9. THE UserGroup Managed_Resource SHALL expose `status.atProvider` fields: `id` (string), `isEnabled` (boolean), `createdBy` (string, user ID), and `dateCreate` (integer, Unix timestamp).

### Requirement 7: UserGroupMembers Managed Resource

**User Story:** As a platform engineer, I want to manage user group membership as a Kubernetes resource, so that group members are declaratively controlled and can reference users by email.

#### Acceptance Criteria

1. THE Provider SHALL register a UserGroupMembers Managed_Resource at API group `slack.crossplane.io/v1alpha1` with kind `UserGroupMembers`, and a namespaced variant at `usergroupmembers.slack.m.crossplane.io/v1alpha1`.
2. THE UserGroupMembers Managed_Resource SHALL expose `spec.forProvider` fields: `userGroupRef` or `userGroupId` (required, reference to a UserGroup or raw usergroup ID) and `userEmails` (required, list of email strings).
3. THE UserGroupMembers Managed_Resource type SHALL embed `xpv1.ResourceSpec` in its spec and `xpv1.ResourceStatus` in its status to satisfy the `resource.Managed` interface.
4. WHEN a UserGroupMembers resource is created or updated, THE UserGroupMembers_Controller SHALL resolve each email in `userEmails` to a Slack user ID using the `users.lookupByEmail` API method.
5. IF an email in `userEmails` cannot be resolved to a Slack user ID, THEN THE UserGroupMembers_Controller SHALL set the resource condition to `Synced=False` with reason `UserNotFound` and include the unresolvable email in the status message.
6. WHEN all emails are resolved, THE UserGroupMembers_Controller SHALL call `usergroups.users.update` with the full list of resolved user IDs to set the group membership.
7. WHEN the UserGroupMembers_Controller observes an existing UserGroupMembers resource, THE UserGroupMembers_Controller SHALL call `usergroups.users.list` and compare the current member set with the desired resolved user IDs.
8. WHEN a UserGroupMembers resource is deleted, THE UserGroupMembers_Controller SHALL call `usergroups.users.update` with an empty user list to remove all members from the group.
9. THE UserGroupMembers Managed_Resource SHALL expose `status.atProvider` fields: `resolvedUserIds` (list of strings) and `memberCount` (integer).

### Requirement 8: Reconciliation Behavior

**User Story:** As a platform engineer, I want the provider to follow standard Crossplane reconciliation patterns using crossplane-runtime, so that managed resources converge to the desired state reliably.

#### Acceptance Criteria

1. THE Provider SHALL use the default Poll_Interval of 5 minutes for all managed resources when no `spec.pollInterval` is set on the referenced ProviderConfig.
2. WHEN a `spec.pollInterval` is set on the referenced ProviderConfig, THE Provider SHALL use the configured interval for reconciliation cycles.
3. WHEN a managed resource is created, THE Provider SHALL set the `crossplane.io/external-name` annotation to the Slack-side identifier returned by the create API call.
4. WHEN the external resource matches the desired spec, THE Provider SHALL set the resource condition to `Synced=True` and `Ready=True`.
5. WHEN the external resource differs from the desired spec, THE Provider SHALL update the external resource and set the resource condition to `Synced=True` after a successful update.
6. IF a reconciliation cycle fails due to a retriable error, THEN THE Provider SHALL requeue the resource with exponential backoff and jitter.
7. IF a reconciliation cycle fails due to a non-retriable error, THEN THE Provider SHALL set the resource condition to `Synced=False` with a descriptive reason and message.
8. THE Provider SHALL use `managed.NewReconciler` from `crossplane-runtime` with a `managed.ExternalConnecter` implementation for each controller.
9. THE Provider SHALL implement the ExternalClient interface (Observe, Create, Update, Delete) for each managed resource kind.

### Requirement 9: CRD Generation, ManagedResourceDefinitions, and Validation

**User Story:** As a provider developer, I want CRDs to be generated from Go type definitions and converted to ManagedResourceDefinitions with safe-start support, so that the provider follows Crossplane v2 activation patterns and reduces API server overhead.

#### Acceptance Criteria

1. WHEN `make generate` is executed, THE Provider build system SHALL generate CRD YAML files from Go type definitions in the `apis/` directory and place them in `package/crds/`.
2. WHEN `make generate` is executed, THE Provider build system SHALL generate `DeepCopyObject` methods for all API types using angryjet.
3. WHEN `make generate` is executed, THE Provider build system SHALL generate getter and setter methods for all managed resource types using angryjet.
4. WHEN CRD YAML files are validated with kubeconform using `--strict` mode, THE generated CRDs SHALL pass with zero errors.
5. THE generated CRDs SHALL include OpenAPI v3 validation schemas for all `spec.forProvider` and `status.atProvider` fields.
6. THE Provider package metadata (`crossplane.yaml`) SHALL declare the `safe-start` capability so that Crossplane v2 installs all MRDs as Inactive by default.
7. WHEN the Provider is installed on a Crossplane v2 cluster, THE Crossplane runtime SHALL automatically convert the provider CRDs to ManagedResourceDefinitions.
8. WHEN a ManagedResourceActivationPolicy targeting a specific MRD is applied, THE Crossplane runtime SHALL activate the corresponding MRD and make the managed resource kind available for use.

### Requirement 10: Container Image and Security

**User Story:** As a security engineer, I want the provider container image to be minimal and free of known vulnerabilities, so that the provider meets organizational security standards.

#### Acceptance Criteria

1. THE Provider container image SHALL use a distroless base image containing only the provider binary and its runtime dependencies.
2. WHEN a Trivy filesystem scan is executed against the provider source, THE scan SHALL report zero critical, high, or medium severity vulnerabilities.
3. THE Provider container image SHALL run as a non-root user.
4. THE Provider SHALL not embed or log the bot token value in any output, log message, or Kubernetes event.

### Requirement 11: Build and Packaging for Family-Scoped Provider

**User Story:** As a provider developer, I want standard Makefile targets for building, packaging, and publishing each family member as an independent Crossplane package, so that CI/CD pipelines can automate the release process for the family-scoped provider.

#### Acceptance Criteria

1. THE Provider repository SHALL be scaffolded from the official `crossplane/provider-template` repository using `make provider.prepare`.
2. WHEN `make provider.addtype` is executed with a resource group and kind, THE build system SHALL scaffold the Go type definitions, controller stubs, and test files for the new managed resource kind.
3. THE build system SHALL use the Crossplane "build" Make submodule for CI/CD targets.
4. WHEN `make build` is executed, THE build system SHALL compile separate provider Go binaries for each family member: `cmd/family-provider/main.go`, `cmd/provider-conversation/main.go`, and `cmd/provider-usergroup/main.go`.
5. WHEN `make docker-build` is executed, THE build system SHALL build separate container images for each family member on a distroless base.
6. WHEN `make docker-push` is executed, THE build system SHALL push all family member container images to the configured container registry.
7. WHEN `make xpkg-build` is executed, THE build system SHALL produce separate Crossplane packages (`.xpkg`) for each family member: `provider-family-slack`, `provider-slack-conversation`, and `provider-slack-usergroup`.
8. THE `provider-family-slack` package SHALL contain only the ProviderConfig CRD and the family provider controller binary.
9. THE `provider-slack-conversation` package SHALL contain the Conversation, ConversationBookmark, and ConversationPin CRDs and the conversation provider controller binary.
10. THE `provider-slack-usergroup` package SHALL contain the UserGroup and UserGroupMembers CRDs and the usergroup provider controller binary.
11. WHEN `make xpkg-push` is executed, THE build system SHALL push all three Crossplane packages to fully qualified registry URLs: `ghcr.io/avodah-inc/provider-family-slack`, `ghcr.io/avodah-inc/provider-slack-conversation`, and `ghcr.io/avodah-inc/provider-slack-usergroup`.

### Requirement 12: Provider Installation and Integration

**User Story:** As a platform engineer, I want to install individual family member providers via Crossplane Provider resources with DeploymentRuntimeConfig support, so that the provider integrates with the existing Flux-managed EKS cluster infrastructure and only the needed resource families are deployed.

#### Acceptance Criteria

1. WHEN a Crossplane Provider resource referencing `ghcr.io/avodah-inc/provider-slack-conversation:v0.1.0` is applied, THE Crossplane runtime SHALL automatically install `provider-family-slack` as a dependency and start both controller pods.
2. WHEN a Crossplane Provider resource referencing `ghcr.io/avodah-inc/provider-slack-usergroup:v0.1.0` is applied, THE Crossplane runtime SHALL automatically install `provider-family-slack` as a dependency if not already present and start the usergroup controller pod.
3. WHEN the Family_Provider pod starts, THE Family_Provider SHALL register the ProviderConfig controller.
4. WHEN the `provider-slack-conversation` pod starts, THE Provider SHALL register the Conversation, ConversationBookmark, and ConversationPin managed resource controllers.
5. WHEN the `provider-slack-usergroup` pod starts, THE Provider SHALL register the UserGroup and UserGroupMembers managed resource controllers.
6. IF a provider pod fails to start due to missing RBAC permissions, THEN THE Provider SHALL emit a Kubernetes event describing the missing permission.
7. THE Provider installation manifests SHALL reference a DeploymentRuntimeConfig resource for Pod-level settings (resource limits, environment variables, tolerations) instead of the removed ControllerConfig.
8. ALL Crossplane package references in Provider resources SHALL use fully qualified image URLs including the registry (e.g., `ghcr.io/avodah-inc/provider-family-slack:v0.1.0`).

### Requirement 13: Family-Scoped Provider Architecture

**User Story:** As a platform engineer, I want the provider structured as a family of independently installable packages, so that I can deploy only the resource families I need, reduce memory footprint, and update each family independently.

#### Acceptance Criteria

1. THE Provider repository SHALL produce three distinct Crossplane packages: `provider-family-slack`, `provider-slack-conversation`, and `provider-slack-usergroup`.
2. THE `provider-family-slack` package SHALL own the ProviderConfig CRD and run as its own controller pod.
3. THE `provider-slack-conversation` package SHALL manage Conversation, ConversationBookmark, and ConversationPin resources and run as its own controller pod.
4. THE `provider-slack-usergroup` package SHALL manage UserGroup and UserGroupMembers resources and run as its own controller pod.
5. THE `provider-slack-conversation` package metadata (`crossplane.yaml`) SHALL declare a dependency on `provider-family-slack` so that the Family_Provider is automatically installed.
6. THE `provider-slack-usergroup` package metadata (`crossplane.yaml`) SHALL declare a dependency on `provider-family-slack` so that the Family_Provider is automatically installed.
7. WHEN only `provider-slack-conversation` is installed, THE cluster SHALL run exactly two controller pods: one for the Family_Provider and one for the conversation family member.
8. WHEN both `provider-slack-conversation` and `provider-slack-usergroup` are installed, THE cluster SHALL run exactly three controller pods: one for the Family_Provider, one for the conversation family member, and one for the usergroup family member.
9. THE Family_Provider SHALL share the ProviderConfig across all installed family members without requiring duplicate credential configuration.

### Requirement 14: Namespaced Managed Resource Support

**User Story:** As a platform engineer, I want managed resources to support both namespaced and cluster-scoped modes, so that the provider is compatible with Crossplane v2 defaults while maintaining backward compatibility.

#### Acceptance Criteria

1. THE Provider SHALL register each managed resource kind with both a cluster-scoped CRD at API group `slack.crossplane.io/v1alpha1` and a namespaced CRD using the `.m` API group suffix convention (e.g., `conversation.slack.m.crossplane.io/v1alpha1`).
2. WHEN a namespaced Managed_Resource is created in a specific namespace, THE Provider SHALL reconcile the resource using the ProviderConfig referenced in the resource spec.
3. WHEN a cluster-scoped Managed_Resource is created, THE Provider SHALL reconcile the resource using the ProviderConfig referenced in the resource spec.
4. THE Provider SHALL apply identical reconciliation logic for both namespaced and cluster-scoped variants of the same managed resource kind.
5. THE namespaced Managed_Resource variants SHALL use the same `spec.forProvider`, `spec.providerConfigRef`, and `status.atProvider` schema as their cluster-scoped counterparts.

### Requirement 15: crossplane-runtime Integration Patterns

**User Story:** As a provider developer, I want the provider to follow standard crossplane-runtime patterns for controller setup and type definitions, so that the codebase is consistent with the Crossplane ecosystem and maintainable.

#### Acceptance Criteria

1. THE Provider SHALL use `managed.NewReconciler` from `crossplane-runtime` to create the reconciler for each managed resource controller.
2. THE Provider SHALL implement the `managed.ExternalConnecter` interface to create ExternalClient instances from ProviderConfig references.
3. THE Provider SHALL implement the ExternalClient interface with Observe, Create, Update, and Delete methods for each managed resource kind.
4. THE Provider SHALL embed `xpv1.ResourceSpec` in the spec type and `xpv1.ResourceStatus` in the status type of every managed resource to satisfy the `resource.Managed` interface.
5. WHEN `make generate` is executed, THE build system SHALL use angryjet to generate getter and setter methods for all managed resource types.
6. THE Provider SHALL use the `crossplane.io/external-name` annotation pattern to store and retrieve external resource identifiers.
