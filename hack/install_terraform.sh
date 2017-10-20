#!/bin/bash
#
# Installs terraform and OCI driver for running integration tests on linux machines

set -o errexit
set -o nounset
set -o pipefail

TF_VERSION=0.10.7
TF_OCI_VERSION=2.0.2

apt-get update \
&& apt-get install -y wget unzip git \
&& rm -rf /var/lib/apt/lists/*

wget -q https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip?_ga=2.166871102.1829630053.1508488704-1222163521.1508488704

unzip terraform* \
&& mv terraform /usr/local/bin \
&& rm terraform*

wget -q https://github.com/oracle/terraform-provider-oci/releases/download/${TF_OCI_VERSION}/linux.tar.gz

tar -zxvf linux.tar.gz \
  && mv linux_amd64/terraform-provider-oci_v${TF_OCI_VERSION} /usr/local/bin/terraform-provider-oci \
  && rm linux.tar.gz

#!/bin/bash
cat << 'EOF' >> /root/.terraformrc
providers {
  oci = "/usr/local/bin/terraform-provider-oci"
}
EOF
