# OCI Flexvolume Driver

[![wercker status](https://app.wercker.com/status/9fdba1db3d8e347756aa19819dddc4d5/s/master "wercker status")](https://app.wercker.com/project/byKey/9fdba1db3d8e347756aa19819dddc4d5)

This project implements a flexvolume driver for Kubernetes clusters
running on Oracle Cloud Infrastructure (OCI). It enables mounting of [OCI block
storage volumes][1] to Kubernetes Pods via the [Flexvolume][2] plugin interface.

We recommend you use this driver in conjunction with the OCI Volume Provisioner.
See the [oci-volume-provisioner][3] for more information.

## Install / Setup

We publish the OCI flexvolume driver as a single binary that needs to be
installed on every node in your Kubernetes cluster.

### Ansible

The recommended way to install the driver is with Ansible.

Compile the oci binary or download the latest release from Github.

```
make build
```

Now you can use Ansible to deploy the driver to your cluster

```
cd ansible
```

Create an inventory file

```
cp hosts.example hosts
```

Add the details for all the masters and nodes in your cluster
(the oci-flexvolume-driver needs to be installed on all masters and workers)

```
[all:vars]
ansible_ssh_user=opc

[masters]
...

[workers]
...
```

#### Run the playbook.

```
ansible-playbook -i hosts \
--private-key=generated/instances_id_rsa site.yaml
```

### Manually

The driver should be installed in the volume plugin path on **every**
node in your Kubernetes cluster at the following location:
`/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/oci`.

NOTE: If running kube-controller-managers in a container you _must_ ensure that
the plugin directory is mounted into the container. See:
https://gitlab-odx.oracle.com/odx/oke-prime/merge_requests/15 for specifics.

#### Configuration

The driver requires API credentials for a OCI account with the ability
to attach and detach [OCI block storage volumes][1] from to/from the appropriate
nodes in the cluster.

These credentials should be provided via a YAML file present on **master** nodes
in the cluster at `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/config.yaml`
in the following format:

```yaml
---
auth:
  tenancy: ocid1.tenancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  compartment: ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  user: ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  region: us-phoenix-1
  key: |
    -----BEGIN RSA PRIVATE KEY-----
    <snip>
    -----END RSA PRIVATE KEY-----
  fingerprint: d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e
```

If `"region"` and/or `"compartment"` are not specified in the config file
they will be retrieved from the hosts [OCI metadata service][4].

#### Extra configuration values

You can set these in the environment to override the default values.

* `OCI_FLEXD_DRIVER_LOG_DIR` - Directory where the log file is written (Default: `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci`)
* `OCI_FLEXD_DRIVER_DIRECTORY` - Directory where the driver binary lives (Default:
`/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci`)

# Tutorial

This guide will walk you through creating a Pod with persistent storage. It assumes
that you have already installed the flexvolume driver in your cluster.

See [examples/nginx.yaml](examples/nginx.yaml) for a finished Kubernetes manifest that ties all these concepts
together.

1. Create a block storage volume. This can be done using the `oci` [CLI][5] as follows:

```bash
$ oci bv volume create \
    --availability-domain="aaaa:PHX-AD-1" \
    --compartment-id "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
```

2. Add a volume to your `pod.yml` in the format below and named with the last
   section of your volume's OCID (see limitations). E.g. a volume with the OCID

```
ocid1.volume.oc1.phx.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

Would be named `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`
in the pod.yml as shown below.

```yaml
volumes:
  - name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    flexVolume:
      driver: "oracle/oci"
      fsType: "ext4"
```

3. Add volume mount(s) in the appropriate container(s) in your as follows:

```yaml
volumeMounts:
  - name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    mountPath: /usr/share/nginx/html
```
(Where `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"` is the
last '.' sparated section of the volume OCID.)

### Fixing a Pod to a Node

It's important to note that a block volume can only be attached to a Node that runs
in the same AD. To get around this problem, you can use a nodeSelector to ensure
that a Pod is scheduled on a particular Node.

This following example shows you how to do this.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx
    ports:
    - containerPort: 80
    volumeMounts:
    - name: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      mountPath: /usr/share/nginx/html
  nodeSelector:
    node.info/availability.domain: 'UpwH-US-ASHBURN-AD-1'
  volumes:
  - name: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    flexVolume:
      driver: "oracle/oci"
      fsType: "ext4"
```

## Debugging

The flexvolume driver writes logs to `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/oci_flexvolume_driver.log` by default.

## Assumptions

- If a Flexvolume is specified for a Pod, it will only work with a single
  replica. (or if there is more than one replica for a Pod, they will all have
  to run on the same Kubernetes Node). This is because a volume can only be
  attached to one instance at any one time. Note: This is in common with both
  the Amazon and Google persistent volume implementations, which also have the
  same constraint.

- If nodes in the cluster span availability domain you must make sure your Pods are scheduled
  in the correct availability domain. This can be achieved using the label selectors with the zone/region.

  Using the [oci-volume-provisioner][6]
  makes this much easier.

- For all nodes in the cluster, the instance display name in the OCI API must
  match with the instance hostname, start with the vnic hostnamelabel or match the public IP.
  This relies on the requirement that the nodename must be resolvable.

## Limitations

Due to [kubernetes/kubernetes#44737][7] ("Flex volumes which implement
`getvolumename` API are getting unmounted during run time") we cannot implement
`getvolumename`. From the issue:

> Detach call uses volume name, so the plugin detach has to work with PV Name

This means that the PV (Persistent Volume) name in the `pod.yml` _must_ be
the last part of the block volume OCID ('.' separated). Otherwise, we would
have no way of determining which volume to detach from which worker node. Even
if we were to store state at the time of volume attachment PV names would have
to be unique across the cluster which is an unreasonable constraint.

The full OCID cannot be used because the PV name must be shorter than 63
characters and cannot contain '.'s. To reconstruct the OCID we use the region
of the master on which `Detach()` is exected so this blocks support for cross
region clusters.

## Support

Please checkout our documentation. If you find a bug, please raise an
[issue](https://github.com/oracle/oci-flexvolume-driver/issues/new)

[1]: https://docs.us-phoenix-1.oraclecloud.com/Content/Block/Concepts/overview.htm
[2]: https://github.com/kubernetes/community/blob/master/contributors/devel/flexvolume.md
[3]: https://github.com/oracle/oci-volume-provisoner
[4]: https://docs.us-phoenix-1.oraclecloud.com/Content/Compute/Tasks/gettingmetadata.htm
[5]: https://docs.us-phoenix-1.oraclecloud.com/Content/API/SDKDocs/cli.htm
[6]: https://github.com/oracle/oci-volume-provisioner
