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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
	"github.com/oracle/oci-flexvolume-driver/pkg/iscsi"
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/client"
	"github.com/oracle/oci-go-sdk/core"
)

const (
	// FIXME: Assume lun 1 for now?? Can we get the LUN via the API?
	diskIDByPathTemplate = "/dev/disk/by-path/ip-%s:%d-iscsi-%s-lun-1"
	volumeOCIDTemplate   = "ocid1.volume.oc1.%s.%s"
	ocidPrefix           = "ocid1."
)

// OCIFlexvolumeDriver implements the flexvolume.Driver interface for OCI.
type OCIFlexvolumeDriver struct{}

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
	path := os.Getenv("OCI_FLEXD_CONFIG_DIRECTORY")
	if path != "" {
		return filepath.Join(path, "config.yaml")
	}

	return filepath.Join(GetDriverDirectory(), "config.yaml")
}

// Init checks that we have the appropriate credentials and metadata API access
// on driver initialisation.
func (d OCIFlexvolumeDriver) Init() flexvolume.DriverStatus {
	path := GetConfigPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		_, err = client.New(path)
		if err != nil {
			return flexvolume.Fail(err)
		}
	} else {
		log.Printf("Config file %q does not exist. Assuming worker node.", path)
	}

	return flexvolume.Succeed()
}

