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

package filestorage

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
	"github.com/oracle/oci-flexvolume-driver/pkg/mount"
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/client"
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/common"
)

const (
	ocidFilestroagePrefix = "ocid1.filesystem."
	ocidPrefix            = "ocid1."
	mountCommand          = "/bin/mount"
)

// OCIFilestorageDriver implements the flexvolume.Driver interface for OCI.
type OCIFilestorageDriver struct{}

func (d OCIFilestorageDriver) Name() string {
	return "oci-filestorage"
}

// Init checks that we have the appropriate credentials and metadata API access
// on driver initialisation.
func (d OCIFilestorageDriver) Init() error {
	path := common.GetConfigPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		_, err = client.New(path)
		if err != nil {
			return err
		}
	} else {
		log.Printf("Config file %q does not exist. Assuming worker node.", path)
	}

	return nil
}

// Claim returns true if this driver handles this ocid
func (d OCIFilestorageDriver) Claim(volumeId string) bool {
	if strings.HasPrefix(volumeId, ocidFilestroagePrefix) {
		return true
	}
	return false
}

// Attach initiates the attachment of the given OCI volume to the k8s worker
// node.
func (d OCIFilestorageDriver) Attach(opts flexvolume.Options, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(common.GetConfigPath())
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	filesystemOCID := opts["kubernetes.io/pvOrVolumeName"]

	filesystem, err := c.GetFileSystem(filesystemOCID)
	if err != nil {
		log.Printf("Failed to get FileSystem")
		return flexvolume.Fail(err.Error())
	}

	mountTarget, err := c.GetMountTargetForAD(*filesystem.AvailabilityDomain)
	if err != nil {
		log.Printf("Failed to GetMountTargetForAD")
		return flexvolume.Fail(err.Error())
	}

	privateIps, err := c.GetMountTargetIPS(mountTarget)
	if err != nil {
		log.Printf("Failed GetMountTargetIPS")
		return flexvolume.Fail(err.Error())
	}
	log.Printf("Mount TargetIPS:%v", privateIps)

	if len(privateIps) == 0 {
		return flexvolume.Fail("MountRarget:% has zero private IPs", *mountTarget.Id)
	}

	// FIXME can I return IP:path???
	// does it have to be a vaild path?
	var path = fmt.Sprintf("/mnt/%s/%s", *privateIps[0].IpAddress, filesystemOCID)

	err = c.AttachFileSystemToMountTarget(filesystem, mountTarget, path)
	if err != nil {
		log.Printf("Failed AttachFileSystemToMountTarget")
		return flexvolume.Fail(err.Error())
	}

	return flexvolume.DriverStatus{
		Status: flexvolume.StatusSuccess,
		Device: path,
	}
}

// Detach detaches the volume from the worker node.
func (d OCIFilestorageDriver) Detach(pvOrVolumeName, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(common.GetConfigPath())
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	filesystemOCID := pvOrVolumeName

	filesystem, err := c.GetFileSystem(filesystemOCID)
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	mountTarget, err := c.GetMountTargetForAD(*filesystem.AvailabilityDomain)
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	privateIps, err := c.GetMountTargetIPS(mountTarget)
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	if len(privateIps) == 0 {
		return flexvolume.Fail("MountRarget:% has zero private IPs", *mountTarget.Id)
	}

	var path = fmt.Sprintf("/mnt/%s/%s", *privateIps[0].IpAddress, filesystemOCID)

	err = c.DetachFileSystemToMountTarget(filesystem, mountTarget, path)
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	return flexvolume.Succeed("Detach %s from %s", *filesystem.Id, *mountTarget.Id)
}

// WaitForAttach does nothing but return true as we have done the
// wait in the mountdevice calls and this means that no creds are needed on worker nodes
func (d OCIFilestorageDriver) WaitForAttach(mountDevice string, _ flexvolume.Options) flexvolume.DriverStatus {
	return flexvolume.DriverStatus{
		Status: flexvolume.StatusSuccess,
		Device: mountDevice,
	}
}

