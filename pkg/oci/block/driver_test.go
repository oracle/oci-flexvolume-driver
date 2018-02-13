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

package block

import "testing"

var volumeOCIDTests = []struct {
	regionKey  string
	volumeName string
	expected   string
}{
	{"phx", "aaaaaa", "ocid1.volume.oc1.phx.aaaaaa"},
	{"iad", "aaaaaa", "ocid1.volume.oc1.iad.aaaaaa"},
	{"fra", "aaaaaa", "ocid1.volume.oc1.eu-frankfurt-1.aaaaaa"},
}

func TestDeriveVolumeOCID(t *testing.T) {
	for _, tt := range volumeOCIDTests {
		result := deriveVolumeOCID(tt.regionKey, tt.volumeName)
		if result != tt.expected {
			t.Errorf("Failed to derive OCID. Expected %s got %s", tt.expected, result)
		}
	}
}
