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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Status denotes the state of a Flexvolume call.
type Status string

// Options is the map (passed as JSON) to some Flexvolume calls.
type Options map[string]string

const (
	// StatusSuccess indicates that the driver call has succeeded.
	StatusSuccess Status = "Success"
	// StatusFailure indicates that the driver call has failed.
	StatusFailure Status = "Failure"
	// StatusNotSupported indicates that the driver call is not supported.
	StatusNotSupported Status = "Not supported"
)

// DriverStatus of a Flexvolume driver call.
type DriverStatus struct {
	// Status of the callout. One of "Success", "Failure" or "Not supported".
	Status Status `json:"status"`
	// Reason for success/failure.
	Message string `json:"message,omitempty"`
	// Path to the device attached. This field is valid only for attach calls.
	// e.g: /dev/sdx
	Device string `json:"device,omitempty"`
	// Represents volume is attached on the node.
	Attached bool `json:"attached,omitempty"`
}

// Option keys
const (
	OptionFSType    = "kubernetes.io/fsType"
	OptionReadWrite = "kubernetes.io/readwrite"
	OptionKeySecret = "kubernetes.io/secret"
	OptionFSGroup   = "kubernetes.io/fsGroup"
	OptionMountsDir = "kubernetes.io/mountsDir"

	OptionKeyPodName      = "kubernetes.io/pod.name"
	OptionKeyPodNamespace = "kubernetes.io/pod.namespace"
	OptionKeyPodUID       = "kubernetes.io/pod.uid"

	OptionKeyServiceAccountName = "kubernetes.io/serviceAccount.name"

	OptionKeypvOrVolumeName = "kubernetes.io/pvOrVolumeName"
)

// Driver is the main Flexvolume interface.
type Driver interface {
	Name() string
	Claim(volumeid string) bool
	Init() error
	Attach(opts Options, nodeName string) DriverStatus
	Detach(mountDevice, nodeName string) DriverStatus
	WaitForAttach(mountDevice string, opts Options) DriverStatus
	IsAttached(opts Options, nodeName string) DriverStatus
	MountDevice(mountDir, mountDevice string, opts Options) DriverStatus
	UnmountDevice(mountDevice string) DriverStatus
	Mount(mountDir string, opts Options) DriverStatus
	Unmount(mountDir string) DriverStatus
}

// ExitWithResult outputs the given Result and exits with the appropriate exit
// code.
func ExitWithResult(result DriverStatus) {
	code := 1
	if result.Status == StatusSuccess || result.Status == StatusNotSupported {
		code = 0
	}

	res, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling result: %v", err)
		fmt.Println(`{"status":"Failure","message":"Error marshaling result to JSON"}`)
	} else {
		s := string(res)
		log.Printf("Command result: %s", s)
		fmt.Println(s)
	}
	os.Exit(code)
}

// Fail creates a StatusFailure Result with a given message.
func Fail(s string, a ...interface{}) DriverStatus {
	msg := fmt.Sprintf(s, a...)
	return DriverStatus{
		Status:  StatusFailure,
		Message: msg,
	}
}

// Succeed creates a StatusSuccess Result with a given message.
func Succeed(s string, a ...interface{}) DriverStatus {
	return DriverStatus{
		Status:  StatusSuccess,
		Message: fmt.Sprintf(s, a...),
	}
}

// NotSupported creates a StatusNotSupported Result with a given message.
func NotSupported(s string, a ...interface{}) DriverStatus {
	return DriverStatus{
		Status:  StatusNotSupported,
		Message: fmt.Sprintf(s, a...),
	}
}

func processOpts(optsStr string) (Options, error) {
	opts := make(Options)
	if err := json.Unmarshal([]byte(optsStr), &opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options %q: %v", optsStr, err)
	}

	opts, err := DecodeKubeSecrets(opts)
	if err != nil {
		return nil, err
	}

	return opts, nil
}

func ClaimVolume(volumeId string, drivers []Driver) Driver {
	for _, driver := range drivers {
		if driver.Claim(volumeId) {
			return driver
		}
	}
	return nil
}

func Init(drivers []Driver) DriverStatus {
	var errors []error
	for _, driver := range drivers {
		err := driver.Init()
		if err != nil {
			errors = append(errors, fmt.Errorf("%s:%s", driver.Name(), err))
		}
	}
	if len(errors) != 0 {
		return Fail("Init failed with errors:%v", errors)
	}
	return Succeed("Init success")
}

func Attach(drivers []Driver, opts Options, nodeName string) DriverStatus {
	volumeId, ok := opts[OptionKeypvOrVolumeName]
	if !ok {
		return Fail("key '%s' not found in attach options",
			OptionKeypvOrVolumeName)
	}
	return ClaimVolume(volumeId, drivers).Attach(opts, nodeName)
}

