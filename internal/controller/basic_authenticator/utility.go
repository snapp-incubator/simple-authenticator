package basic_authenticator

import (
	"github.com/snapp-incubator/simple-authenticator/api/v1alpha1"
	"strings"
)

func assignAnnotation(authenticator *v1alpha1.BasicAuthenticator, key, value string) {
	if authenticator.ObjectMeta.Annotations == nil {
		authenticator.ObjectMeta.Annotations = make(map[string]string)
	}
	authenticator.ObjectMeta.Annotations[key] = value
}

func getServiceName(appService string) string {
	serviceParts := strings.Split(appService, ".")
	if len(serviceParts) == 3 && serviceParts[2] == "svc" {
		// checking for appservice.ns.svc format
		return serviceParts[0]
	} else if len(serviceParts) == 1 {
		return serviceParts[0]
	} else {
		return ""
	}
}
