package basic_authenticator

import (
	"context"
	defaultError "errors"
	"github.com/opdev/subreconciler"
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"math"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Provision provisions the required resources for the basicAuthenticator object
func (r *BasicAuthenticatorReconciler) Provision(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Do the actual reconcile work
	subProvisioner := []subreconciler.FnWithRequest{
		r.ensureSecret,
		r.ensureConfigmap,
		r.ensureDeployment,
	}
	for _, provisioner := range subProvisioner {
		result, err := provisioner(ctx, req)
		if subreconciler.ShouldHaltOrRequeue(result, err) {
			return subreconciler.Evaluate(result, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}

func (r *BasicAuthenticatorReconciler) getLatestBasicAuthenticator(ctx context.Context, req ctrl.Request, basicAuthenticator *v1alpha1.BasicAuthenticator) (*ctrl.Result, error) {
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

func (r *BasicAuthenticatorReconciler) ensureSecret(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}
	credentialName := basicAuthenticator.Spec.CredentialsSecretRef
	var credentialSecret corev1.Secret
	if credentialName == "" {
		//create secret
		newSecret := createCredentials(basicAuthenticator)
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

func (r *BasicAuthenticatorReconciler) ensureConfigmap(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	authenticatorConfig := createNginxConfigmap(basicAuthenticator)
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

func (r *BasicAuthenticatorReconciler) ensureDeployment(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	_ = log.FromContext(ctx)
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}
	if basicAuthenticator.ObjectMeta.Annotations == nil {
		return subreconciler.RequeueWithError(defaultError.New("configmap annotation and secret annotation must be set"))

	}
	authenticatorConfigName, configmapExists := basicAuthenticator.ObjectMeta.Annotations[ConfigmapAnnotation]
	if !configmapExists {
		return subreconciler.RequeueWithError(defaultError.New("configmap annotation not set"))
	}

	secretName, secretExists := basicAuthenticator.ObjectMeta.Annotations[SecretAnnotation]
	if !secretExists {
		return subreconciler.RequeueWithError(defaultError.New("secret annotation not set"))
	}
	//Deciding to create sidecar injection or create deployment
	isSidecar := basicAuthenticator.Spec.Type == "sidecar"
	if isSidecar {
		return r.createSidecarAuthenticator(ctx, req, basicAuthenticator, authenticatorConfigName, secretName)
	} else {
		return r.createDeploymentAuthenticator(ctx, req, basicAuthenticator, authenticatorConfigName, secretName)
	}
}

func (r *BasicAuthenticatorReconciler) createDeploymentAuthenticator(ctx context.Context, req ctrl.Request, basicAuthenticator *v1alpha1.BasicAuthenticator, authenticatorConfigName, secretName string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	newDeployment := createNginxDeployment(basicAuthenticator, authenticatorConfigName, secretName, r.CustomConfig)
	foundDeployment := &appv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: newDeployment.Name, Namespace: basicAuthenticator.Namespace}, foundDeployment)
	if errors.IsNotFound(err) {
		if err := ctrl.SetControllerReference(basicAuthenticator, newDeployment, r.Scheme); err != nil {
			logger.Error(err, "failed to set deployment owner")
			return subreconciler.RequeueWithError(err)
		}
		if basicAuthenticator.Spec.AdaptiveScale && basicAuthenticator.Spec.AppService != "" {
			replica, err := r.acquireTargetReplica(ctx, basicAuthenticator)
			if err != nil {
				logger.Error(err, "failed to acquire target replica using adaptiveScale")
				return subreconciler.RequeueWithError(err)
			}
			newDeployment.Spec.Replicas = &replica
		}

		//create deployment
		err := r.Create(ctx, newDeployment)
		if err != nil {
			logger.Error(err, "failed to create new deployment")
			return subreconciler.RequeueWithError(err)
		}
		err = r.Get(ctx, types.NamespacedName{Name: foundDeployment.Name, Namespace: basicAuthenticator.Namespace}, foundDeployment)
		if err != nil {
			logger.Error(err, "failed to refetch")
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
		targetReplica := newDeployment.Spec.Replicas
		if basicAuthenticator.Spec.AdaptiveScale && basicAuthenticator.Spec.AppService != "" {
			replica, err := r.acquireTargetReplica(ctx, basicAuthenticator)
			if err != nil {
				logger.Error(err, "failed to acquire target replica using adaptiveScale")
			}
			targetReplica = &replica
		}

		if !reflect.DeepEqual(newDeployment.Spec, foundDeployment.Spec) {
			logger.Info("updating deployment")

			foundDeployment.Spec = newDeployment.Spec
			foundDeployment.Spec.Replicas = targetReplica

			err := r.Update(ctx, foundDeployment)
			if err != nil {
				logger.Error(err, "failed to update deployment")
				return subreconciler.RequeueWithError(err)
			}
			err = r.Get(ctx, types.NamespacedName{Name: foundDeployment.Name, Namespace: basicAuthenticator.Namespace}, foundDeployment)
			if err != nil {
				logger.Error(err, "failed to refetch")
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
	return subreconciler.ContinueReconciling()
}

func (r *BasicAuthenticatorReconciler) createSidecarAuthenticator(ctx context.Context, req ctrl.Request, basicAuthenticator *v1alpha1.BasicAuthenticator, authenticatorConfigName, secretName string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	deploymentsToUpdate, err := injector(ctx, basicAuthenticator, authenticatorConfigName, secretName, r.CustomConfig, r.Client)
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
	return subreconciler.ContinueReconciling()
}

func (r *BasicAuthenticatorReconciler) acquireTargetReplica(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator) (int32, error) {
	var targetService corev1.Service
	// service should be in same ns with basic auth
	if err := r.Get(ctx, types.NamespacedName{Name: basicAuthenticator.Spec.AppService, Namespace: basicAuthenticator.ObjectMeta.Namespace}, &targetService); err != nil {
		return -1, err
	}
	labelSelector := targetService.Spec.Selector

	deployments := &appv1.DeploymentList{}
	if err := r.List(ctx, deployments, client.MatchingLabels(labelSelector)); err != nil {
		return -1, err
	}

	if len(deployments.Items) == 0 {
		return -1, defaultError.New("no deployment is selected by appService")
	}

	targetDeploy := deployments.Items[0] //we expect it to be single deployment
	if targetDeploy.ObjectMeta.Annotations == nil {
		targetDeploy.ObjectMeta.Annotations = make(map[string]string)
	}
	log.FromContext(context.Background()).Info("----- debug", "deployment revision:", targetDeploy.ResourceVersion)

	targetDeploy.ObjectMeta.Annotations[ExternallyManaged] = basicAuthenticator.Name

	err := r.Update(ctx, &targetDeploy)
	if err != nil {
		return -1, err
	}
	replicas := deployments.Items[0].Spec.Replicas
	targetReplica := math.Floor(float64((*replicas + 1) / 2))
	log.FromContext(context.Background()).Info("-------debug", "target Replica inside", targetReplica)

	return int32(targetReplica), nil
}
