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
	"testing"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
)

// TestMountNonexistentPath checks an error is return when the required mout
// point does not exist during mounting.
func TestMountNonexistentPath(t *testing.T) {
	d := fw.NewDriver()
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

	// Finally detach the device whether or not the test fails following this
	// point.
	defer func() {
		res = d.Detach(fw.VolumeName, fw.NodeName)
		if res.Status != flexvolume.StatusSuccess {
			t.Fatalf("Failed to Detach(): %+v", res)
		}
		t.Logf("Detach(): %+v", res)
	}()

	// Mount the new device. This should fail as no matching path has been
	// created on the host.
	device := res.Device
	mountPoint := "/tmp/" + fw.VolumeName
	res = d.MountDevice(mountPoint, device, opts)
	if res.Status != flexvolume.StatusFailure {
		t.Fatalf("Expected MountDevice() to fail: %+v", res)
	}
	t.Logf("MountDevice(): %+v", res)
}
