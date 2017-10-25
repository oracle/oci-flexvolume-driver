# Copyright 2017 The OCI Flexvolume Driver Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
