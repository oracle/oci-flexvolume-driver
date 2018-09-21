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

package framework

import (
	"errors"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/driver"
)

// Framework used to help with integration testing.
type Framework struct {
	VolumeName string
	NodeName   string
	NodeOCID   string
}

// New testing framework.
func New() *Framework {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	return &Framework{
		VolumeName: os.Getenv("VOLUME_NAME"),
		NodeName:   hostname,
		NodeOCID:   os.Getenv("NODE_OCID"),
	}
}

// NewDriver creates a new driver with a fake kubeclient
func (f *Framework) NewDriver() *driver.OCIFlexvolumeDriver {
	n := v1.Node{}
	n.Name = f.NodeName
	n.Spec.ProviderID = f.NodeOCID

	nl := new(v1.NodeList)
	nl.Items = []v1.Node{n}

	d := &driver.OCIFlexvolumeDriver{
		K: fake.NewSimpleClientset(nl),
	}
	return d
}

// Init the framework.
func (f *Framework) Init() error {
	if f.VolumeName == "" {
		return errors.New("VOLUME_NAME env var unset")
	}
	if f.NodeOCID == "" {
		return errors.New("NODE_OCID env var unset")
	}
	return nil
}

// Run the tests and exit with the status code
func (f *Framework) Run(run func() int) {
	os.Exit(run())
}

// Cleanup afterwards.
func (f *Framework) Cleanup() {
}
