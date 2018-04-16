#!/bin/sh

# https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/flexvolume-deployment.md#driver-deployment-script

set -o errexit
set -o pipefail

VENDOR=oracle
DRIVER=oci

# Assuming the single driver file is located at /$DRIVER inside the DaemonSet image.

driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}
if [ ! -d "/flexmnt/$driver_dir" ]; then
  mkdir "/flexmnt/$driver_dir"
fi

cp "/$DRIVER" "/flexmnt/$driver_dir/.$DRIVER"
mv -f "/flexmnt/$driver_dir/.$DRIVER" "/flexmnt/$driver_dir/$DRIVER"

while : ; do
  # Sleep but allow shutdown signals to still be honored.
  sleep 3600 &
  wait
done
