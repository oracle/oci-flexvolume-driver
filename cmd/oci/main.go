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

package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/oracle/oci-flexvolume-driver/pkg/flexvolume"
	"github.com/oracle/oci-flexvolume-driver/pkg/oci/driver"
)

// version/build is set at build time to the version of the driver being built.
var version string
var build string

// DefaultDriver is the default flexvolume driver symlink extension.
const DefaultDriver string = "oci-bvs"

var (
	drivers = make(map[string]flexvolume.Driver)
)

// GetLogPath returns the default path to the driver log file.
func GetLogPath() string {
	path := os.Getenv("OCI_FLEXD_DRIVER_LOG_DIR")
	if path == "" {
		path = driver.GetDriverDirectory()
	}
	return path + "/oci_flexvolume_driver.log"
}

func registerDrivers() {
	registerDriver("oci-bvs", &driver.OCIFlexvolumeDriver{})
}

func main() {
	// TODO: Maybe use sirupsen/logrus?
	f, err := os.OpenFile(GetLogPath(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	log.SetPrefix(fmt.Sprintf("%d ", os.Getpid()))

	log.SetOutput(f)

	log.Printf("OCI FlexVolume Driver version: %s (%s)", version, build)

	registerDrivers()

	driver, err := getDriverFromArgs()
	if err != nil {
		log.Fatalf(err.Error())
	}

	flexvolume.ExitWithResult(flexvolume.ExecDriver(driver, os.Args))
}

func getDriverFromArgs() (flexvolume.Driver, error) {
	driver, err := getDriver(DefaultDriver) //Block volume is default
	if err != nil {
		return nil, err
	}

	if len(os.Args) == 0 {
		log.Printf("No arguments found, using default driver %s", DefaultDriver)
		return driver, nil
	}

	dir := path.Base(os.Args[0])
	dir = strings.TrimPrefix(dir, "oracle~")

	if dir != "oci" && dir != DefaultDriver {
		driver, err = getDriver(dir)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Using %s driver", dir)

	return driver, nil
}

func registerDriver(name string, driver flexvolume.Driver) {
	if drivers[name] == nil {
		drivers[name] = driver
	}
}

func getDriver(name string) (flexvolume.Driver, error) {
	driver, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("could not find a registered driver for %s", name)
	}
	return driver, nil
}
