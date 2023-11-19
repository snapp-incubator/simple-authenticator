package controller

import (
	"context"
	"fmt"
	"github.com/sinamna/BasicAthenticator/api/v1alpha1"
	"github.com/sinamna/BasicAthenticator/pkg/htpasswd"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	ConfigMountPath = "/etc/nginx/conf.d"
	SecretMountPath = "/etc/secret"
	//TODO: maybe using better templating?
	template = `server {
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
	nginxImageAddress := "curlimages/curl:latest"
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
			Name:      deploymentName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    labels,
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
							Name:    nginxContainerName,
							Image:   nginxImageAddress,
							Command: []string{"sleep", "infinity"}, //temp
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
									Name:      credentialName,
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
							Name: credentialName,
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
			Name:      configmapName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    labels,
		},
		Data: data,
	}
	return configMap
}

func (r *BasicAuthenticatorReconciler) CreateCredentials(authenticator *v1alpha1.BasicAuthenticator) *corev1.Secret {
	username, password := htpasswd.GenerateRandomString(20), htpasswd.GenerateRandomString(20)
	htpasswdString := fmt.Sprintf("%s:%s", username, password)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-credentials", authenticator.Name),
			Namespace: authenticator.Namespace,
		},
		StringData: map[string]string{
			".htpasswd": htpasswdString,
		},
	}
	return secret
}

func (r *BasicAuthenticatorReconciler) Injector(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string, credentialName string) ([]*appsv1.Deployment, error) {
	nginxImageAddress := "curlimages/curl:latest"
	if r.CustomConfig != nil && r.CustomConfig.WebserverConf.Image != "" {
		nginxImageAddress = r.CustomConfig.WebserverConf.Image
	}

	nginxContainerName := "nginx"
	if r.CustomConfig != nil && r.CustomConfig.WebserverConf.ContainerName != "" {
		nginxContainerName = r.CustomConfig.WebserverConf.ContainerName
	}
	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)
	var deploymentList appsv1.DeploymentList
	if err := r.Client.List(
		ctx,
		&deploymentList,
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(basicAuthenticator.Spec.Selector.MatchLabels)},
		client.InNamespace(basicAuthenticator.Namespace)); err != nil {
		return nil, err
	}
	resultDeployments := make([]*appsv1.Deployment, 0)

	for _, deployment := range deploymentList.Items {
		// we can use revision number to update container config
		_, isInjected := deployment.ObjectMeta.Annotations["basic.authenticator.inject/revision"]
		if isInjected {
			continue
		}
		deployment.ObjectMeta.Annotations = map[string]string{
			"basic.authenticator.inject/revision": "1",
		}

		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, corev1.Container{
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
					Name:      credentialName,
					MountPath: SecretMountPath,
				},
			},
		})
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: configMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		})
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: credentialName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: credentialName,
				},
			},
		})
		resultDeployments = append(resultDeployments, &deployment)
	}
	return resultDeployments, nil
}

func FillTemplate(template string, secretPath string, authenticator *v1alpha1.BasicAuthenticator) string {
	var result string
	var appservice string
	if authenticator.Spec.Type == "sidecar" {
		appservice = "localhost"
	} else {
		appservice = authenticator.Spec.AppService
	}
	result = strings.Replace(template, "AUTHENTICATOR_PORT", fmt.Sprintf("%d", authenticator.Spec.AuthenticatorPort), 1)
	result = strings.Replace(result, "FILE_PATH", secretPath, 1)
	result = strings.Replace(result, "APP_SERVICE", appservice, 1)
	result = strings.Replace(result, "APP_PORT", fmt.Sprintf("%d", authenticator.Spec.AppPort), 1)
	return result
}
