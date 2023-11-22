package basic_authenticator

import "github.com/snapp-incubator/simple-authenticator/api/v1alpha1"

func assignAnnotation(authenticator *v1alpha1.BasicAuthenticator, key, value string) {
	if authenticator.ObjectMeta.Annotations == nil {
		authenticator.ObjectMeta.Annotations = make(map[string]string)
	}
	authenticator.ObjectMeta.Annotations[key] = value
}
