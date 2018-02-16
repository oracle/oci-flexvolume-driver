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

package client

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/instancemeta"
)

var ociRegions = map[string]string{
	"phx": "us-phoenix-1",
	"iad": "us-ashburn-1",
	"fra": "eu-frankfurt-1",
}

// AuthConfig holds the configuration required for communicating with the OCI
// API.
type AuthConfig struct {
	Region               string `yaml:"region"`
	RegionKey            string `yaml:"regionKey"`
	TenancyOCID          string `yaml:"tenancy"`
	CompartmentOCID      string `yaml:"compartment"`
	UserOCID             string `yaml:"user"`
	PrivateKey           string `yaml:"key"`
	PrivateKeyPassphrase string `yaml:"key_passphase"`
	Fingerprint          string `yaml:"fingerprint"`
	VcnOCID              string `yaml:"vcn"`
}

type StorageConfig struct {
	MountTargetAd1OCID string `yaml:"mounttargetAD1ID"`
	MountTargetAd2OCID string `yaml:"mounttargetAD2ID"`
	MountTargetAd3OCID string `yaml:"mounttargetAD3ID"`
}

// Config holds the configuration for the OCI flexvolume driver.
type Config struct {
	Auth     AuthConfig    `yaml:"auth"`
	Storage  StorageConfig `yaml:"storage"`
	metadata instancemeta.Interface
}

// NewConfig creates a new Config based on the contents of the given io.Reader.
func NewConfig(r io.Reader) (*Config, error) {
	if r == nil {
		return nil, errors.New("no config provided")
	}

	c := &Config{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	c.metadata = instancemeta.New()

	if err := c.setDefaults(); err != nil {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// ConfigFromFile reads the file at the given path and marshals it into a Config
// object.
func ConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	return NewConfig(f)
}

func (c *Config) setDefaults() error {
	if c.Auth.Region == "" || c.Auth.CompartmentOCID == "" {
		meta, err := c.metadata.Get()
		if err != nil {
			return err
		}

		if c.Auth.Region == "" {
			c.Auth.Region = meta.Region
		}
		if c.Auth.CompartmentOCID == "" {
			c.Auth.CompartmentOCID = meta.CompartmentOCID
		}
	}

	err := c.setRegionFields(c.Auth.Region)
	if err != nil {
		return fmt.Errorf("setting config region fields: %v", err)
	}
	return nil
}

// setRegionFields accepts either a region short name or a region long name and
// sets both the Region and RegionKey fields.
func (c *Config) setRegionFields(region string) error {
	input := region
	region = strings.ToLower(region)

	var name, key string
	name, ok := ociRegions[region]
	if !ok {
		for key, name = range ociRegions {
			if name == region {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("tried to connect to unsupported OCI region %q", input)
		}
	} else {
		key = region
	}

	c.Auth.Region = name
	c.Auth.RegionKey = key

	return nil
}

// validate checks that all required fields are populated.
func (c *Config) validate() error {
	return validateConfig(c).ToAggregate()
}

func validateConfig(c *Config) field.ErrorList {
	errList := field.ErrorList{}

	if c.Auth.Region == "" {
		errList = append(errList, field.Required(field.NewPath("region"), ""))
	}
	if c.Auth.RegionKey == "" {
		errList = append(errList, field.Required(field.NewPath("region_key"), ""))
	}
	if c.Auth.TenancyOCID == "" {
		errList = append(errList, field.Required(field.NewPath("tenancy"), ""))
	}
	if c.Auth.CompartmentOCID == "" {
		errList = append(errList, field.Required(field.NewPath("compartment"), ""))
	}
	if c.Auth.UserOCID == "" {
		errList = append(errList, field.Required(field.NewPath("user"), ""))
	}
	if c.Auth.PrivateKey == "" {
		errList = append(errList, field.Required(field.NewPath("key"), ""))
	}
	if c.Auth.Fingerprint == "" {
		errList = append(errList, field.Required(field.NewPath("fingerprint"), ""))
	}

	return errList
}
