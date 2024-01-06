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

## Contributing
Contributions are warmly welcomed. Feel free to submit issues or pull requests.

## License
This project is licensed under the [Apache License 2.0](https://github.com/snapp-incubator/s3-operator/blob/main/LICENSE).