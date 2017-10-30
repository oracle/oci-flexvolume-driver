# Ansible installer for oci-flexvolume-driver

This Ansible playbook will install the oci-flexvolume-driver
on an existing Kubernetes cluster built with the open source
[terraform kubernetes installer](https://github.com/oracle/terraform-kubernetes-installer).

## Getting started

#### Create an inventory file

Create an inventory file that contains the kube master and kube workers in
your cluster (the oci-flexvolume-driver needs to be installed on all masters and workers)

```
[all:vars]
ansible_ssh_user=ubuntu

[masters]
master ansible_ssh_host=...

[workers]
worker1 ansible_ssh_host=...
```

#### Run the playbook.

*If you're using Ubuntu 16.04 you might need to use python3 as in the example below.*

```
ansible-playbook -i hosts \
    --private-key=generated/instances_id_rsa \
    -e 'ansible_python_interpreter=/usr/bin/python3'
```
