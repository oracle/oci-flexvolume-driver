package common

import (
	"os"
)

// GetDriverDirectory gets the ath for the flexvolume driver either from the
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
