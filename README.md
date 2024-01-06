# Simple Authenticator

## Introduction

The Simple Authenticator is a Kubernetes-based tool designed to streamline and enhance basic authentication within a cluster. It serves as a crucial component for managing microservices' security and performance efficiently.

### Purpose

In Kubernetes environments, effective authentication is essential, especially when dealing with internal traffic between microservices.
### Features

- **NGINX Deployment:** Supports both sidecar and standalone deployment of NGINX for secure authentication.
- **Adaptive Scale Support:** Dynamically scales based on the number of pods in the targeted service, optimizing resource utilization.
- **Plain Username and Password Authentication:** Simplifies credential management by transforming secrets to NGINX preferences automatically.

The Simple Authenticator ensures that authentication between microservices is both secure and efficient, contributing to a robust and well-architected Kubernetes environment.

# Installation of Simple Authenticator

## Using Makefile
Deploy Simple Authenticator using the Makefile:

   ```sh
   make deploy
   ```

## Using Helm
Deploy Simple Authenticator using Helm:

   ```sh
   helm upgrade --install simple-authenticator oci://ghcr.io/snapp-incubator/simple-authenticator/helm-charts/simple-authenticator --version v0.1.8
   ```

## Using OLM (Operator Lifecycle Manager)
All the operator releases are bundled and pushed to the [Snappcloud hub](https://github.com/snapp-incubator/snappcloud-hub) which is a hub for the catalog sources. Install using Operator Lifecycle Manager (OLM) by following these steps:
1. Install [snappcloud hub catalog-source](https://github.com/snapp-incubator/snappcloud-hub/blob/main/catalog-source.yml)
2. Apply the subscription manifest as shown below:
```yaml
    apiVersion: operators.coreos.com/v1alpha1
    kind: Subscription
    metadata:
      name: simple-authenticator
      namespace: operators
    spec:
      channel: stable-v1
      installPlanApproval: Automatic
      name: simple-authenticator
      source: snappcloud-hub-catalog
      sourceNamespace: openshift-marketplace
      config:
        resources:
          limits:
            cpu: 2
            memory: 2Gi
          requests:
            cpu: 1
            memory: 1Gi
```
## Development

### Run locally
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

### Building testing image

```shell
make docker-build IMG=<desired-image-tag>
```

### Building the helm chart

We use [helmify](https://github.com/arttor/helmify) to generate Helm chart from kustomize rendered manifests. To update
the chart run:

```shell
make helm
```

## Usage


The Simple Authenticator operates through the `BasicAuthenticator` custom resource, providing a straightforward way to manage authentication credentials.

### Creating a Basic Authenticator

To create a `BasicAuthenticator`, apply a manifest with your customizations:

```yaml
apiVersion: authenticator.snappcloud.io/v1alpha1
kind: BasicAuthenticator
metadata:
  name: example-basicauthenticator
  namespace: simple-authenticator-test
spec:
  type: "sidecar" 
  replicas: 2 
  selector: 
    matchLabels:
      app: my-app
  serviceType: "ClusterIP" 
  appPort: 8080
  appService: "my-app-service"
  adaptiveScale: false 
  authenticatorPort: 80 
  credentialsSecretRef: "my-credentials-secret"
```

### Authentication Fields

- `type`: Sidecar or standalone deployment.
- `replicas`: Number of replicas (optional, used in deployment mode).
- `selector`: Selector for targeting specific labels (optional, used in sidecar mode).
- `serviceType`: Service type (optional).
- `appPort`: Port where the application is running (required).
- `appService`: Name of the application service (optional).
- `adaptiveScale`: Enable or disable adaptive scaling (optional, used in deployment mode).
- `authenticatorPort`: Port for the authenticator (required).
- `credentialsSecretRef`: Reference to the credentials secret (optional).

### Authenticator Modes

The Simple Authenticator offers two distinct operational modes to cater to different architectural needs in a Kubernetes environment: Deployment Mode and Sidecar Mode.

#### Deployment Mode Configuration

- __Application Service and Port__: Target application's service and port.
- __Authenticator Port__: Port for NGINX deployment to listen to.
- __Adaptive Scaling__: Automatic scaling based on number of pods of targeted service.
- __Replicas__: Number of NGINX deployment replicas.

#### Sidecar Mode Configuration

- __Application Port__: Application's port within the pod.
- __Authenticator Port__: Port for NGINX sidecar to listen to.
- __Selector__: Targets specific pod(s) for adding the NGINX sidecar.

#### Trade-offs Between Deployment and Sidecar Modes

Deployment Mode is preferable for scenarios requiring clear separation between the authentication layer and application, and is more scalable for environments with many pods. Sidecar Mode, on the other hand, is suited for scenarios where simplicity, reduced latency, and tight integration between the application and the authentication layer are priorities, albeit at the cost of increased resource consumption per pod.

### Credential Format

Secrets specified in `credentialsSecretRef` must contain `username` and `password` fields. If not correctly formatted, the secret will be rejected. Secrets must reside in `BasicAuthenticator`'s namespace.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-credentials-secret
  namespace: simple-authenticator-test
type: Opaque
stringData:
  username: <username>
  password: <password>
```

### Automatic Credential Generation

If no `credentialsSecretRef` is set, a secret with a random username and password will be automatically generated.


## Contributing
Contributions are warmly welcomed. Feel free to submit issues or pull requests.

## License
This project is licensed under the [Apache License 2.0](https://github.com/snapp-incubator/s3-operator/blob/main/LICENSE).