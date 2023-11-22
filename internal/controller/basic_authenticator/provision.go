package basic_authenticator

import (
	"context"
	errors2 "errors"
	"github.com/opdev/subreconciler"
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const basicAuthenticatorFinalizer = "basicauthenticators.authenticator.snappcloud.io/finalizer"

// Provision provisions the required resources for the basicAuthenticator object
func (r *BasicAuthenticatorReconciler) Provision(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Do the actual reconcile work
	subProvisioner := []subreconciler.FnWithRequest{
		r.secretProvisioner,
		r.configmapProvisioner,
		r.deploymentProvisioner,
	}
	for _, provisioner := range subProvisioner {
		result, err := provisioner(ctx, req)
		if subreconciler.ShouldHaltOrRequeue(result, err) {
			return subreconciler.Evaluate(result, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}

func (r *BasicAuthenticatorReconciler) GetLatestBasicAuthenticator(ctx context.Context, req ctrl.Request, basicAuthenticator *v1alpha1.BasicAuthenticator) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	err := r.Get(ctx, req.NamespacedName, basicAuthenticator)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found. Ignoring since object must be deleted")
			return subreconciler.DoNotRequeue()
		}
		logger.Error(err, "Failed to get BasicAuthenticator")
		return subreconciler.RequeueWithError(err)
	}
	return subreconciler.ContinueReconciling()
}

func (r *BasicAuthenticatorReconciler) secretProvisioner(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.GetLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}
	credentialName := basicAuthenticator.Spec.CredentialsSecretRef
	var credentialSecret corev1.Secret
	if credentialName == "" {
		//create secret
		newSecret := r.CreateCredentials(basicAuthenticator)
		err := r.Get(ctx, types.NamespacedName{Name: newSecret.Name, Namespace: newSecret.Namespace}, &credentialSecret)
		if errors.IsNotFound(err) {
			// update basic auth
			err := r.Create(ctx, newSecret)
			if err != nil {
				logger.Error(err, "failed to create new secret")
				return subreconciler.RequeueWithError(err)
			}

			credentialName = newSecret.Name
			credentialSecret = *newSecret
			basicAuthenticator.Spec.CredentialsSecretRef = credentialName

			//saving secretName inorder to be used in next steps
			assignAnnotation(basicAuthenticator, SecretAnnotation, credentialName)
			err = r.Update(ctx, basicAuthenticator)
			if err != nil {
				logger.Error(err, "failed to updated basic authenticator")
				return subreconciler.RequeueWithError(err)
			}

		} else {
			return subreconciler.Requeue()
		}
	} else {
		err := r.Get(ctx, types.NamespacedName{Name: credentialName, Namespace: basicAuthenticator.Namespace}, &credentialSecret)
		if err != nil {
			logger.Error(err, "failed to fetch secret")
			return subreconciler.RequeueWithError(err)
		}

		//saving secretName inorder to be used in next steps
		assignAnnotation(basicAuthenticator, SecretAnnotation, credentialName)
		err = r.Update(ctx, basicAuthenticator)
		if err != nil {
			logger.Error(err, "failed to updated basic authenticator")
			return subreconciler.RequeueWithError(err)
		}
	}
	return subreconciler.ContinueReconciling()
}

func (r *BasicAuthenticatorReconciler) configmapProvisioner(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.GetLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	authenticatorConfig := r.CreateNginxConfigmap(basicAuthenticator)
	var foundConfigmap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: authenticatorConfig.Name, Namespace: basicAuthenticator.Namespace}, &foundConfigmap)
	if errors.IsNotFound(err) {
		if err := ctrl.SetControllerReference(basicAuthenticator, authenticatorConfig, r.Scheme); err != nil {
			logger.Error(err, "failed to set configmap owner")
			return subreconciler.RequeueWithError(err)
		}
		err := r.Create(ctx, authenticatorConfig)
		if err != nil {
			logger.Error(err, "failed to create new configmap")
			return subreconciler.RequeueWithError(err)
		}
		//saving secretName inorder to be used in next steps
		assignAnnotation(basicAuthenticator, ConfigmapAnnotation, authenticatorConfig.Name)

		err = r.Update(ctx, basicAuthenticator)
		if err != nil {
			logger.Error(err, "failed to updated basic authenticator")
			return subreconciler.RequeueWithError(err)
		}
		return subreconciler.Requeue()
	} else if err != nil {
		logger.Error(err, "failed to fetch configmap")
		return subreconciler.RequeueWithError(err)
	} else {
		if !reflect.DeepEqual(authenticatorConfig.Data, foundConfigmap.Data) {
			logger.Info("updating configmap")
			foundConfigmap.Data = authenticatorConfig.Data
			err := r.Update(ctx, &foundConfigmap)
			if err != nil {
				logger.Error(err, "failed to update configmap")
				return subreconciler.RequeueWithError(err)
			}
		}
		assignAnnotation(basicAuthenticator, ConfigmapAnnotation, authenticatorConfig.Name)
		err = r.Update(ctx, basicAuthenticator)
		if err != nil {
			logger.Error(err, "failed to updated basic authenticator")
			return subreconciler.RequeueWithError(err)
		}
	}

	return subreconciler.ContinueReconciling()
}

