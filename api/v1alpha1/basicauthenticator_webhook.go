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
	"context"
	"errors"
	htpasswd "github.com/snapp-incubator/simple-authenticator/pkg/htpasswd"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"strings"
	"time"
)

var (
	runtimeClient     client.Client
	ValidationTimeout time.Duration
)

// log is for logging in this package.
var basicauthenticatorlog = logf.Log.WithName("basicauthenticator-resource")

func (r *BasicAuthenticator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	runtimeClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-authenticator-snappcloud-io-v1alpha1-basicauthenticator,mutating=true,failurePolicy=fail,sideEffects=None,groups=authenticator.snappcloud.io,resources=basicauthenticators,verbs=create;update,versions=v1alpha1,name=mbasicauthenticator.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &BasicAuthenticator{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *BasicAuthenticator) Default() {
	basicauthenticatorlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-authenticator-snappcloud-io-v1alpha1-basicauthenticator,mutating=false,failurePolicy=fail,sideEffects=None,groups=authenticator.snappcloud.io,resources=basicauthenticators,verbs=create;update,versions=v1alpha1,name=vbasicauthenticator.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &BasicAuthenticator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BasicAuthenticator) ValidateCreate() error {
	basicauthenticatorlog.Info("validate create", "name", r.Name)

	if err := r.validateCredentials(); err != nil {
		basicauthenticatorlog.Error(err, "Failed to validate credentials")
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BasicAuthenticator) ValidateUpdate(old runtime.Object) error {
	basicauthenticatorlog.Info("validate update", "name", r.Name)

	if err := r.validateCredentials(); err != nil {
		basicauthenticatorlog.Error(err, "Failed to validate credentials")
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BasicAuthenticator) ValidateDelete() error {
	basicauthenticatorlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *BasicAuthenticator) validateCredentials() error {
	secretName := r.Spec.CredentialsSecretRef
	if secretName == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ValidationTimeout)
	defer cancel()
	var credentials v1.Secret

	err := runtimeClient.Get(ctx, types.NamespacedName{Namespace: r.Namespace, Name: secretName}, &credentials)
	if err != nil {
		basicauthenticatorlog.Error(err, "failed to fetch secret")
		return err
	}

	htpasswdByte, exists := credentials.Data[".htpasswd"]
	if !exists {
		return errors.New("illegal format. data missing \".htpasswd\" field")
	}
	htpasswdStr := string(htpasswdByte)
	if !htpasswd.ValidateHtpasswdFormat(strings.TrimSpace(htpasswdStr)) {
		return errors.New("failed to validate format of htpasswd. htpasswd should be like \"username:password\"")
	}
	return nil
}
