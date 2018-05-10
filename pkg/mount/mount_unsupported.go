// +build !linux

/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mount

// Mounter points to the mount path
type Mounter struct {
	mounterPath string
}

// Mount mounts the source to the target
func (mounter *Mounter) Mount(source string, target string, fstype string, options []string) error {
	return nil
}

// Unmount unmounts the source from the target
func (mounter *Mounter) Unmount(target string) error {
	return nil
}

// List returns a list of all mounted filesystems.
func (mounter *Mounter) List() ([]MountPoint, error) {
	return []MountPoint{}, nil
}

// IsLikelyNotMountPoint determines if a directory is not a mountpoint.
func (mounter *Mounter) IsLikelyNotMountPoint(file string) (bool, error) {
	return true, nil
}

// GetDeviceNameFromMount - given a mount point, find the device name from its global mount point
func (mounter *Mounter) GetDeviceNameFromMount(mountPath, pluginDir string) (string, error) {
	return "", nil
}

// DeviceOpened - verifies whether the device is still mounted on the system
func (mounter *Mounter) DeviceOpened(pathname string) (bool, error) {
	return false, nil
}

//PathIsDevice - determines if a path is a device
func (mounter *Mounter) PathIsDevice(pathname string) (bool, error) {
	return true, nil
}

func (mounter *SafeFormatAndMount) formatAndMount(source string, target string, fstype string, options []string) error {
	return nil
}

func (mounter *SafeFormatAndMount) diskLooksUnformatted(disk string) (bool, error) {
	return true, nil
}

// IsNotMountPoint - determines if a directory is not a mountpoint
func IsNotMountPoint(file string) (bool, error) {
	return true, nil
}
