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

package flexvolume

import (
	"testing"
)

const defaultTestOps = `{"kubernetes.io/fsType":"ext4","kubernetes.io/readwrite":"rw","kubernetes.io/pvOrVolumeName":"mockvolumeid"}`
const noVolIDTestOps = `{"kubernetes.io/fsType":"ext4","kubernetes.io/readwrite":"rw"}`

func assertSuccess(t *testing.T, expected DriverStatus, status DriverStatus) {
	if status != expected {
		t.Fatalf(`Expected '%#v' got '%#v'`, expected, status)
	}
}

func assertFailure(t *testing.T, expected DriverStatus, status DriverStatus) {
	if status != expected {
		t.Fatalf(`Expected '%#v' got '%#v'`, expected, status)
	}
}

func TestInit(t *testing.T) {
	status := ExecDriver([]Driver{mockFlexvolumeDriver{}},
		[]string{"oci", "init"})

	expected := DriverStatus{Status: "Success", Message: "Init success"}

	assertSuccess(t, expected, status)
}

// TestVolumeName tests that the getvolumename call-out results in
// StatusNotSupported as the call-out is broken as of the latest stable Kube
// release (1.6.4).
func TestGetVolumeName(t *testing.T) {
	status := ExecDriver([]Driver{mockFlexvolumeDriver{}},
		[]string{"oci", "getvolumename", defaultTestOps})

	expected := DriverStatus{Status: "Not supported", Message: "getvolumename is broken as of kube 1.6.4"}

	assertFailure(t, expected, status)

}

func TestNoVolumeIDDispatch(t *testing.T) {
	status := ExecDriver([]Driver{&mockFlexvolumeDriver{}},
		[]string{"oci", "attach", noVolIDTestOps, "nodeName"})

	expected := DriverStatus{
		Status:  "Failure",
		Message: "key 'kubernetes.io/pvOrVolumeName' not found in attach options",
	}
	assertFailure(t, expected, status)
}

func TestAttachUnsuported(t *testing.T) {
	status := ExecDriver([]Driver{&mockFlexvolumeDriver{}},
		[]string{"oci", "attach", defaultTestOps, "nodeName"})

	expected := DriverStatus{Status: "Not supported"}
	assertFailure(t, expected, status)
}
