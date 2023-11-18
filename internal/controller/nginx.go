package controller

import (
	"github.com/sinamna/BasicAthenticator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

func (r *BasicAuthenticatorReconciler) CreateBasicAuthenticatorDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator) (*appsv1.Deployment, error) {
	deploy := appsv1.Deployment{}

}
