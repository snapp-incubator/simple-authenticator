package basic_authenticator

import (
	"context"
	defaultError "errors"
	"fmt"
	"github.com/pkg/errors"
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	"github.com/snapp-incubator/simple-authenticator/internal/config"
	"github.com/snapp-incubator/simple-authenticator/pkg/htpasswd"
	"github.com/snapp-incubator/simple-authenticator/pkg/random_generator"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// TODO: come up with better name that "nginx"
func createNginxDeployment(basicAuthenticator *v1alpha1.BasicAuthenticator, configMapName string, credentialName string, customConfig *config.CustomConfig) *appsv1.Deployment {
	nginxImageAddress := getNginxContainerImage(customConfig)
	nginxContainerName := getNginxContainerName(customConfig)

	deploymentName := random_generator.GenerateRandomName(basicAuthenticator.Name, "deployment")
	replicas := int32(basicAuthenticator.Spec.Replicas)
	authenticatorPort := int32(basicAuthenticator.Spec.AuthenticatorPort)

	basicAuthLabels := map[string]string{"app": deploymentName, basicAuthenticatorNameLabel: basicAuthenticator.Name}

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
									Items: []corev1.KeyToPath{
										{
											Key:  SecretHtpasswdField,
											Path: SecretHtpasswdField,
										},
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

func createNginxConfigmap(basicAuthenticator *v1alpha1.BasicAuthenticator) *corev1.ConfigMap {
	configmapName := random_generator.GenerateRandomName(basicAuthenticator.Name, "configmap")
	basicAuthLabels := map[string]string{
		basicAuthenticatorNameLabel: basicAuthenticator.Name,
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

func updateHtpasswdField(secret *corev1.Secret) error {
	username, ok := secret.Data["username"]
	if !ok {
		return defaultError.New("username not found in secret")
	}
	password, ok := secret.Data["password"]
	if !ok {
		return defaultError.New("password not found in secret")
	}
	salt, err := random_generator.GenerateRandomString(8)
	if err != nil {
		return errors.Wrap(err, "failed to generate salt")
	}
	hashedPassword, err := htpasswd.ApacheHash(string(password), salt)
	if err != nil {
		return err
	}
	htpasswdString := fmt.Sprintf("%s:%s", string(username), hashedPassword)
	secret.Data["htpasswd"] = []byte(htpasswdString)
	return nil
}
func createCredentials(basicAuthenticator *v1alpha1.BasicAuthenticator) (*corev1.Secret, error) {
	username, err := random_generator.GenerateRandomString(20)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate username")
	}
	password, err := random_generator.GenerateRandomString(20)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate password")
	}
	salt, err := random_generator.GenerateRandomString(10)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate salt")
	}
	basicAuthLabels := map[string]string{
		basicAuthenticatorNameLabel: basicAuthenticator.Name,
	}
	secretName := random_generator.GenerateRandomName(basicAuthenticator.Name, salt)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    basicAuthLabels,
		},
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
	}
	return secret, nil
}
func createNginxService(ctx context.Context, basicAuthenticator *v1alpha1.BasicAuthenticator, selector *metav1.LabelSelector) *corev1.Service {
	serviceName := fmt.Sprintf("%s-svc", basicAuthenticator.Name)
	serviceType := getServiceType(basicAuthenticator.Spec.ServiceType)
	targetPort := intstr.IntOrString{Type: intstr.Int, IntVal: int32(basicAuthenticator.Spec.AuthenticatorPort)}
	basicAuthLabel := map[string]string{
		basicAuthenticatorNameLabel: basicAuthenticator.Name,
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: basicAuthenticator.Namespace,
			Labels:    basicAuthLabel,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector.MatchLabels,
			Type:     serviceType,
			Ports: []corev1.ServicePort{
				{
					Port:       int32(basicAuthenticator.Spec.AuthenticatorPort),
					TargetPort: targetPort,
					Name:       "authenticator",
				},
			},
		},
	}
	return &svc
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
		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}
		deployment.Labels[basicAuthenticatorNameLabel] = basicAuthenticator.Name
		idx := getContainerIndex(deployment.Spec.Template.Spec.Containers, nginxContainerName)
		if idx == -1 { // meaning its the first time creating container
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
		} //TODO: handling config change later (idx >=0)

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

func getServiceType(serviceType string) corev1.ServiceType {
	switch serviceType {
	case "NodePort":
		return corev1.ServiceTypeNodePort
	case "LoadBalancer":
		return corev1.ServiceTypeLoadBalancer
	default:
		return corev1.ServiceTypeClusterIP
	}
}

func getContainerIndex(containers []corev1.Container, name string) int {
	for idx, container := range containers {
		if container.Name == name {
			return idx
		}
	}
	return -1
}
