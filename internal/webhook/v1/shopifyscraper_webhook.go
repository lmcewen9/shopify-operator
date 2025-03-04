/*
Copyright 2025.

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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lukemcewencomv1 "github.com/lmcewen9/shopify-crd/api/v1"
)

// nolint:unused
// log is for logging in this package.
var shopifyscraperlog = logf.Log.WithName("shopifyscraper-resource")

// SetupShopifyScraperWebhookWithManager registers the webhook for ShopifyScraper in the manager.
func SetupShopifyScraperWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&lukemcewencomv1.ShopifyScraper{}).
		WithValidator(&ShopifyScraperCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-lukemcewen-com-v1-shopifyscraper,mutating=false,failurePolicy=fail,sideEffects=None,groups=lukemcewen.com,resources=shopifyscrapers,verbs=create;update,versions=v1,name=vshopifyscraper-v1.kb.io,admissionReviewVersions=v1

// ShopifyScraperCustomValidator struct is responsible for validating the ShopifyScraper resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ShopifyScraperCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ShopifyScraperCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ShopifyScraper.
func (v *ShopifyScraperCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	shopifyscraper, ok := obj.(*lukemcewencomv1.ShopifyScraper)
	if !ok {
		return nil, fmt.Errorf("expected a ShopifyScraper object but got %T", obj)
	}
	shopifyscraperlog.Info("Validation for ShopifyScraper upon creation", "name", shopifyscraper.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ShopifyScraper.
func (v *ShopifyScraperCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	shopifyscraper, ok := newObj.(*lukemcewencomv1.ShopifyScraper)
	if !ok {
		return nil, fmt.Errorf("expected a ShopifyScraper object for the newObj but got %T", newObj)
	}
	shopifyscraperlog.Info("Validation for ShopifyScraper upon update", "name", shopifyscraper.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ShopifyScraper.
func (v *ShopifyScraperCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	shopifyscraper, ok := obj.(*lukemcewencomv1.ShopifyScraper)
	if !ok {
		return nil, fmt.Errorf("expected a ShopifyScraper object but got %T", obj)
	}
	shopifyscraperlog.Info("Validation for ShopifyScraper upon deletion", "name", shopifyscraper.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
