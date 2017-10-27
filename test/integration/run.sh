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
#  - $OCI_API_KEY (pointing to the pem file) or 
#    $OCI_API_KEY_VAR (containing the base64 encoded pem file content)
#
#  - $INSTANCE_KEY (pointing to the private key file) or 
#    $INSTANCE_KEY_VAR (containing the base64 encoded private key file content)
#
#  - $INSTANCE_KEY_PUB (pointing to the public key file) or 
#    $INSTANCE_KEY_PUB_VAR (containing the base64 encoded public key file content)

set -o errexit
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

function check_env() {
    if [[ ! -n "$1" ]] && [[ ! -n "$2" ]]; then
        echo "Error: Either $3 or $4 must be set"
        exit 1
    fi
}

function create_key_file() {
    if [[ -n "$1" ]]; then
        cp "$1" $3
    else
        echo "$2" | openssl enc -base64 -d -A > $3
    fi
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

# Check environment.
check_env "$OCI_API_KEY" "$OCI_API_KEY_VAR" "OCI_API_KEY" "OCI_API_KEY_VAR"
check_env "$INSTANCE_KEY" "$INSTANCE_KEY_VAR" "INSTANCE_KEY" "INSTANCE_KEY_VAR"
check_env "$INSTANCE_KEY_PUB" "$INSTANCE_KEY_PUB_VAR" "INSTANCE_KEY_PUB" "INSTANCE_KEY_PUB_VAR"

# Set keys from environment.
create_key_file "$OCI_API_KEY" "$OCI_API_KEY_VAR" "_tmp/oci_api_key.pem"
create_key_file "$INSTANCE_KEY" "$INSTANCE_KEY_VAR" "_tmp/instance_key"
create_key_file "$INSTANCE_KEY_PUB" "$INSTANCE_KEY_PUB_VAR" "_tmp/instance_key.pub"
chmod 600 _tmp/instance_key
chmod 600 _tmp/oci_api_key.pem

export TF_VAR_ssh_public_key="$(cat _tmp/instance_key.pub)"
export TF_VAR_ssh_private_key="$(cat _tmp/instance_key)"

export TF_VAR_test_id="flexvolume-driver-integration-$(date '+%Y-%m-%d-%H-%M')"

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
VOLUME_NAME=$(terraform output volume_ocid | cut -d'.' -f5)

# Run tests on provisioned instance.
ssh \
    -o UserKnownHostsFile=/dev/null \
    -o LogLevel=quiet \
    -o StrictHostKeyChecking=no \
    -i _tmp/instance_key \
    opc@${INSTANCE_IP} \
    "bash --login -c \"sudo OCI_FLEXD_DRIVER_DIRECTORY=/tmp VOLUME_NAME=${VOLUME_NAME} /tmp/integration-tests -test.v\""
RET_CODE=$?
