package basic_authenticator

import (
	"github.com/snapp-incubator/simple-authenticator/internal/config"
)

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
