package basic_authenticator

const (
	nginxDefaultImageAddress    = "nginx:1.25.3"
	nginxDefaultContainerName   = "nginx"
	SecretAnnotation            = "authenticator.snappcloud.io/secret.name"
	ConfigmapAnnotation         = "authenticator.snappcloud.io/configmap.name"
	basicAuthenticatorFinalizer = "basicauthenticator.snappcloud.io/finalizer"
	ExternallyManaged           = "basicauthenticator.snappcloud.io/externally.managed"
)
