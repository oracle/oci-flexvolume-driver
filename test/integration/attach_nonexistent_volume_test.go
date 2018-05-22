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
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/block"
)

// TestAttachNonexistentVolume tests that attach fails for invalid volume OCIDs.
func TestAttachNonexistentVolume(t *testing.T) {
	d := &block.OCIFlexvolumeDriver{}
	opts := flexvolume.Options{
		"kubernetes.io/fsType":         "ext4",
		"kubernetes.io/pvOrVolumeName": "non-existent-volume",
		"kubernetes.io/readwrite":      "rw",
	}

	// Attach the volume to the instance.
	res := d.Attach(opts, fw.NodeName)
	if res.Status != flexvolume.StatusFailure {
		t.Fatalf("Expected Attach() to fail: %+v", res)
	}
	t.Logf("Attach(): %+v", res)
}
