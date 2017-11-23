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
)

// Framework used to help with integration testing.
type Framework struct {
	VolumeName string
	NodeName   string
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
	}
}

// Init the framework.
func (f *Framework) Init() error {
	if f.VolumeName == "" {
		return errors.New("VOLUME_NAME env var unset")
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