func (r *BasicAuthenticatorReconciler) deploymentProvisioner(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.GetLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}
	if basicAuthenticator.ObjectMeta.Annotations == nil {
		return subreconciler.RequeueWithError(errors2.New("configmap annotation and secret annotation must be set"))

	}
	authenticatorConfigName, configmapExists := basicAuthenticator.ObjectMeta.Annotations[ConfigmapAnnotation]
	if !configmapExists {
		return subreconciler.RequeueWithError(errors2.New("configmap annotation not set"))
	}

	secretName, secretExists := basicAuthenticator.ObjectMeta.Annotations[SecretAnnotation]
	if !secretExists {
		return subreconciler.RequeueWithError(errors2.New("secret annotation not set"))
	}
	//Deciding to create sidecar injection or create deployment
	isSidecar := basicAuthenticator.Spec.Type == "sidecar"
	if isSidecar {
		deploymentsToUpdate, err := r.Injector(ctx, basicAuthenticator, authenticatorConfigName, secretName)
		if err != nil {
			logger.Error(err, "failed to inject into deployments")
			return subreconciler.RequeueWithError(err)
		}
		for _, deploy := range deploymentsToUpdate {
			if err := ctrl.SetControllerReference(basicAuthenticator, deploy, r.Scheme); err != nil {
				logger.Error(err, "failed to set injected deployment owner")
				return subreconciler.RequeueWithError(err)
			}
			err := r.Update(ctx, deploy)
			if err != nil {
				logger.Error(err, "failed to update injected deployments")
				return subreconciler.RequeueWithError(err)
			}
		}
	} else {
		newDeployment := r.CreateNginxDeployment(basicAuthenticator, authenticatorConfigName, secretName)
		foundDeployment := &appv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{Name: newDeployment.Name, Namespace: basicAuthenticator.Namespace}, foundDeployment)
		if errors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(basicAuthenticator, newDeployment, r.Scheme); err != nil {
				logger.Error(err, "failed to set deployment owner")
				return subreconciler.RequeueWithError(err)
			}
			//create deployment
			err := r.Create(ctx, newDeployment)
			if err != nil {
				logger.Error(err, "failed to create new deployment")
				return subreconciler.RequeueWithError(err)
			}
			logger.Info("created deployment")

			return subreconciler.Requeue()
		} else if err != nil {
			if err != nil {
				logger.Error(err, "failed to fetch deployment")
				return subreconciler.RequeueWithError(err)
			}
		} else {
			//update deployment

			if !reflect.DeepEqual(newDeployment.Spec, foundDeployment.Spec) {
				logger.Info("updating deployment")
				foundDeployment.Spec = newDeployment.Spec
				err := r.Update(ctx, foundDeployment)
				if err != nil {
					logger.Error(err, "failed to update deployment")
					return subreconciler.RequeueWithError(err)
				}
			}
			logger.Info("updating ready replicas")
			basicAuthenticator.Status.ReadyReplicas = int(foundDeployment.Status.ReadyReplicas)
			err := r.Status().Update(ctx, basicAuthenticator)
			if err != nil {
				logger.Error(err, "failed to update basic authenticator status")
				return subreconciler.RequeueWithError(err)
			}
		}
	}

	return subreconciler.ContinueReconciling()
}
func assignAnnotation(authenticator *v1alpha1.BasicAuthenticator, key, value string) {
	if authenticator.ObjectMeta.Annotations == nil {
		authenticator.ObjectMeta.Annotations = make(map[string]string)
	}
	authenticator.ObjectMeta.Annotations[key] = value
}