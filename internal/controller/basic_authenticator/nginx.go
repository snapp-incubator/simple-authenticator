package basic_authenticator

import (
	"context"
	"fmt"
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	"github.com/snapp-incubator/simple-authenticator/internal/config"
	"github.com/snapp-incubator/simple-authenticator/pkg/random_generator"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	ConfigMountPath = "/etc/nginx/conf.d"
	SecretMountDir  = "/etc/secret"
	SecretMountPath = "/etc/secret/.htpasswd"
	//TODO: maybe using better templating?
	template = `server {
	listen AUTHENTICATOR_PORT;
	location / {
		resolver    8.8.8.8;
		auth_basic	"basic authentication area";
		auth_basic_user_file "FILE_PATH";
		proxy_pass http://APP_SERVICE:APP_PORT;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
}`
)

// TODO: come up with better name that "nginx"
func createNginxDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string, credentialName string, customConfig *config.CustomConfig) *appsv1.Deployment {
	nginxImageAddress := nginxDefaultImageAddress
	if customConfig != nil && customConfig.WebserverConf.Image != "" {
		nginxImageAddress = customConfig.WebserverConf.Image
	}

	nginxContainerName := nginxDefaultContainerName
	if customConfig != nil && customConfig.WebserverConf.ContainerName != "" {
		nginxContainerName = customConfig.WebserverConf.ContainerName
	}

	deploymentName := random_generator.GenerateRandomName(basicAuthenticator.Name, "deployment")
	replicas := int32(basicAuthenticator.Spec.Replicas)
	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)

	basicAuthLabels := map[string]string{"app": deploymentName}

	//TODO: mount secret as volume
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    basicAuthLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: basicAuthLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deploymentName,
					Labels: basicAuthLabels,
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
									Name:      credentialName,
									MountPath: SecretMountDir,
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

func createNginxConfigmap(basicAuthenticator *v1alpha1.BasicAuthenticator) *corev1.ConfigMap {
	configmapName := random_generator.GenerateRandomName(basicAuthenticator.Name, "configmap")
	basicAuthLabels := map[string]string{
		"app": basicAuthenticator.Name,
	}
	nginxConf := fillTemplate(template, SecretMountPath, basicAuthenticator)
	data := map[string]string{
		"nginx.conf": nginxConf,
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    basicAuthLabels,
		},
		Data: data,
	}
	return configMap
}

func createCredentials(basicAuthenticator *v1alpha1.BasicAuthenticator) *corev1.Secret {
	username, password := random_generator.GenerateRandomString(20), random_generator.GenerateRandomString(20)
	htpasswdString := fmt.Sprintf("%s:%s", username, password)
	secretName := random_generator.GenerateRandomName(basicAuthenticator.Name, random_generator.GenerateRandomString(10))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: basicAuthenticator.Namespace,
		},
		StringData: map[string]string{
			".htpasswd": htpasswdString,
		},
	}
	return secret
}

func injector(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string, credentialName string, customConfig *config.CustomConfig, k8Client client.Client) ([]*appsv1.Deployment, error) {
	nginxImageAddress := getNginxContainerImage(customConfig)
	nginxContainerName := getNginxContainerName(customConfig)

	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)
	var deploymentList appsv1.DeploymentList
	if err := k8Client.List(
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
					MountPath: SecretMountDir,
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

func fillTemplate(template string, secretPath string, authenticator *v1alpha1.BasicAuthenticator) string {
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
