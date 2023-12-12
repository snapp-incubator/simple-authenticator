package basic_authenticator

const (
	nginxDefaultImageAddress    = "nginx:1.25.3"
	nginxDefaultContainerName   = "nginx"
	basicAuthenticatorNameLabel = "basicauthenticator.snappcloud.io/name"
	basicAuthenticatorFinalizer = "basicauthenticator.snappcloud.io/finalizer"
	ExternallyManaged           = "basicauthenticator.snappcloud.io/externally.managed"
	ConfigMountPath             = "/etc/nginx/conf.d"
	SecretMountDir              = "/etc/secret"
	SecretMountPath             = "/etc/secret/htpasswd"
	SecretHtpasswdField         = "htpasswd"
	//TODO: maybe using better templating?
	template = `server {
	listen AUTHENTICATOR_PORT;
	location / {
		auth_basic	"basic authentication area";
		auth_basic_user_file "FILE_PATH";
		proxy_pass http://APP_SERVICE:APP_PORT;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
}`
	StatusAvailable   = "Available"
	StatusReconciling = "Reconciling"
	StatusDeleting    = "Deleting"
)
