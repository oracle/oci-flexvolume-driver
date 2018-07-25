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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/common/auth"
	"github.com/oracle/oci-go-sdk/core"
)

const (
	ociWaitDuration = 1 * time.Second
	ociMaxRetries   = 120
)

// Interface abstracts the OCI SDK and application specific convenience methods
// for interacting with the OCI API.
type Interface interface {
	// FindVolumeAttachment searches for a volume attachment in either the state
	// ATTACHING or ATTACHED and returns the first volume attachment found.
	FindVolumeAttachment(volumeID string) (core.VolumeAttachment, error)

	// WaitForVolumeAttached polls waiting for a OCI block volume to be in the
	// ATTACHED state.
	WaitForVolumeAttached(volumeAttachmentID string) (core.VolumeAttachment, error)

	// GetInstance retrieves the oci.Instance for a given OCID.
	GetInstance(id string) (*core.Instance, error)

	// AttachVolume attaches a block storage volume to the specified instance.
	// See https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/VolumeAttachment/AttachVolume
	AttachVolume(instanceID, volumeID string) (core.VolumeAttachment, int, error)

	// DetachVolume detaches a storage volume from the specified instance.
	// See: https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/Volume/DetachVolume
	DetachVolume(volumeAttachmentID string) error

	// WaitForVolumeDetached polls waiting for a OCI block volume to be in the
	// DETACHED state.
	WaitForVolumeDetached(volumeAttachmentID string) error

	// GetConfig returns the Config associated with the OCI API client.
	GetConfig() *Config
}

// client extends a barmetal.Client.
type client struct {
	compute *core.ComputeClient
	network *core.VirtualNetworkClient
	config  *Config
	ctx     context.Context
	timeout time.Duration
}

// New initialises a OCI API client from a config file.
func New(configPath string) (Interface, error) {
	config, err := ConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}
	var configProvider common.ConfigurationProvider
	if config.UseInstancePrincipals {
		cp, err := auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			return nil, err
		}
		configProvider = cp
	} else {
		configProvider = common.NewRawConfigurationProvider(
			config.Auth.TenancyOCID,
			config.Auth.UserOCID,
			config.Auth.Region,
			config.Auth.Fingerprint,
			config.Auth.PrivateKey,
			&config.Auth.Passphrase,
		)
	}
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	err = configureCustomTransport(&computeClient.BaseClient)
	if err != nil {
		return nil, err
	}

	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	err = configureCustomTransport(&virtualNetworkClient.BaseClient)
	if err != nil {
		return nil, err
	}

	return &client{
		compute: &computeClient,
		network: &virtualNetworkClient,
		config:  config,
		ctx:     context.Background(),
		timeout: time.Minute}, nil
}

// WaitForVolumeAttached polls waiting for a OCI block volume to be in the
// ATTACHED state.
func (c *client) WaitForVolumeAttached(volumeAttachmentID string) (core.VolumeAttachment, error) {
	// TODO: Replace with "k8s.io/apimachinery/pkg/util/wait".
	request := core.GetVolumeAttachmentRequest{
		VolumeAttachmentId: &volumeAttachmentID,
	}
	for i := 0; i < ociMaxRetries; i++ {
		r, err := func() (core.GetVolumeAttachmentResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.compute.GetVolumeAttachment(ctx, request)
		}()
		if err != nil {
			return nil, err
		}
		attachment := r.VolumeAttachment
		state := attachment.GetLifecycleState()
		switch state {
		case core.VolumeAttachmentLifecycleStateAttaching:
			time.Sleep(ociWaitDuration)
		case core.VolumeAttachmentLifecycleStateAttached:
			return attachment, nil
		default:
			return nil, fmt.Errorf("unexpected state %q while wating for volume attach", state)
		}
	}
	return nil, fmt.Errorf("maximum number of retries (%d) exceeed attaching volume", ociMaxRetries)
}

// FindVolumeAttachment searches for a volume attachment in either the state of
// ATTACHING or ATTACHED and returns the first volume attachment found.
func (c *client) FindVolumeAttachment(volumeID string) (core.VolumeAttachment, error) {
	var page *string

	for {
		request := core.ListVolumeAttachmentsRequest{
			CompartmentId: common.String(c.config.Auth.CompartmentOCID),
			Page:          page,
			VolumeId:      &volumeID,
		}

		r, err := func() (core.ListVolumeAttachmentsResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.compute.ListVolumeAttachments(ctx, request)
		}()
		if err != nil {
			return nil, err
		}

		for _, attachment := range r.Items {
			state := attachment.GetLifecycleState()
			if state == core.VolumeAttachmentLifecycleStateAttaching ||
				state == core.VolumeAttachmentLifecycleStateAttached {
				return attachment, nil
			}
		}

		if page = r.OpcNextPage; r.OpcNextPage == nil {
			break
		}
	}

	return nil, fmt.Errorf("failed to find volume attachment for %q", volumeID)
}