// IsAttached checks whether the volume is attached to the host.
func (d OCIFilestorageDriver) IsAttached(opts flexvolume.Options, nodeName string) flexvolume.DriverStatus {
	c, err := client.New(common.GetConfigPath())
	if err != nil {
		return flexvolume.Fail(err.Error())
	}

	filesystemOCID := opts["kubernetes.io/pvOrVolumeName"]

	filesystem, err := c.GetFileSystem(filesystemOCID)
	if err != nil {
		log.Printf("Failed to get FileSystem")
		return flexvolume.Fail(err.Error())
	}

	mountTarget, err := c.GetMountTargetForAD(*filesystem.AvailabilityDomain)
	if err != nil {
		log.Printf("Failed to GetMountTargetForAD")
		return flexvolume.Fail(err.Error())
	}

	privateIps, err := c.GetMountTargetIPS(mountTarget)
	if err != nil {
		log.Printf("Failed GetMountTargetIPS")
		return flexvolume.Fail(err.Error())
	}
	log.Printf("Mount TargetIPS:%v", privateIps)

	if len(privateIps) == 0 {
		return flexvolume.Fail("MountRarget:% has zero private IPs", *mountTarget.Id)
	}

	var path = fmt.Sprintf("/mnt/%s/%s", *privateIps[0].IpAddress, filesystemOCID)

	attached, err := c.IsFileSystemAttached(filesystem, mountTarget, path)
	if err != nil {
		log.Printf("Failed IsFileSystemAttached")
		return flexvolume.Fail(err.Error())
	}
	return flexvolume.DriverStatus{
		Status:   flexvolume.StatusSuccess,
		Attached: attached,
	}
}

// MountDevice connects the iSCSI target on the k8s worker node before mounting
// and (if necessary) formatting the disk.
func (d OCIFilestorageDriver) MountDevice(mountDir, mountDevice string, opts flexvolume.Options) flexvolume.DriverStatus {

	parts := strings.Split(mountDevice, "/")
	IPaddress := parts[2]
	source := fmt.Sprintf("%s:%s", IPaddress, mountDevice)

	mounter := mount.New("")

	notMnt, err := mounter.IsLikelyNotMountPoint(mountDir)
	log.Printf("NFS mount set up: %s %v %v", mountDir, !notMnt, err)
	if err != nil && !os.IsNotExist(err) {
		return flexvolume.Fail(err.Error())
	}
	if !notMnt {
		return flexvolume.Fail("Already mount point")
	}
	os.MkdirAll(mountDir, 0750)

	options := []string{}
	/*if b.readOnly {
		options = append(options, "ro")
	}*/
	//mountOptions := volume.JoinMountOptions(b.mountOptions, options)
	err = mounter.Mount(source, mountDir, "nfs", options)
	if err != nil {
		log.Printf("Failed to mount:%v")
		return flexvolume.Fail(err.Error())
	}

	return flexvolume.Fail("MountDevice")
}

// UnmountDevice unmounts the disk, logs out the iscsi target, and deletes the
// iscsi node record.
func (d OCIFilestorageDriver) UnmountDevice(mountPath string) flexvolume.DriverStatus {
	mounter := mount.New("")

	err := mounter.Unmount(mountPath)
	if err != nil {
		log.Printf("Failed to unmount:%v")
		return flexvolume.Fail(err.Error())
	}

	return flexvolume.Succeed(fmt.Sprintf("UnmountDevice:", mountPath))
}

// Mount is unimplemented as we use the --enable-controller-attach-detach flow
// and as such mount the drive in MountDevice().
func (d OCIFilestorageDriver) Mount(mountDir string, opts flexvolume.Options) flexvolume.DriverStatus {
	return flexvolume.NotSupported("Mount")
}

// Unmount is unimplemented as we use the --enable-controller-attach-detach flow
// and as such unmount the drive in UnmountDevice().
func (d OCIFilestorageDriver) Unmount(mountDir string) flexvolume.DriverStatus {
	return flexvolume.NotSupported("Unmount")
}
