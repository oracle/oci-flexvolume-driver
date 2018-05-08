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

driver_dir="/flexmnt/$VENDOR${VENDOR:+"~"}${DRIVER}"

LOG_FILE="$driver_dir/oci_flexvolume_driver.log"

if [ ! -d "$driver_dir" ]; then
  mkdir "$driver_dir"
fi

cp "/$DRIVER" "$driver_dir/.$DRIVER"
mv -f "$driver_dir/.$DRIVER" "$driver_dir/$DRIVER"

while : ; do
  touch $LOG_FILE
  tail -f $LOG_FILE
done
