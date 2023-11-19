package controller

import (
	"fmt"
	"github.com/sinamna/BasicAthenticator/api/v1alpha1"
	"github.com/sinamna/BasicAthenticator/pkg/hash"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	ConfigMountPath = "/etc/nginx/conf.d"
	SecretMountPath = "/etc/secret/credentials"
	SecretName      = "credentials"
	//TODO: maybe using better templating?
	template = ` 
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
func (r *BasicAuthenticatorReconciler) CreateNginxDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string, credentialName string) *appsv1.Deployment {
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

	//TODO: mount secret as volume
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deploymentName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  nginxContainerName,
							Image: nginxImageAddress,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: authenticatorPort,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configMapName,
									MountPath: ConfigMountPath,
								},
								{
									Name:      SecretName,
									MountPath: SecretMountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: configMapName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
						{
							Name: SecretName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: credentialName,
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

func (r *BasicAuthenticatorReconciler) CreateNginxConfigmap(basicAuthenticator *v1alpha1.BasicAuthenticator) *corev1.ConfigMap {
	configmapName := fmt.Sprintf("%s-nginx-conf", basicAuthenticator.Name)
	labels := map[string]string{
		"app": basicAuthenticator.Name,
	}
	nginxConf := FillTemplate(template, SecretMountPath, basicAuthenticator)
	data := map[string]string{
		"nginx.conf": nginxConf,
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configmapName,
			Labels: labels,
		},
		Data: data,
	}
	return configMap
}

func (r *BasicAuthenticatorReconciler) CreateCredentials(authenticator *v1alpha1.BasicAuthenticator) *corev1.Secret {
	username, password := hash.GenerateRandomString(20), hash.GenerateRandomString(20)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-credentials", authenticator.Name),
		},
		StringData: map[string]string{
			"username": username,
			"password": password,
		},
	}
	return secret
}

func FillTemplate(template string, secretPath string, authenticator *v1alpha1.BasicAuthenticator) string {
	var result string
	result = strings.Replace(template, "AUTHENTICATOR_PORT", fmt.Sprintf("%d", authenticator.Spec.AuthenticatorPort), 1)
	result = strings.Replace(result, "FILE_PATH", secretPath, 1)
	result = strings.Replace(result, "APP_SERVICE", authenticator.Spec.AppService, 1)
	result = strings.Replace(result, "APP_PORT", fmt.Sprintf("%d", authenticator.Spec.AppPort), 1)
	return result
}
