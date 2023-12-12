package basic_authenticator

import (
	"context"
	"errors"
	"github.com/opdev/subreconciler"
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *BasicAuthenticatorReconciler) Cleanup(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Do the actual reconcile work
	subRecs := []subreconciler.FnWithRequest{
		r.setDeletionStatus,
		r.removeInjectedContainers,
		r.removeCleanupFinalizer,
	}
	for _, rec := range subRecs {
		result, err := rec(ctx, req)
		if subreconciler.ShouldHaltOrRequeue(result, err) {
			return subreconciler.Evaluate(result, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}
func (r *BasicAuthenticatorReconciler) setDeletionStatus(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}
	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.RequeueWithError(err)
	}
	basicAuthenticator.Status.State = StatusDeleting

	if err := r.Update(ctx, basicAuthenticator); err != nil {
		r.logger.Error(err, "Failed to update status while cleaning")
		return subreconciler.RequeueWithError(err)
	}
	return subreconciler.ContinueReconciling()
}
func (r *BasicAuthenticatorReconciler) removeInjectedContainers(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}

	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.RequeueWithError(err)
	}

	if basicAuthenticator.Spec.Type != "sidecar" {
		return subreconciler.ContinueReconciling()
	}
	basicAuthLabel := map[string]string{
		basicAuthenticatorNameLabel: basicAuthenticator.Name,
	}
	deployments, err := getTargetDeployment(ctx, basicAuthenticator, r.Client, basicAuthLabel)
	if err != nil {
		r.logger.Error(err, "failed to get target deployments to clean up")
		return subreconciler.RequeueWithError(err)
	}
	configmaps, err := getTargetConfigmapNames(ctx, basicAuthenticator, r.Client, basicAuthLabel)
	if err != nil {
		r.logger.Error(err, "failed to get target configmap to clean up")
		return subreconciler.RequeueWithError(err)
	}
	secrets, err := getTargetSecretName(ctx, basicAuthenticator, r.Client, basicAuthLabel)
	if err != nil {
		r.logger.Error(err, "failed to get target secret to clean up")
		return subreconciler.RequeueWithError(err)
	}
	r.logger.Info("debug", "configmap", configmaps, "secret", secrets)

	cleanupDeployments := removeInjectedResources(deployments, secrets, configmaps)
	for _, deploy := range cleanupDeployments {
		if err := r.Update(ctx, deploy); err != nil {
			r.logger.Error(err, "failed to add update cleaned up deployments")
			return subreconciler.RequeueWithError(err)
		}
	}
	return subreconciler.ContinueReconciling()
}
func (r *BasicAuthenticatorReconciler) removeCleanupFinalizer(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	basicAuthenticator := &v1alpha1.BasicAuthenticator{}
	if r, err := r.getLatestBasicAuthenticator(ctx, req, basicAuthenticator); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.RequeueWithError(err)
	}
	if controllerutil.ContainsFinalizer(basicAuthenticator, basicAuthenticatorFinalizer) {
		if ok := controllerutil.RemoveFinalizer(basicAuthenticator, basicAuthenticatorFinalizer); !ok {
			r.logger.Error(errors.New("finalizer not updated"), "Failed to remove finalizer for BasicAuthenticator")
			return subreconciler.Requeue()
		}
	}

	if err := r.Update(ctx, basicAuthenticator); err != nil {
		r.logger.Error(err, "Failed to remove finalizer for BasicAuthenticator")
		return subreconciler.RequeueWithError(err)
	}
	return subreconciler.ContinueReconciling()
}

func getTargetDeployment(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, k8Client client.Client, basicAuthLabels map[string]string) ([]*appsv1.Deployment, error) {
	var deploymentList appsv1.DeploymentList
	if err := k8Client.List(
		ctx,
		&deploymentList,
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(basicAuthLabels)},
		client.InNamespace(basicAuthenticator.Namespace)); err != nil {
		return nil, err
	}

	resultDeployments := make([]*appsv1.Deployment, 0)
	for _, deploy := range deploymentList.Items {
		resultDeployments = append(resultDeployments, &deploy)
	}
	return resultDeployments, nil
}
func getTargetConfigmapNames(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, k8Client client.Client, basicAuthLabels map[string]string) ([]string, error) {
	var configMapList v1.ConfigMapList
	if err := k8Client.List(
		ctx,
		&configMapList,
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(basicAuthLabels)},
		client.InNamespace(basicAuthenticator.Namespace)); err != nil {
		return nil, err
	}
	resultConfigMaps := make([]string, 0)
	for _, cm := range configMapList.Items {
		resultConfigMaps = append(resultConfigMaps, cm.Name)
	}
	return resultConfigMaps, nil
}
func getTargetSecretName(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, k8Client client.Client, basicAuthLabels map[string]string) ([]string, error) {
	var secretList v1.SecretList
	if err := k8Client.List(
		ctx,
		&secretList,
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(basicAuthLabels)},
		client.InNamespace(basicAuthenticator.Namespace)); err != nil {
		return nil, err
	}

	resultSecrets := make([]string, 0)
	for _, sec := range secretList.Items {
		resultSecrets = append(resultSecrets, sec.Name)
	}
	return resultSecrets, nil
}
func removeInjectedResources(deployments []*appsv1.Deployment, secrets []string, configmap []string) []*appsv1.Deployment {
	for _, deploy := range deployments {
		containers := make([]v1.Container, 0)
		for _, container := range deploy.Spec.Template.Spec.Containers {
			if container.Name != nginxDefaultContainerName {
				containers = append(containers, container)
			}
		}
		deploy.Spec.Template.Spec.Containers = containers
		volumes := make([]v1.Volume, 0)
		for _, vol := range deploy.Spec.Template.Spec.Volumes {
			if !existsInList(secrets, vol.Name) && !existsInList(configmap, vol.Name) {
				volumes = append(volumes, vol)
			}
		}
		deploy.Spec.Template.Spec.Volumes = volumes
		if deploy.Labels != nil {
			delete(deploy.Labels, basicAuthenticatorNameLabel)
		}
	}
	return deployments
}

func existsInList(strList []string, targetStr string) bool {
	for _, val := range strList {
		if val == targetStr {
			return true
		}
	}
	return false
}
