# Install

This guide will show you how to install the flexvolume driver on an existing Kubernetes cluster.

## Create Docker secret

```bash
kubectl create secret docker-registry oci-docker-secret \
    --docker-server=wcr.io \
    --docker-username=$DOCKER_REGISTRY_USERNAME \
    --docker-password=$DOCKER_REGISTRY_PASSWORD \
    --docker-email=k8s@oracle.com
```

Create auth configuration for driver as a K8s secret

```bash
kubectl create secret generic flexvolume-secret --from-file=manifests/config.yaml
```

## Deploy the driver on all worker nodes

```
kubectl create -f manifests/ds.yaml
```

