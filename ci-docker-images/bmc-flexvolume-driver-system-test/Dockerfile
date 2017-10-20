from ubuntu:16.04

ENV http_proxy=http://www-proxy.us.oracle.com:80
ENV HTTP_PROXY=http://www-proxy.us.oracle.com:80
ENV https_proxy http://www-proxy.us.oracle.com:80
ENV HTTPS_PROXY http://www-proxy.us.oracle.com:80

# Install the required dependencies.
RUN apt-get update && apt-get install -y \
  git \
  openssh-client \
  python \
  unzip \
  wget

# Install terraform.
ADD https://releases.hashicorp.com/terraform/0.10.7/terraform_0.10.7_linux_amd64.zip terraform_0.10.7_linux_amd64.zip
RUN unzip terraform_0.10.7_linux_amd64.zip && \
  mv terraform /usr/bin/

# Installs the OCI terraform provider.
ADD https://github.com/oracle/terraform-provider-oci/releases/download/v2.0.1/linux.tar.gz linux.tar.gz
RUN tar -xzf linux.tar.gz -C / && \
  echo 'providers { oci = "/linux_amd64/terraform-provider-oci_v2.0.1" }' > ~/.terraformrc
