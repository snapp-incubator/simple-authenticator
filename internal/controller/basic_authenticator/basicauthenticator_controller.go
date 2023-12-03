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

package basic_authenticator

import (
	"context"
	"github.com/go-logr/logr"
	authenticatorv1alpha1 "github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	"github.com/snapp-incubator/simple-authenticator/internal/config"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// BasicAuthenticatorReconciler reconciles a BasicAuthenticator object
type BasicAuthenticatorReconciler struct {
	client.Client
	Scheme                      *runtime.Scheme
	CustomConfig                *config.CustomConfig
	configMapName               string
	credentialName              string
	basicAuthenticatorNamespace string
	logger                      logr.Logger
}

//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=authenticator.snappcloud.io,resources=basicauthenticators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *BasicAuthenticatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx)
	r.logger.Info("reconcile triggered")
	r.logger.Info(req.String())
	r.initVars(req)
	return r.Provision(ctx, req)
}

func (r *BasicAuthenticatorReconciler) initVars(request ctrl.Request) {
	r.basicAuthenticatorNamespace = request.Namespace
	//configmap name and credential name's value would be set in reconcile loop
}

// SetupWithManager sets up the controller with the Manager.
func (r *BasicAuthenticatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authenticatorv1alpha1.BasicAuthenticator{}).
		Owns(&appv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Watches(
			&source.Kind{Type: &appv1.Deployment{}},
			handler.EnqueueRequestsFromMapFunc(r.findExternallyManagedDeployments),
		).
		Complete(r)
}

func (r *BasicAuthenticatorReconciler) findExternallyManagedDeployments(deployment client.Object) []reconcile.Request {
	deploy, ok := deployment.(*appv1.Deployment)
	if !ok {
		return nil
	}
	if deploy.ObjectMeta.Annotations == nil {
		return nil
	}
	basicAuthName, exists := deploy.ObjectMeta.Annotations[ExternallyManaged]
	if !exists {
		return nil
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{Name: basicAuthName, Namespace: deploy.Namespace},
		},
	}
}
