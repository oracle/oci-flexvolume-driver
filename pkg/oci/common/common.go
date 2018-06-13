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

package common

import (
	"os"
)

// GetDriverDirectory gets the path for the flexvolume driver either from the
// env or default.
func GetDriverDirectory() string {
	// TODO(apryde): Document this ENV var.
	path := os.Getenv("OCI_FLEXD_DRIVER_DIRECTORY")
	if path == "" {
		path = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci"
	}
	return path
}

// GetConfigPath gets the path to the OCI API credentials.
func GetConfigPath() string {
	return GetDriverDirectory() + "/config.yaml"
}
