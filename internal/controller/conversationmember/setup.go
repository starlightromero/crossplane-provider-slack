/*
Copyright 2026 Starlight Romero.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package conversationmember

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	conversationmemberv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversationmember/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	slack "github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/features"
)

// Setup adds a controller that reconciles ConversationMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return setup(mgr, o)
}

// SetupGated adds a controller that reconciles ConversationMember managed resources
// only if the EnableAlphaConversationMember feature flag is enabled.
func SetupGated(mgr ctrl.Manager, o controller.Options, f *feature.Flags) error {
	if f != nil && !f.Enabled(features.EnableAlphaConversationMember) {
		return nil
	}
	return setup(mgr, o)
}

func setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(conversationmemberv1alpha1.ConversationMemberGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(conversationmemberv1alpha1.ConversationMemberGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
			usage: resource.NewLegacyProviderConfigUsageTracker(
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
		For(&conversationmemberv1alpha1.ConversationMember{}).
		Complete(r)
}
