// Copyright 2017 The OCI Flexvolume Driver Authors
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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/instancemeta"
)

var ociRegions = map[string]string{
	"phx": "us-phoenix-1",
	"iad": "us-ashburn-1",
	"fra": "eu-frankfurt-1",
}

// Config holds the details required to communicate with the OCI API.
type Config struct {
	Region         string `json:"region"`
	RegionKey      string `json:"region_key"`
	TenancyID      string `json:"tenancy_ocid"`
	CompartmentID  string `json:"compartment_ocid"`
	UserID         string `json:"user_ocid"`
	PrivateKeyFile string `json:"key_file"`
	Fingerprint    string `json:"fingerprint"`

	metadata instancemeta.Interface
}

// NewConfig creates a new Config based on the contents of the given io.Reader.
func NewConfig(r io.Reader) (*Config, error) {
	c, err := unmarshalConfig(r)
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

func unmarshalConfig(r io.Reader) (*Config, error) {
	c := &Config{}
	err := json.NewDecoder(r).Decode(c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return c, nil
}

func (c *Config) setDefaults() error {
	if c.Region == "" || c.CompartmentID == "" {
		meta, err := c.metadata.Get()
		if err != nil {
			return err
		}

		if c.Region == "" {
			c.Region = meta.Region
		}
		if c.CompartmentID == "" {
			c.CompartmentID = meta.CompartmentOCID
		}
	}

	err := c.setRegionFields(c.Region)
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

	c.Region = name
	c.RegionKey = key

	return nil
}

// validate checks that all required fields are populated.
func (c *Config) validate() error {
	return validateConfig(c).ToAggregate()
}

func validateConfig(c *Config) field.ErrorList {
	errList := field.ErrorList{}

	if c.Region == "" {
		errList = append(errList, field.Required(field.NewPath("region"), ""))
	}
	if c.RegionKey == "" {
		errList = append(errList, field.Required(field.NewPath("region_key"), ""))
	}
	if c.TenancyID == "" {
		errList = append(errList, field.Required(field.NewPath("tenancy_ocid"), ""))
	}
	if c.CompartmentID == "" {
		errList = append(errList, field.Required(field.NewPath("compartment_ocid"), ""))
	}
	if c.UserID == "" {
		errList = append(errList, field.Required(field.NewPath("user_ocid"), ""))
	}
	if c.PrivateKeyFile == "" {
		errList = append(errList, field.Required(field.NewPath("key_file"), ""))
	}
	if c.Fingerprint == "" {
		errList = append(errList, field.Required(field.NewPath("fingerprint"), ""))
	}

	return errList
}
