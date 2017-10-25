#!/usr/bin/env bash

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

# Runs the integration tests for the OCI flexvolume driver.
#
# Required ENV vars:
#  - $OCI_API_KEY:      base64 encoded OCI API signing key.
#  - $INSTANCE_KEY:     base64 encoded SSH private key corresponding to
#                       $INSTANCE_KEY_PUB.
#  - $INSTANCE_KEY_PUB: base64 encoded SSH public key (added to instance
#                       authorized keys).

set -o errexit
set -o nounset
set -o pipefail

function help_text() {
    echo "usage: ./run.sh [--no-destroy] [--help]"
    echo
    echo "Provisions and runs the integration tests for the OCI flexvolume driver."
    echo
    echo "optional arguments:"
    echo "  -h, --help            show this help message and exit"
    echo "  --no-destroy          leave the test infrastructure up on completion"
}

# Whether or not to leave the instance and volume in-place after executing the
# test.
NO_DESTROY=0

while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
    --no-destroy)
    echo "INFO: '--no-destroy' flag set."
    NO_DESTROY=1
    shift
    ;;
    -h|--help)
    help_text
    exit 0
    shift
    ;;
    *)
      help_text
      exit 0
    shift
    ;;
  esac
done

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/terraform"

pushd "${BASE_DIR}"

# Create temporary directory for storing keys.
mkdir -p _tmp/

RET_CODE=1
function _trap_1 {
  if [ "${NO_DESTROY}" -ne "1" ]; then
    rm -rf ${BASE_DIR}/_tmp  # keep keys around when NO_DESTROY is set
  fi
  popd
  exit ${RET_CODE}
}
trap _trap_1 EXIT

# Set keys from environment.
echo $OCI_API_KEY | openssl enc -base64 -d -A > _tmp/oci_api_key.pem
echo $INSTANCE_KEY | openssl enc -base64 -d -A > _tmp/instance_key
echo $INSTANCE_KEY_PUB | openssl enc -base64 -d -A > _tmp/instance_key.pub
chmod 600 _tmp/instance_key
chmod 600 _tmp/oci_api_key.pem

export TF_VAR_ssh_public_key="$(cat _tmp/instance_key.pub)"
export TF_VAR_ssh_private_key="$(cat _tmp/instance_key)"

export TF_VAR_flexvolume_test_id="flexvolume-driver-integration-$(date '+%Y-%m-%d-%H-%M')"
echo "Test ID: ${TF_VAR_flexvolume_test_id}"

# Provision OCI instance and block storage volume for test.
terraform init .
terraform apply .

# Make sure we terraform destroy on exit.
function _trap_2 {
  if [ "${NO_DESTROY}" -ne "1" ]; then
    terraform destroy -force ${BASE_DIR}
  fi
  _trap_1
}
trap _trap_2 EXIT

INSTANCE_IP=$(terraform output instance_public_ip)
echo "Instance IP: ${INSTANCE_IP}"

VOLUME_NAME=$(terraform output volume_ocid | cut -d'.' -f5)
echo "Volume name: ${VOLUME_NAME}"

# Run tests on provisioned instance.
ssh \
    -o UserKnownHostsFile=/dev/null \
    -o LogLevel=quiet \
    -o StrictHostKeyChecking=no \
    -i _tmp/instance_key \
    opc@${INSTANCE_IP} \
    "bash --login -c \"sudo OCI_FLEXD_DRIVER_DIRECTORY=/tmp VOLUME_NAME=${VOLUME_NAME} /tmp/integration-tests -test.v\""
RET_CODE=$?
