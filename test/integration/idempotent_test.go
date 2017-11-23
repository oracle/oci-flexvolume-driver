// Copyright 2017 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"os"
	"testing"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/driver"
)

// TestIdempotent checks that Attach, MountDevice, and UnmountDevice are
// idempotent and (currently) that Detach is **not** idempotent.
func TestIdempotent(t *testing.T) {
	d := &driver.OCIFlexvolumeDriver{}
	opts := flexvolume.Options{
		"kubernetes.io/fsType":         "ext4",
		"kubernetes.io/pvOrVolumeName": fw.VolumeName,
		"kubernetes.io/readwrite":      "rw",
	}

	// Attach the volume to the instance.
	res := d.Attach(opts, fw.NodeName)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to Attach(): %+v", res)
	}
	t.Logf("Attach(): %+v", res)

	// Wait for the volume attachment.
	res = d.WaitForAttach(res.Device, opts)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to WaitForAttach(): %+v", res)
	}
	t.Logf("WaitForAttach(): %+v", res)

	device := res.Device

	// Attach again is no-op and does not error...
	res = d.Attach(opts, fw.NodeName)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to Attach(): %+v", res)
	}
	t.Logf("Attach(): %+v", res)

	// Mount the new device.
	mountPoint := "/tmp/" + fw.VolumeName
	err := os.MkdirAll(mountPoint, 0777)
	if err != nil {
		t.Fatalf("Failed to create mountpoint directory: %v", err)
	}
	res = d.MountDevice(mountPoint, device, opts)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to MountDevice(): %+v", res)
	}
	t.Logf("MountDevice(): %+v", res)

	// Mounting again is no-op and does not error...
	res = d.MountDevice(mountPoint, device, opts)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to MountDevice(): %+v", res)
	}
	t.Logf("MountDevice(): %+v", res)

	// Unmount the mountpoint.
	res = d.UnmountDevice(mountPoint)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to UnmountDevice(): %+v", res)
	}
	t.Logf("UnmountDevice(): %+v", res)

	// Unmounting again is no-op and does not error...
	res = d.UnmountDevice(mountPoint)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to UnmountDevice(): %+v", res)
	}
	t.Logf("UnmountDevice(): %+v", res)

	// Detach the volume from the instance.
	res = d.Detach(fw.VolumeName, fw.NodeName)
	if res.Status != flexvolume.StatusSuccess {
		t.Fatalf("Failed to Detach(): %+v", res)
	}
	t.Logf("Detach(): %+v", res)

	// Detaching the volume again is **NOT** idempotent and errors...
	res = d.Detach(fw.VolumeName, fw.NodeName)
	if res.Status != flexvolume.StatusFailure {
		t.Fatalf("Failed to Detach(): %+v", res)
	}
	t.Logf("Detach(): %+v", res)
}
