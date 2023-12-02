package basic_authenticator

import (
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	"github.com/snapp-incubator/simple-authenticator/internal/config"
)

func assignAnnotation(authenticator *v1alpha1.BasicAuthenticator, key, value string) {
	if authenticator.ObjectMeta.Annotations == nil {
		authenticator.ObjectMeta.Annotations = make(map[string]string)
	}
	authenticator.ObjectMeta.Annotations[key] = value
}

func getNginxContainerImage(customConfig *config.CustomConfig) string {

	if customConfig != nil && customConfig.WebserverConf.Image != "" {
		return customConfig.WebserverConf.Image
	}
	return nginxDefaultImageAddress
}
func getNginxContainerName(customConfig *config.CustomConfig) string {
	if customConfig != nil && customConfig.WebserverConf.ContainerName != "" {
		return customConfig.WebserverConf.ContainerName
	}
	return nginxDefaultContainerName
}