/*	Detach(mountDevice, nodeName string) DriverStatus
	WaitForAttach(mountDevice string, opts Options) DriverStatus
	IsAttached(opts Options, nodeName string) DriverStatus
	MountDevice(mountDir, mountDevice string, opts Options) DriverStatus
	UnmountDevice(mountDevice string) DriverStatus
	Mount(mountDir string, opts Options) DriverStatus
	Unmount(mountDir string) DriverStatus*/

// ExecDriver executes the appropriate FlexvolumeDriver command based on
// recieved call-out.
func ExecDriver(drivers []Driver, args []string) DriverStatus {
	if len(args) < 2 {
		return Fail("Expected at least one argument")
	}

	log.Printf("'%s %s' called with %s", args[0], args[1], args[2:])

	switch args[1] {
	// <driver executable> init
	case "init":
		return Init(drivers)
	// <driver executable> getvolumename <json options>
	// Currently broken as of lates kube release (1.6.4). Work around hardcodes
	// exiting with StatusNotSupported.
	// TODO(apryde): Investigate current situation and version support
	// requirements.
	case "getvolumename":
		return NotSupported("getvolumename is broken as of kube 1.6.4")

	// <driver executable> attach <json options> <node name>
	case "attach":
		if len(args) != 4 {
			return Fail("attach expected exactly 4 arguments; got ", args)
		}

		opts, err := processOpts(args[2])
		if err != nil {
			return Fail(err.Error())
		} else {
			nodeName := args[3]

			return Attach(drivers, opts, nodeName)
		}

	// <driver executable> detach <mount device> <node name>
	case "detach":
		if len(args) != 4 {
			return Fail("detach expected exactly 4 arguments; got ", args)
		}

		volumeName := args[2]
		nodeName := args[3]
		driver := ClaimVolume(volumeName, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeName)
		}
		return driver.Detach(volumeName, nodeName)

	// <driver executable> waitforattach <mount device> <json options>
	case "waitforattach":
		if len(args) != 4 {
			return Fail("waitforattach expected exactly 4 arguments; got ", args)
		}

		mountDevice := args[2]
		opts, err := processOpts(args[3])
		if err != nil {
			return Fail(err.Error())
		}
		volumeId, ok := opts[OptionKeypvOrVolumeName]
		if !ok {
			return Fail("key '%s' not found in attach options",
				OptionKeypvOrVolumeName)
		}

		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.WaitForAttach(mountDevice, opts)

	// <driver executable> isattached <json options> <node name>
	case "isattached":
		if len(args) != 4 {
			return Fail("isattached expected exactly 4 arguments; got ", args)
		}

		opts, err := processOpts(args[2])
		if err != nil {
			return Fail(err.Error())
		}
		volumeId, ok := opts[OptionKeypvOrVolumeName]
		if !ok {
			return Fail("key '%s' not found in attach options:%#v",
				OptionKeypvOrVolumeName,
				opts)
		}

		nodeName := args[3]
		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.IsAttached(opts, nodeName)

	// <driver executable> mountdevice <mount dir> <mount device> <json options>
	case "mountdevice":
		if len(args) != 5 {
			return Fail("mountdevice expected exactly 5 arguments; got ", args)
		}

		mountDir := args[2]
		mountDevice := args[3]

		opts, err := processOpts(args[4])
		if err != nil {
			return Fail(err.Error())
		}

		volumeId, ok := opts[OptionKeypvOrVolumeName]
		if !ok {
			return Fail("key '%s' not found in attach options",
				OptionKeypvOrVolumeName)
		}

		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.MountDevice(mountDir, mountDevice, opts)

	// <driver executable> unmountdevice <mount dir>
	case "unmountdevice":
		if len(args) != 3 {
			return Fail("unmountdevice expected exactly 3 arguments; got ", args)
		}

		mountDir := args[2]

		// assumes that the last part of mountDir is the volumeId
		parts := strings.Split(mountDir, "/")
		volumeId := parts[len(parts)-1]

		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.UnmountDevice(mountDir)

	// <driver executable> mount <mount dir> <json options>
	case "mount":
		if len(args) != 4 {
			return Fail("mount expected exactly 4 arguments; got ", args)
		}

		mountDir := args[2]

		opts, err := processOpts(args[3])
		if err != nil {
			return Fail(err.Error())
		}

		volumeId, ok := opts[OptionKeypvOrVolumeName]
		if !ok {
			return Fail("key '%s' not found in attach options",
				OptionKeypvOrVolumeName)
		}

		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.Mount(mountDir, opts)

	// <driver executable> unmount <mount dir>
	case "unmount":
		if len(args) != 3 {
			return Fail("mount expected exactly 3 arguments; got ", args)
		}

		mountDir := args[2]

		// assumes that the last part of mountDir is the volumeId
		parts := strings.Split(mountDir, "/")
		volumeId := parts[len(parts)-1]

		driver := ClaimVolume(volumeId, drivers)
		if driver != nil {
			return Fail("Unable to find flexdriver for volume id '%s'", volumeId)
		}

		return driver.Unmount(mountDir)

	default:
		return Fail("Invalid command; got ", args)
	}
}
