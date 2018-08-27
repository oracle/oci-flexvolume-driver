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
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"log"
	"os"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/instancemeta"
)

// AuthConfig holds the configuration required for communicating with the OCI
// API.
type AuthConfig struct {
	Region               string `yaml:"region"`
	RegionKey            string `yaml:"regionKey"`
	TenancyOCID          string `yaml:"tenancy"`
	CompartmentOCID      string `yaml:"compartment"` // DEPRECATED (we no longer directly use this)
	UserOCID             string `yaml:"user"`
	PrivateKey           string `yaml:"key"`
	Passphrase           string `yaml:"passphrase"`
	PrivateKeyPassphrase string `yaml:"key_passphase"` // DEPRECATED
	Fingerprint          string `yaml:"fingerprint"`
	VcnOCID              string `yaml:"vcn"`
}

// Config holds the configuration for the OCI flexvolume driver.
type Config struct {
	Auth AuthConfig `yaml:"auth"`

	metadata              instancemeta.Interface
	UseInstancePrincipals bool `yaml:"useInstancePrincipals"`
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
	meta, err := c.metadata.Get()
	if err != nil {
		return err
	}
	if c.Auth.Region == "" {
		c.Auth.Region = meta.Region
	}
	if c.Auth.RegionKey == "" {
		c.Auth.RegionKey = meta.RegionKey
	}
	if c.Auth.CompartmentOCID == "" {
		c.Auth.CompartmentOCID = meta.CompartmentOCID
	}
	if c.Auth.Passphrase == "" && c.Auth.PrivateKeyPassphrase != "" {
		log.Print("config: auth.key_passphrase is DEPRECIATED and will be removed in a later release. Please set auth.passphrase instead.")
		c.Auth.Passphrase = c.Auth.PrivateKeyPassphrase
	}

	return nil
}

// validate checks that all required fields are populated.
func (c *Config) validate() error {
	return ValidateConfig(c).ToAggregate()
}

func validateAuthConfig(c *Config, fldPath *field.Path) field.ErrorList {
	errList := field.ErrorList{}

	if c.Auth.VcnOCID == "" {
		errList = append(errList, field.Required(fldPath.Child("vcn"), ""))
	}
	checkFields := map[string]string{"tenancy": c.Auth.TenancyOCID,
		"user":        c.Auth.UserOCID,
		"key":         c.Auth.PrivateKey,
		"fingerprint": c.Auth.Fingerprint}

	for fieldName, fieldValue := range checkFields {
		if fieldValue == "" {
			if !c.UseInstancePrincipals {
				errList = append(errList, field.Required(fldPath.Child(fieldName), ""))
			}
		} else {
			if c.UseInstancePrincipals {
				log.Printf("config: Instance principal authentication is enabled. Note the %s field will be ignored", fldPath.Child(fieldName))
			}
		}
	}

	return errList
}

// ValidateConfig validates the OCI Flexible Volume Provisioner config file.
func ValidateConfig(c *Config) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateAuthConfig(c, field.NewPath("auth"))...)
	return allErrs
}
