#!/bin/bash

# Copyright 2017 Oracle and/or its affiliates. All rights reserved.
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

set -o errexit
set -o pipefail

VENDOR=oracle
DRIVER=oci

ORACLE_OCI_FOLDER=$VENDOR${VENDOR:+"~"}${DRIVER}

driver_dir="/flexmnt/$ORACLE_OCI_FOLDER"

LOG_FILE="$driver_dir/oci_flexvolume_driver.log"

config_file_name="config.yaml"
config_tmp_dir="/tmp"

CONFIG_FILE="$config_tmp_dir/$config_file_name"

if [ ! -d "$driver_dir" ]; then
  mkdir "$driver_dir"
fi

if [ ! -d "$driver_dir-bvs" ]; then
  mkdir "$driver_dir-bvs"
fi

cp "/$DRIVER" "$driver_dir/.$DRIVER"
mv -f "$driver_dir/.$DRIVER" "$driver_dir/$DRIVER"

ln -sf "../$ORACLE_OCI_FOLDER/$DRIVER" "$driver_dir-bvs/$DRIVER-bvs"

if [ -f "$CONFIG_FILE" ]; then
  cp  "$CONFIG_FILE"  "$driver_dir/$config_file_name"
fi

while : ; do
  touch $LOG_FILE
  tail -f $LOG_FILE
done
