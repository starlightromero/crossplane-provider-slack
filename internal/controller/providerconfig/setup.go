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

package providerconfig

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
)

const (
	shortWait = 30 * time.Second
	timeout   = 2 * time.Minute

	errGetPC        = "cannot get ProviderConfig"
	errUpdateStatus = "cannot update ProviderConfig status"
)

// Setup adds a controller that reconciles ProviderConfig resources. It handles
// usage tracking (via the crossplane-runtime providerconfig reconciler) and
// credential validation (setting Ready conditions based on Secret contents).
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName("ProviderConfig")

	of := resource.ProviderConfigKinds{
		Config:    v1alpha1.SchemeGroupVersion.WithKind("ProviderConfig"),
		Usage:     v1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsage"),
		UsageList: v1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsageList"),
	}

	// The standard crossplane-runtime providerconfig reconciler handles usage
	// tracking and finalizer management.
	pcr := providerconfig.NewReconciler(mgr, of,
		providerconfig.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	// Our custom reconciler wraps the standard one and adds credential
	// validation logic.
	r := &reconciler{
		client: mgr.GetClient(),
		pcr:    pcr,
		log:    logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name)),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.ProviderConfig{}).
		Watches(&v1alpha1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(r)
}

// reconciler reconciles ProviderConfig resources by delegating usage tracking
// to the standard crossplane-runtime reconciler and adding credential
// validation.
type reconciler struct {
	client client.Client
	pcr    *providerconfig.Reconciler
	log    logging.Logger
}

// Reconcile handles a ProviderConfig reconciliation request.
func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling ProviderConfig")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get the ProviderConfig.
	pc := &v1alpha1.ProviderConfig{}
	if err := r.client.Get(ctx, req.NamespacedName, pc); err != nil {
		log.Debug(errGetPC, "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetPC)
	}

	// If the ProviderConfig is being deleted, delegate to the standard
	// reconciler for finalizer and usage cleanup.
	if meta.WasDeleted(pc) {
		return r.pcr.Reconcile(ctx, req)
	}

	// Delegate to the standard reconciler for usage tracking and finalizer
	// management.
	result, err := r.pcr.Reconcile(ctx, req)
	if err != nil {
		return result, err
	}

	// Re-fetch the ProviderConfig after the standard reconciler may have
	// updated it.
	if err := r.client.Get(ctx, req.NamespacedName, pc); err != nil {
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetPC)
	}

	// Validate credentials and set Ready condition.
	token, extractErr := ExtractToken(ctx, r.client, pc)
	SetCredentialCondition(pc, token, extractErr)

	if err := r.client.Status().Update(ctx, pc); err != nil {
		log.Debug(errUpdateStatus, "error", err)
		return reconcile.Result{RequeueAfter: shortWait}, nil
	}

	return result, nil
}
