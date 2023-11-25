package basic_authenticator

const (
	nginxDefaultImageAddress    = "nginx:1.25.3"
	nginxDefaultContainerName   = "nginx"
	SecretAnnotation            = "authenticator.snappcloud.io/secret.name"
	ConfigmapAnnotation         = "authenticator.snappcloud.io/configmap.name"
	basicAuthenticatorFinalizer = "basicauthenticators.authenticator.snappcloud.io/finalizer"
)
