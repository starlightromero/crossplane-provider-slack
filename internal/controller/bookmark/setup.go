/*
Copyright 2024 Avodah Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bookmark

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	bookmarkv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/bookmark/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	slack "github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/features"
)

// Setup adds a controller that reconciles ConversationBookmark managed resources.
// TODO(poll-interval): WithPollInterval should be configured from ProviderConfig.spec.pollInterval.
// The actual wiring requires reading the ProviderConfig at setup time or dynamically in Connect,
// then passing the parsed duration to managed.NewReconciler via managed.WithPollInterval(duration).
// Default poll interval is 5 minutes when not set in ProviderConfig.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return setup(mgr, o)
}

// SetupGated adds a controller that reconciles ConversationBookmark managed resources
// only if the EnableAlphaConversationBookmark feature flag is enabled.
func SetupGated(mgr ctrl.Manager, o controller.Options, f *feature.Flags) error {
	if f != nil && !f.Enabled(features.EnableAlphaConversationBookmark) {
		return nil
	}
	return setup(mgr, o)
}

func setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(bookmarkv1alpha1.ConversationBookmarkGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(bookmarkv1alpha1.ConversationBookmarkGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(
				mgr.GetClient(),
				&v1alpha1.ProviderConfigUsage{},
			),
			newFn: func(token string, opts ...slack.ClientOption) slack.ClientAPI {
				return slack.NewClient(token, opts...)
			},
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&bookmarkv1alpha1.ConversationBookmark{}).
		Complete(r)
}