func (c *client) getVCNCompartment() (*string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()

	vcn, err := c.network.GetVcn(ctx, core.GetVcnRequest{VcnId: &c.config.Auth.VcnOCID})
	if err != nil {
		return nil, err
	}

	return vcn.CompartmentId, nil
}

// GetInstance retrieves the corresponding core.Instance by OCID.
func (c *client) GetInstance(id string) (*core.Instance, error) {
	resp, err := c.compute.GetInstance(c.ctx, core.GetInstanceRequest{
		InstanceId: &id,
	})

	if err != nil {
		return nil, err
	}

	return &resp.Instance, nil
}

// AttachVolume attaches a block storage volume to the specified instance.
func (c *client) AttachVolume(instanceID, volumeID string) (core.VolumeAttachment, int, error) {
	request := core.AttachVolumeRequest{
		AttachVolumeDetails: core.AttachIScsiVolumeDetails{
			InstanceId: &instanceID,
			VolumeId:   &volumeID,
		},
	}
	r, err := func() (core.AttachVolumeResponse, error) {
		ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
		defer cancel()
		return c.compute.AttachVolume(ctx, request)
	}()
	if err != nil {
		return nil, r.RawResponse.StatusCode, err
	}
	return r.VolumeAttachment, r.RawResponse.StatusCode, nil
}

// DetachVolume detaches a storage volume from the specified instance.
func (c *client) DetachVolume(volumeAttachmentID string) error {
	request := core.DetachVolumeRequest{
		VolumeAttachmentId: &volumeAttachmentID,
	}
	err := func() error {
		ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
		defer cancel()
		_, err := c.compute.DetachVolume(ctx, request)
		return err
	}()
	if err != nil {
		return err
	}
	return nil
}

// WaitForVolumeDetached polls waiting for a OCI block volume to be in the
// DETACHED state.
func (c *client) WaitForVolumeDetached(volumeAttachmentID string) error {
	// TODO: Replace with "k8s.io/apimachinery/pkg/util/wait".
	request := core.GetVolumeAttachmentRequest{
		VolumeAttachmentId: &volumeAttachmentID,
	}
	for i := 0; i < ociMaxRetries; i++ {
		r, err := func() (core.GetVolumeAttachmentResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.compute.GetVolumeAttachment(ctx, request)
		}()
		if err != nil {
			return err
		}
		attachment := r.VolumeAttachment
		state := attachment.GetLifecycleState()
		switch state {
		case core.VolumeAttachmentLifecycleStateDetaching:
			time.Sleep(ociWaitDuration)
		case core.VolumeAttachmentLifecycleStateDetached:
			return nil
		default:
			return fmt.Errorf("unexpected state %q while wating for volume detach", state)
		}
	}
	return fmt.Errorf("maximum number of retries (%d) exceeed detaching volume", ociMaxRetries)
}

// GetConfig returns the Config associated with the OCI API client.
func (c *client) GetConfig() *Config {
	return c.config
}

// configureCustomTransport customises the base client's transport to use
// the environment variable specified proxy and/or certificate.
func configureCustomTransport(baseClient *common.BaseClient) error {

	httpClient := baseClient.HTTPClient.(*http.Client)

	var transport *http.Transport
	if httpClient.Transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	} else {
		transport = httpClient.Transport.(*http.Transport)
	}

	ociProxy := os.Getenv("OCI_PROXY")
	if ociProxy != "" {
		proxyURL, err := url.Parse(ociProxy)
		if err != nil {
			return fmt.Errorf("failed to parse OCI proxy url: %s, err: %v", ociProxy, err)
		}
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyURL, nil
		}
	}

	trustedCACertPath := os.Getenv("TRUSTED_CA_CERT_PATH")
	if trustedCACertPath != "" {
		trustedCACert, err := ioutil.ReadFile(trustedCACertPath)
		if err != nil {
			return fmt.Errorf("failed to read root certificate: %s, err: %v", trustedCACertPath, err)
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(trustedCACert)
		if !ok {
			return fmt.Errorf("failed to parse root certificate: %s", trustedCACertPath)
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: caCertPool}
	}
	httpClient.Transport = transport
	return nil
}
