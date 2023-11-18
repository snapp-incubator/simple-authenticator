package controller

import (
	"fmt"
	"github.com/sinamna/BasicAthenticator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	ConfigMountPath = "/etc/nginx/conf.d"
	template        = `
	server {
		listen AUTHENTICATOR_PORT;
		location / {
			auth_basic	"basic authentication area";
			auth_basic_user_file FILE_PATH;
			proxy_pass http://APP_SERVICE:APP_PORT
		}
	}
`
)

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

	deploymentName := fmt.Sprintf("%s-nginx", basicAuthenticator.Name)
	replicas := int32(basicAuthenticator.Spec.Replicas)
	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)

	labels := map[string]string{"app": deploymentName}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deploymentName,
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

func (r *BasicAuthenticatorReconciler) CreateNginxConfigmap(basicAuthenticator *v1alpha1.BasicAuthenticator, secretRef string) *v1.ConfigMap {
	appPort := int32(basicAuthenticator.Spec.AppPort)
	configmapName := fmt.Sprintf("%s-nginx-conf", basicAuthenticator.Name)
	labels := map[string]string{
		"app": basicAuthenticator.Name,
	}
	nginxConf := FillTemplate(template, secretRef, basicAuthenticator)
	data := map[string]string{
		"nginx.conf": nginxConf,
	}
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configmapName,
			Labels: labels,
		},
		Data: data,
	}
	return configMap
}

func FillTemplate(template string, secretPath string, authenticator *v1alpha1.BasicAuthenticator) string {
	var result string
	result = strings.Replace(template, "AUTHENTICATOR_PORT", fmt.Sprintf("%d", authenticator.Spec.AuthenticatorPort), 1)
	result = strings.Replace(result, "FILE_PATH", secretPath, 1)
	result = strings.Replace(result, "APP_SERVICE", authenticator.Spec.AppService, 1)
	result = strings.Replace(result, "APP_PORT", fmt.Sprintf("%d", authenticator.Spec.AppPort), 1)
	return result
}
