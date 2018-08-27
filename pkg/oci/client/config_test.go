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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/instancemeta"
)

func TestConfigDefaulting(t *testing.T) {
	expectedCompartmentOCID := "ocid1.compartment.oc1..aaaaaaaa3um2atybwhder4qttfhgon4j3hcxgmsvnyvx4flfjyewkkwfzwnq"
	expectedRegion := "us-phoenix-1"
	expectedRegionKey := "phx"

	cfg := &Config{metadata: instancemeta.NewMock(
		&instancemeta.InstanceMetadata{
			CompartmentOCID: expectedCompartmentOCID,
			RegionKey:       expectedRegionKey,
			Region:          expectedRegion,
		},
	)}

	err := cfg.setDefaults()
	if err != nil {
		t.Fatalf("cfg.setDefaults() => %v, expected no error", err)
	}

	if cfg.Auth.Region != expectedRegion {
		t.Fatalf("Expected cfg.Region = %q, got %q", cfg.Auth.Region, expectedRegion)
	}
	if cfg.Auth.RegionKey != expectedRegionKey {
		t.Fatalf("Expected cfg.RegionKey = %q, got %q", cfg.Auth.RegionKey, expectedRegionKey)
	}

	if cfg.Auth.CompartmentOCID != expectedCompartmentOCID {
		t.Fatalf("Expected cfg.CompartmentOCID = %q, got %q", cfg.Auth.CompartmentOCID, expectedCompartmentOCID)
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
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{},
		}, {
			name: "allows missing_region",
			in: &Config{
				Auth: AuthConfig{
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{},
		}, {
			name: "allows missing_region_key",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{},
		}, {
			name: "missing_tenancy_ocid",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.tenancy", BadValue: ""},
			},
		}, {
			name: "missing_user_ocid",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.user", BadValue: ""},
			},
		}, {
			name: "missing_key_file",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.key", BadValue: ""},
			},
		}, {
			name: "missing_fingerprint",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					VcnOCID:         "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.fingerprint", BadValue: ""},
			},
		}, {
			name: "missing_vcn",
			in: &Config{
				Auth: AuthConfig{
					Region:          "us-phoenix-1",
					RegionKey:       "phx",
					CompartmentOCID: "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					TenancyOCID:     "ocid1.tennancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UserOCID:        "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					PrivateKey:      "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint:     "d4:1d:8c:d9:8f:00:b2:04:e9:80:09:98:ec:f8:42:7e",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.vcn", BadValue: ""},
			},
		}, {
			name: "valid with instance principals enabled",
			in: &Config{
				UseInstancePrincipals: true,
				Auth: AuthConfig{
					VcnOCID: "ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			},
			errs: field.ErrorList{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConfig(tt.in)
			if !reflect.DeepEqual(result, tt.errs) {
				t.Errorf("ValidateConfig(%+v)\n=> %+v\nExpected: %+v", tt.in, result, tt.errs)
			}
		})
	}
}
