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
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/instancemeta"
)

const completeConfigString = `{
  "region": "us-phoenix-1",
  "tenancy_ocid": "ocid1.tenancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "compartment_ocid": "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "user_ocid": "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "key_file": "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
  "fingerprint": "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e"
}`

func TestConfigDefaulting(t *testing.T) {
	expectedCompartmentID := "ocid1.compartment.oc1..aaaaaaaa3um2atybwhder4qttfhgon4j3hcxgmsvnyvx4flfjyewkkwfzwnq"
	expectedRegion := "us-phoenix-1"
	expectedRegionKey := "phx"

	cfg := &Config{metadata: instancemeta.NewMock(
		&instancemeta.InstanceMetadata{
			CompartmentOCID: expectedCompartmentID,
			Region:          expectedRegionKey, // instance metadata API only returns the region key
		},
	)}

	err := cfg.setDefaults()
	if err != nil {
		t.Fatalf("cfg.setDefaults() => %v, expected no error", err)
	}

	if cfg.Region != expectedRegion {
		t.Fatalf("Expected cfg.Region = %q, got %q", cfg.Region, expectedRegion)
	}
	if cfg.RegionKey != expectedRegionKey {
		t.Fatalf("Expected cfg.RegionKey = %q, got %q", cfg.RegionKey, expectedRegionKey)
	}

	if cfg.CompartmentID != expectedCompartmentID {
		t.Fatalf("Expected cfg.CompartmentID = %q, got %q", cfg.CompartmentID, expectedCompartmentID)
	}
}

func TestUnmarshalConfig(t *testing.T) {
	r := strings.NewReader(completeConfigString)
	_, err := unmarshalConfig(r)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling config: %v", err)
	}
}

func TestConfigSetRegion(t *testing.T) {
	var testCases = []struct {
		in          string
		region      string
		shortRegion string
		shouldErr   bool
	}{
		{"us-phoenix-1", "us-phoenix-1", "phx", false},
		{"US-PHOENIX-1", "us-phoenix-1", "phx", false},
		{"phx", "us-phoenix-1", "phx", false},
		{"PHX", "us-phoenix-1", "phx", false},

		{"us-ashburn-1", "us-ashburn-1", "iad", false},
		{"US-ASHBURN-1", "us-ashburn-1", "iad", false},
		{"iad", "us-ashburn-1", "iad", false},
		{"IAD", "us-ashburn-1", "iad", false},

		{"eu-frankfurt-1", "eu-frankfurt-1", "fra", false},
		{"EU-FRANKFURT-1", "eu-frankfurt-1", "fra", false},
		{"fra", "eu-frankfurt-1", "fra", false},
		{"FRA", "eu-frankfurt-1", "fra", false},

		// error cases
		{"us-east", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range testCases {
		t.Run(tt.in, func(t *testing.T) {
			cfg := &Config{}
			err := cfg.setRegionFields(tt.in)
			if err != nil {
				if !tt.shouldErr {
					t.Errorf("SetRegionFields(%q) unexpected error: %v", tt.in, err)
				}
			}

			if cfg.Region != tt.region {
				t.Errorf("SetRegionFields(%q) => {Region: %q}; want {Region: %q}", tt.in, cfg.Region, tt.region)
			}
			if cfg.RegionKey != tt.shortRegion {
				t.Errorf("SetRegionFields(%q) => {RegionShortName: %q}; want {RegionShortName: %q}", tt.in, cfg.RegionKey, tt.shortRegion)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name string
		in   *Config
		errs field.ErrorList
	}{
		{
			name: "valid",
			in: &Config{
				Region:         "us-phoenix-1",
				RegionKey:      "phx",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{},
		}, {
			name: "missing_region",
			in: &Config{
				RegionKey:      "phx",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "region", BadValue: ""},
			},
		}, {
			name: "missing_region_key",
			in: &Config{
				Region:         "us-phoenix-1",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "region_key", BadValue: ""},
			},
		}, {
			name: "missing_tenancy_ocid",
			in: &Config{
				Region:         "us-phoenix-1",
				RegionKey:      "phx",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "tenancy_ocid", BadValue: ""},
			},
		}, {
			name: "missing_compartment_ocid",
			in: &Config{
				Region:         "us-phoenix-1",
				RegionKey:      "phx",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "compartment_ocid", BadValue: ""},
			},
		}, {
			name: "missing_user_ocid",
			in: &Config{
				Region:         "us-phoenix-1",
				RegionKey:      "phx",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
				Fingerprint:    "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "user_ocid", BadValue: ""},
			},
		}, {
			name: "missing_key_file",
			in: &Config{
				Region:        "us-phoenix-1",
				RegionKey:     "phx",
				CompartmentID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Fingerprint:   "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "key_file", BadValue: ""},
			},
		}, {
			name: "missing_figerprint",
			in: &Config{
				Region:         "us-phoenix-1",
				RegionKey:      "phx",
				CompartmentID:  "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				TenancyID:      "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				UserID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PrivateKeyFile: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci/flexvolume_driver.pem",
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "fingerprint", BadValue: ""},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := validateConfig(tt.in)
			if !reflect.DeepEqual(result, tt.errs) {
				t.Errorf("ValidateConfig(%#v)\n=> %#v\nExpected: %#v", tt.in, result, tt.errs)
			}
		})
	}
}
