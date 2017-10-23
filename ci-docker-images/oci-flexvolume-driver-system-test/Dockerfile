FROM ubuntu:16.04

ARG TERRAFORM_VERSION=0.10.7
ARG OCI_TERRAFORM_PROVIDER_VERSION="2.0.2"

# Installs the required dependencies.
RUN http_proxy=$http_proxy apt-get update && apt-get install -y \
    git \
    libssl-dev \
    openssh-client \
    python \
    unzip \
    wget

# Installs terraform.
RUN https_proxy=$https_proxy wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN mv terraform /usr/bin/

# Installs the OCI terraform provider.
RUN https_proxy=$https_proxy wget https://github.com/oracle/terraform-provider-oci/releases/download/${OCI_TERRAFORM_PROVIDER_VERSION}/linux.tar.gz
RUN tar -xzvf linux.tar.gz -C /
RUN echo "providers { oci = \"/linux_amd64/terraform-provider-oci_v${OCI_TERRAFORM_PROVIDER_VERSION}\" }" > ~/.terraformrc
