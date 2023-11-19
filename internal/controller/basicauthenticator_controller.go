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

package controller

import (
	"context"
	"github.com/sinamna/BasicAthenticator/internal/config"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authenticatorv1alpha1 "github.com/sinamna/BasicAthenticator/api/v1alpha1"
)

// BasicAuthenticatorReconciler reconciles a BasicAuthenticator object
type BasicAuthenticatorReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	CustomConfig *config.CustomConfig
}

//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BasicAuthenticator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BasicAuthenticatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile triggered")

	basicAuthenticator := &authenticatorv1alpha1.BasicAuthenticator{}
	err := r.Get(ctx, req.NamespacedName, basicAuthenticator)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BasicAuthenticator")
		return ctrl.Result{}, err
	}
	basicAuthenticator.Status.Ready = true
	err = r.Status().Update(ctx, basicAuthenticator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, req.NamespacedName, basicAuthenticator); err != nil {
		logger.Error(err, "failed to refetch")
		return ctrl.Result{}, err
	}
	////TODO:handle deletion scenario and clean up
	//if basicAuthenticator.GetDeletionTimestamp() != nil {
	//	// clean up
	//}

	credentialName := basicAuthenticator.Spec.CredentialsSecretRef
	var credentialSecret *corev1.Secret
	if credentialName == "" {
		//create secret
		secret := r.CreateCredentials(basicAuthenticator)
		// update basic auth
		err := r.Create(ctx, secret)
		if err != nil {
			logger.Error(err, "failed to create new secret")
			return ctrl.Result{}, err
		}
		credentialName = secret.Name
		credentialSecret = secret
	} else {
		err := r.Get(ctx, types.NamespacedName{Name: credentialName, Namespace: basicAuthenticator.Namespace}, credentialSecret)
		if err != nil {
			logger.Error(err, "failed to fetch secret")
			return ctrl.Result{}, err
		}
	}

	//create configmap
	nginxConfig := r.CreateNginxConfigmap(basicAuthenticator)

	//create deployment or sidecar
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BasicAuthenticatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authenticatorv1alpha1.BasicAuthenticator{}).
		Owns(&appv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