// deriveVolumeOCID will figure out the correct OCID for a volume
// based solely on the region key and volumeName. Because of differences
// across regions we need to impose some awkward logic here to get the correct
// OCID or if it is already an OCID then return the OCID.
func deriveVolumeOCID(regionKey string, volumeName string) string {
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

// Attach initiates the attachment of the given OCI volume to the k8s worker
// node.
func (d OCIFlexvolumeDriver) Attach(opts flexvolume.Options, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(GetConfigPath())
	if err != nil {
		return flexvolume.Fail(errors.New("Failed to create new client from config path " + err.Error()))
	}

	instance, err := c.GetInstanceByNodeName(nodeName)
	if err != nil {
		return flexvolume.Fail(errors.New("Failed to get instance by node name " + err.Error()))
	}

	volumeOCID := deriveVolumeOCID(c.GetConfig().Auth.RegionKey, opts["kubernetes.io/pvOrVolumeName"])

	log.Printf("Attaching volume %s -> instance %s", volumeOCID, *instance.Id)

	attachment, statusCode, err := c.AttachVolume(*instance.Id, volumeOCID)
	if err != nil {
		if statusCode != 409 {
			log.Printf("AttachVolume: %+v %d", err, statusCode)
			return flexvolume.Fail(errors.New("Failed to attach volume " + err.Error()))
		}
		// If we get a 409 conflict response when attaching we
		// presume that the device is already attached.
		log.Printf("Attach(): Volume %q already attached.", volumeOCID)
		attachment, err = c.FindVolumeAttachment(volumeOCID)
		if err != nil {
			return flexvolume.Fail(errors.New("Failed to find volume attachment " + err.Error()))
		}
		if *attachment.GetInstanceId() != *instance.Id {
			return flexvolume.Fail("Already attached to instance: ", *instance.Id)
		}
	}

	attachment, err = c.WaitForVolumeAttached(*attachment.GetId())
	if err != nil {
		return flexvolume.Fail(err)
	}

	log.Printf("attach: %s attached", *attachment.GetId())
	iscsiAttachment, ok := attachment.(core.IScsiVolumeAttachment)
	if !ok {
		return flexvolume.Fail("Only ISCSI volume attachments are currently supported")
	}

	return flexvolume.DriverStatus{
		Status: flexvolume.StatusSuccess,
		Device: fmt.Sprintf(diskIDByPathTemplate, *iscsiAttachment.Ipv4, *iscsiAttachment.Port, *iscsiAttachment.Iqn),
	}
}

// Detach detaches the volume from the worker node.
func (d OCIFlexvolumeDriver) Detach(pvOrVolumeName, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(GetConfigPath())
	if err != nil {
		return flexvolume.Fail(err)
	}

	volumeOCID := deriveVolumeOCID(c.GetConfig().Auth.RegionKey, pvOrVolumeName)
	attachment, err := c.FindVolumeAttachment(volumeOCID)
	if err != nil {
		return flexvolume.Fail(err)
	}

	err = c.DetachVolume(*attachment.GetId())
	if err != nil {
		return flexvolume.Fail(err)
	}

	err = c.WaitForVolumeDetached(*attachment.GetId())
	if err != nil {
		return flexvolume.Fail(err)
	}
	return flexvolume.Succeed()
}

// WaitForAttach searches for the the volume attachment created by Attach() and
// waits for its life cycle state to reach ATTACHED.
func (d OCIFlexvolumeDriver) WaitForAttach(mountDevice string, _ flexvolume.Options) flexvolume.DriverStatus {
	return flexvolume.DriverStatus{
		Status: flexvolume.StatusSuccess,
		Device: mountDevice,
	}
}

// IsAttached checks whether the volume is attached to the host.
// TODO(apryde): The documentation states that this is called from the Kubelet
// and KCM. Implementation requries credentials which won't be present on nodes
// but I've only ever seen it called by the KCM.
func (d OCIFlexvolumeDriver) IsAttached(opts flexvolume.Options, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(GetConfigPath())
	if err != nil {
		return flexvolume.Fail(err)
	}

	volumeOCID := deriveVolumeOCID(c.GetConfig().Auth.RegionKey, opts["kubernetes.io/pvOrVolumeName"])
	attachment, err := c.FindVolumeAttachment(volumeOCID)
	if err != nil {
		return flexvolume.DriverStatus{
			Status:   flexvolume.StatusSuccess,
			Message:  err.Error(),
			Attached: false,
		}
	}

	log.Printf("attach: found volume attachment %s", *attachment.GetId())

	return flexvolume.DriverStatus{
		Status:   flexvolume.StatusSuccess,
		Attached: true,
	}
}

// MountDevice connects the iSCSI target on the k8s worker node before mounting
// and (if necessary) formatting the disk.
func (d OCIFlexvolumeDriver) MountDevice(mountDir, mountDevice string, opts flexvolume.Options) flexvolume.DriverStatus {
	iSCSIMounter, err := iscsi.NewFromDevicePath(mountDevice)
	if err != nil {
		return flexvolume.Fail(err)
	}

	if isMounted, oErr := iSCSIMounter.DeviceOpened(mountDevice); oErr != nil {
		return flexvolume.Fail(oErr)
	} else if isMounted {
		return flexvolume.Succeed("Device already mounted. Nothing to do.")
	}

	if err = iSCSIMounter.AddToDB(); err != nil {
		return flexvolume.Fail(err)
	}
	if err = iSCSIMounter.SetAutomaticLogin(); err != nil {
		return flexvolume.Fail(err)
	}
	if err = iSCSIMounter.Login(); err != nil {
		return flexvolume.Fail(err)
	}

	if !waitForPathToExist(mountDevice, 20) {
		return flexvolume.Fail("Failed waiting for device to exist: ", mountDevice)
	}

	options := []string{}
	if opts[flexvolume.OptionReadWrite] == "ro" {
		options = []string{"ro"}
	}
	err = iSCSIMounter.FormatAndMount(mountDevice, mountDir, opts[flexvolume.OptionFSType], options)
	if err != nil {
		return flexvolume.Fail(err)
	}

	return flexvolume.Succeed()
}

// UnmountDevice unmounts the disk, logs out the iscsi target, and deletes the
// iscsi node record.
func (d OCIFlexvolumeDriver) UnmountDevice(mountPath string) flexvolume.DriverStatus {
	iSCSIMounter, err := iscsi.NewFromMountPointPath(mountPath)
	if err != nil {
		if err == iscsi.ErrMountPointNotFound {
			return flexvolume.Succeed("Mount point not found. Nothing to do.")
		}
		return flexvolume.Fail(err)
	}

	if err = iSCSIMounter.UnmountPath(mountPath); err != nil {
		return flexvolume.Fail(err)
	}
	if err = iSCSIMounter.Logout(); err != nil {
		return flexvolume.Fail(err)
	}
	if err = iSCSIMounter.RemoveFromDB(); err != nil {
		return flexvolume.Fail(err)
	}

	return flexvolume.Succeed()
}

// Mount is unimplemented as we use the --enable-controller-attach-detach flow
// and as such mount the drive in MountDevice().
func (d OCIFlexvolumeDriver) Mount(mountDir string, opts flexvolume.Options) flexvolume.DriverStatus {
	return flexvolume.NotSupported()
}

// Unmount is unimplemented as we use the --enable-controller-attach-detach flow
// and as such unmount the drive in UnmountDevice().
func (d OCIFlexvolumeDriver) Unmount(mountDir string) flexvolume.DriverStatus {
	return flexvolume.NotSupported()
}
