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

package driver

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
)

// DefaultDriver is the default flexvolume driver symlink extension.
const DefaultDriver string = "oci-bvs"

const (
	volumeOCIDTemplate = "ocid1.volume.oc1.%s.%s"
	ocidPrefix         = "ocid1."
)

var (
	drivers = make(map[string]flexvolume.Driver)
)

// Register adds a driver to the drivers map.
func Register(name string, driver flexvolume.Driver) {
	driver, ok := drivers[name]
	if !ok {
		drivers[name] = driver
	}
}

// Get returns a driver from the map associated with a given key.
func Get(name string) (flexvolume.Driver, error) {
	driver, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("could not find a registered driver for %s", name)
	}
	return driver, nil
}

// NameFromArgs returns the name (map key) of the driver from a given path.
func NameFromArgs(args []string) string {
	if len(args) == 0 {
		return DefaultDriver
	}

	driverName := strings.TrimPrefix(path.Base(os.Args[0]), "oracle~")
	if driverName == "oci" {
		return DefaultDriver
	}
	return driverName
}

// GetDirectory gets the ath for the flexvolume driver either from the
// env or default.
func GetDirectory() string {
	// TODO(apryde): Document this ENV var.
	path := os.Getenv("OCI_FLEXD_DRIVER_DIRECTORY")
	if path == "" {
		path = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci"
	}
	return path
}

// GetConfigPath gets the path to the OCI API credentials.
func GetConfigPath() string {
	path := os.Getenv("OCI_FLEXD_CONFIG_DIRECTORY")
	if path != "" {
		return filepath.Join(path, "config.yaml")
	}

	return filepath.Join(GetDirectory(), "config.yaml")
}

// deriveVolumeOCID will figure out the correct OCID for a volume
// based solely on the region key and volumeName. Because of differences
// across regions we need to impose some awkward logic here to get the correct
// OCID or if it is already an OCID then return the OCID.
func DeriveVolumeOCID(regionKey string, volumeName string) string {
	if strings.HasPrefix(volumeName, ocidPrefix) {
		return volumeName
	}

	var volumeOCID string
	if regionKey == "fra" {
		volumeOCID = fmt.Sprintf(volumeOCIDTemplate, "eu-frankfurt-1", volumeName)
	} else {
		volumeOCID = fmt.Sprintf(volumeOCIDTemplate, regionKey, volumeName)
	}

	return volumeOCID
}
