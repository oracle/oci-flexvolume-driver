#!/bin/bash

# temos-proxy-on http://www-proxy-lon.uk.oracle.com:80
# temos-proxy-on http://www-proxy-adcq7-new.us.oracle.com:80

# the docker registry credentials to pull the image down from
#
# docker login -u templecloud -p 8dbacdeeaa64ed4fba2d958123c314e45b8557f53be0bee92c411db2897ae3d2 wcr.io
# export DOCKER_REGISTRY_USERNAME="templecloud"
# export DOCKER_REGISTRY_PASSWORD="8dbacdeeaa64ed4fba2d958123c314e45b8557f53be0bee92c411db2897ae3d2"
# DIR_NAME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# compile for mac
export GOOS=darwin

# function init_system_test() {
#   PRIVATE_INSTANCE_KEY="${DIR_NAME}/test/integration/oci_instance_key.pem"
#   PUBLIC_INSTANCE_KEY="${DIR_NAME}/test/integration/oci_instance_key_public.pem"
#   rm -f "${PRIVATE_INSTANCE_KEY}"
#   rm -f "${PUBLIC_INSTANCE_KEY}"
#   openssl genrsa -out "${PRIVATE_INSTANCE_KEY}" 2048 > /dev/null 2>&1
#   chmod go-rwx "${PRIVATE_INSTANCE_KEY}"
#   openssl rsa -pubout -in "${PRIVATE_INSTANCE_KEY}" -out "${PUBLIC_INSTANCE_KEY}" > /dev/null 2>&1
# 
# 
#   export OCI_API_KEY="${HOME}/.oci/oci_api_key.pem"
#   export INSTANCE_KEY="${PRIVATE_INSTANCE_KEY}"
#   export INSTANCE_KEY_PUB="${PUBLIC_INSTANCE_KEY}"
# 
#   # export OCI_API_KEY_VAR=$(cat ~/.oci/oci_api_key.pem  | base64)
#   # export INSTANCE_KEY_VAR=$(cat "${PRIVATE_INSTANCE_KEY}" | base64)
#   # export INSTANCE_KEY_PUB_VAR=$(cat "${PUBLIC_INSTANCE_KEY}" | base64)
# }
# 
# # init_system_test
# 
# export KUBECONFIG='/Users/tlangfor/Work/dev/gitlab-odx/kubernetes/terraformk8sbmc/tlangfor-dev/tlangfor-dev-admin.conf'
# export CLUSTER_INSTANCE_SSH_KEY='/Users/tlangfor/Work/dev/gitlab-odx/kubernetes/terraformk8sbmc/tlangfor-dev/master'
# export OCICONFIG='/Users/tlangfor/.oraclebmc/oci-api-config.yaml'
# 
# export INSTANCE_KEY="/Users/tlangfor/.ssh/obmc-bristoldev/obmc-bristoldev"
# export MASTER_IP=129.146.87.216
# export SLAVE0_IP=129.146.17.209
# export SLAVE1_IP=129.146.28.246
# export WERCKER_API_TOKEN="f845db7c4ece178c00373ee58eadc14a275f70b98558cc3484e1fed17e627670"
# 

export OCI_COMPARTMENT="ocid1.compartment.oc1..aaaaaaaaujjg4lf3v6uaqeml7xfk7stzvrxeweaeyolhh75exuoqxpqjb4qq"

# spinnaker tenancy registry
# 
export OCIRUSERNAME="spinnaker/everest-ocir-push"
export OCIRPASSWORD="_9i0+]i8}ZkBCkqmC+Vj"
export OCIREGISTRY="iad.ocir.io"

# spinnaker - ashburn 
# 
export ENV_NAME="oci-ccm-test-cluster"
export OCI_CONFIG="${HOME}/.oci/spinnaker/config-iad"
export OCI_REGION="us-ashburn-1"

function test-ashburn-oci() {
    echo "test unauthorised oci connection..."
    oci iam region list
    echo "test authorised oci connection..."
    oci iam user list \
      --compartment-id "${OCI_COMPARTMENT}" \
      --config-file "${OCI_CONFIG}"
}

# export CLOUDCONFIG="${DIR}/cloud-provider.yaml"
# export CCM_SECLIST_ID="ocid1.securitylist.oc1.iad.aaaaaaaaqshdqfwpgqnvt42vrvtn4oqlpxvmgte5r5j7aczkxghodftx77gq"
# export K8S_SECLIST_ID="ocid1.securitylist.oc1.iad.aaaaaaaazsac74oe2fml7bhpmkbboik7zfzsma2eakeummgeyvbuzpjbvs4a"
export CLOUDCONFIG="${DIR}/cloud-provider-ashburn.yaml"
export CCM_SECLIST_ID="ocid1.securitylist.oc1.iad.aaaaaaaagekgnvc75yxb66xj2qgocejhyk5xuhojm6wimmteqa5fj43kk5lq"
export K8S_SECLIST_ID="ocid1.securitylist.oc1.iad.aaaaaaaajzgrgfma2xzzdv4erczaiphuyaflazbt2bvzw2cwj2pxo3fygo5a"

# # spinnaker - london
# # 
# export ENV_NAME="oci-ccm-test-cluster-london"
# export OCI_REGION="us-london-1"
# export OCI_CONFIG="${HOME}/.oci/spinnaker/config-lon"

# function test-london-oci() {
#     echo "test unauthorised oci connection..."
#     oci iam region list
#     echo "test authorised oci connection..."
#     oci iam user list \
#       --compartment-id "${OCI_COMPARTMENT}" \
#       --config-file "${OCI_CONFIG}"
# }

# spinaker - all 
# 
export CLOUD_PROVIDER_CFG=${CLOUDCONFIG}
export CLUSTER_ENV="/Users/tlangfor/Work/dev/orahub/kubernetes-test-terraform/${ENV_NAME}"
export KUBECONFIG="${CLUSTER_ENV}/generated/kubeconfig"
export OCI_TENANCY="ocid1.tenancy.oc1..aaaaaaaaxf3fuazosc6xng7l75rj6uist5jb6ken64t3qltimxnkymddqbma"
export OCI_COMPARTMENT="ocid1.compartment.oc1..aaaaaaaaujjg4lf3v6uaqeml7xfk7stzvrxeweaeyolhh75exuoqxpqjb4qq"
export OCI_USER="ocid1.user.oc1..aaaaaaaardi2xuizwtokwpmd2bfpafzevmg3hcwq6w6v7bhas4idkmi4pazq"
export FINGERPRINT="21:66:36:c7:a0:13:29:b8:29:3d:5f:33:f7:1c:fb:ca"
export PRIVATE_KEY=$(cat "${HOME}/.ssh/oci-cloud-controller-manager/oci-cloud-controller-manager.pem")

