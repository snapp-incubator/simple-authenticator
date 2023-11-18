package controller

import (
	"fmt"
	"github.com/sinamna/BasicAthenticator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ConfigMountPath = "/etc/nginx/conf.d"

// TODO: come up with better name that "nginx"
func (r *BasicAuthenticatorReconciler) CreateNginxDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string) *appsv1.Deployment {
	nginxImageAddress := "nginx/nginx"
	if r.CustomConfig != nil && r.CustomConfig.WebserverConf.Image != "" {
		nginxImageAddress = r.CustomConfig.WebserverConf.Image
	}

	nginxContainerName := "nginx"
	if r.CustomConfig != nil && r.CustomConfig.WebserverConf.ContainerName != "" {
		nginxContainerName = r.CustomConfig.WebserverConf.ContainerName
	}

	nginxName := fmt.Sprintf("%s-nginx", basicAuthenticator.Name)
	replicas := int32(basicAuthenticator.Spec.Replicas)
	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)

	labels := map[string]string{"app": nginxName}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nginxName,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   nginxName,
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  nginxContainerName,
							Image: nginxImageAddress,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: authenticatorPort,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      configMapName,
									MountPath: ConfigMountPath,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: configMapName,
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}

//Deploymentfunc (r *BasicAuthenticatorReconciler) CreateBasicAuthenticatorConfigmap(basicAuthenticator *v1alpha1.BasicAuthenticator) (*appsv1.Deployment, error) {
//	deploy := appsv1.Deployment{}
//
//}

func (r *BasicAuthenticatorReconciler) CreateBasicAuthenticatorDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string) *appsv1.Deployment {
}
