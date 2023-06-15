/*
Copyright 2023.

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

package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var ironicdatabaselog = logf.Log.WithName("ironicdatabase-resource")

func (r *IronicDatabase) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-metal3-io-v1alpha1-ironicdatabase,mutating=true,failurePolicy=fail,sideEffects=None,groups=metal3.io,resources=ironicdatabases,verbs=create;update,versions=v1alpha1,name=mironicdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &IronicDatabase{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *IronicDatabase) Default() {}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-metal3-io-v1alpha1-ironicdatabase,mutating=false,failurePolicy=fail,sideEffects=None,groups=metal3.io,resources=ironicdatabases,verbs=create;update,versions=v1alpha1,name=vironicdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &IronicDatabase{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *IronicDatabase) ValidateCreate() error {
	ironicdatabaselog.Info("validate create", "name", r.Name)
	return validateDatabase(&r.Spec, nil)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *IronicDatabase) ValidateUpdate(old runtime.Object) error {
	ironicdatabaselog.Info("validate update", "name", r.Name)
	return validateDatabase(&r.Spec, &old.(*IronicDatabase).Spec)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *IronicDatabase) ValidateDelete() error {
	return nil
}

func validateDatabase(db *IronicDatabaseSpec, old *IronicDatabaseSpec) error {
	if db.ExternalIP != "" && db.CredentialsSecretName == "" {
		return errors.New("external database requires credentials")
	}

	return nil
}
