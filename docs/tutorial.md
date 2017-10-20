# Tutorial

This guide will walk you through creating a pod with persistent storage. It assumes
that you have already installed the flexvolume driver in your cluster.

See examples/nginx.yaml for a finished Kubernetes manifest that ties all these concepts
together.

1. Create a block storage volume. This can be done using the `oci` CLI as follows:

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

### Fixing a pod to a node

It's important to note that a block volume can only be attached to a node that runs
in the same AD. To get around this problem, you can use a nodeSelector to ensure
that a pod is scheduled on a particular node.

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
    - name: abuwcljreo5hftcm75uemj3wyirmwtvxe62qan5fm52bx5n2tn22abxr3lbq
      mountPath: /usr/share/nginx/html
  nodeSelector:
    node.info/availability.domain: 'UpwH-US-ASHBURN-AD-1'
  volumes:
  - name: abuwcljreo5hftcm75uemj3wyirmwtvxe62qan5fm52bx5n2tn22abxr3lbq
    flexVolume:
      driver: "oracle/oci"
      fsType: "ext4"
```

## Debugging

The flexvolume driver writes logs to /usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/oci_flexvolume_driver.log by default.

## Assumptions

- If a Flexvolume is specified for a pod, it will only work with a single
  replica. (or if there is more than one replica for a pod, they will all have
  to run on the same Kubernetes node). This is because a volume can only be
  attached to one instance at any one time. Note: This is in common with both
  the Amazon and Google persistent volume implementations, which also have the
  same constraint.

- If Nodes in the cluster span availability domain you must make sure your pods are scheduled
  in the correct availability domain. This can be achieved using the label selectors with the zone/region.

  Using the [oci-volume-provisioner](https://github.com/oracle/oci-volume-provisioner)
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

This issue is not set to be resolved until the release of Kubernetes 1.8.
