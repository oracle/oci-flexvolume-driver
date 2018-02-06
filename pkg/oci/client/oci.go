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
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/client/cache"

	"github.com/oracle/oci-go-sdk/common"
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
	FindVolumeAttachment(volumeId string) (core.VolumeAttachment, error)

	// WaitForVolumeAttached polls waiting for a OCI block volume to be in the
	// ATTACHED state.
	WaitForVolumeAttached(volumeAttachmentId string) (core.VolumeAttachment, error)

	// GetInstanceByNodeName retrieves the oci.Instance corresponding or
	// a SearchError if no instance matching the node name is found.
	GetInstanceByNodeName(name string) (*core.Instance, error)

	// AttachVolume attaches a block storage volume to the specified instance.
	// See https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/VolumeAttachment/AttachVolume
	AttachVolume(instanceId, volumeId string) (core.VolumeAttachment, int, error)

	// DetachVolume detaches a storage volume from the specified instance.
	// See: https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/Volume/DetachVolume
	DetachVolume(volumeAttachmentId string) error

	// WaitForVolumeDetached polls waiting for a OCI block volume to be in the
	// DETACHED state.
	WaitForVolumeDetached(volumeAttachmentId string) error

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
	configProvider := common.NewRawConfigurationProvider(
		config.Auth.TenancyOCID,
		config.Auth.UserOCID,
		config.Auth.Region,
		config.Auth.Fingerprint,
		config.Auth.PrivateKey,
		&config.Auth.PrivateKeyPassphrase,
	)
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
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
func (c *client) WaitForVolumeAttached(volumeAttachmentId string) (core.VolumeAttachment, error) {
	// TODO: Replace with "k8s.io/apimachinery/pkg/util/wait".
	request := core.GetVolumeAttachmentRequest{
		VolumeAttachmentId: &volumeAttachmentId,
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
func (c *client) FindVolumeAttachment(volumeId string) (core.VolumeAttachment, error) {
	var page *string
	for {
		request := core.ListVolumeAttachmentsRequest{
			CompartmentId: &c.config.Auth.CompartmentOCID,
			Page:          page,
			VolumeId:      &volumeId,
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

	return nil, fmt.Errorf("failed to find volume attachment for %q", volumeId)
}

func (c *client) getAllSubnetsForVNC() (*[]core.Subnet, error) {
	var page *string
	subnetList := []core.Subnet{}
	for {
		request := core.ListSubnetsRequest{
			CompartmentId: &c.config.Auth.CompartmentOCID,
			VcnId:         &c.config.Auth.VcnOCID,
			Page:          page,
		}
		r, err := func() (core.ListSubnetsResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
			defer cancel()
			return c.network.ListSubnets(ctx, request)
		}()
		if err != nil {
			return nil, err
		}
		subnets := r.Items
		log.Printf("Subnets:%#v", subnets)
		subnetList = append(subnetList, subnets...)
		if page = r.OpcNextPage; r.OpcNextPage == nil {
			break
		}
	}
	return &subnetList, nil
}

func (c *client) isVnicAttachmentInSubnets(vnicAttachment *core.VnicAttachment, subnets *[]core.Subnet) bool {
	for _, subnet := range *subnets {
		if vnicAttachment.SubnetId == subnet.Id {
			return true
		}
	}
	return false
}

// findInstanceByNodeNameIsVnic try to find the BM Instance
// // it makes the assumption that he nodename has to be resolvable
// https://kubernetes.io/docs/concepts/architecture/nodes/#management
// So if the displayname doesn't match the nodename then
// 1) get the IP of the node name doing a reverse lookup and see if we can find it.
// I'm leaving the DNS lookup till later as the options below fix the OKE issue
// 2) see if the nodename is equal to the hostname label
// 3) see if the nodename is an ip
func (c *client) findInstanceByNodeNameIsVnic(cache *cache.OCICache, nodeName string) (*core.Instance, error) {
	subnets, err := c.getAllSubnetsForVNC()
	if err != nil {
		log.Printf("Error getting subnets for VCN: %s", c.config.Auth.VcnOCID)
		return nil, err
	}

	var running []core.Instance
	var page *string
	for {
		vnicAttachmentsRequest := core.ListVnicAttachmentsRequest{
			CompartmentId: &c.config.Auth.CompartmentOCID,
			Page:          page,
		}
		vnicAttachments, err := func() (core.ListVnicAttachmentsResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.compute.ListVnicAttachments(ctx, vnicAttachmentsRequest)
		}()
		if err != nil {
			return nil, err
		}
		for _, attachment := range vnicAttachments.Items {
			if !c.isVnicAttachmentInSubnets(&attachment, subnets) {
				continue
			}
			if attachment.LifecycleState == core.VnicAttachmentLifecycleStateAttached {
				vnic, ok := cache.GetVnic(*attachment.VnicId)
				if !ok {
					vnicRequest := core.GetVnicRequest{
						VnicId: attachment.VnicId,
					}
					vnicResponse, err := func() (core.GetVnicResponse, error) {
						ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
						defer cancel()
						return c.network.GetVnic(ctx, vnicRequest)
					}()
					if err != nil {
						log.Printf("Error getting Vnic for attachment: %s(%v)", *attachment.Id, err)
						continue
					}
					vnic = &vnicResponse.Vnic
					cache.SetVnic(*attachment.VnicId, vnic)
				}
				if *vnic.PublicIp == nodeName ||
					(*vnic.HostnameLabel != "" && strings.HasPrefix(nodeName, *vnic.HostnameLabel)) {
					instanceRequest := core.GetInstanceRequest{
						InstanceId: attachment.InstanceId,
					}
					instanceResponse, err := func() (core.GetInstanceResponse, error) {
						ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
						defer cancel()
						return c.compute.GetInstance(ctx, instanceRequest)
					}()
					if err != nil {
						log.Printf("Error getting instance for attachment: %s", *attachment.InstanceId)
						return nil, err
					}
					instance := instanceResponse.Instance
					if instance.LifecycleState == core.InstanceLifecycleStateRunning {
						running = append(running, instance)
					}
				}
			}
		}
		if page = vnicAttachments.OpcNextPage; vnicAttachments.OpcNextPage == nil {
			break
		}
	}
	if len(running) != 1 {
		return nil, fmt.Errorf("expected one instance vnic ip/hostname '%s' but got %d", nodeName, len(running))
	}

	return &running[0], nil
}

func (c *client) findInstanceByNodeNameIsDisplayName(nodeName string) (*core.Instance, error) {
	var running []core.Instance
	var page *string
	for {
		listInstancesRequest := core.ListInstancesRequest{
			CompartmentId: &c.config.Auth.CompartmentOCID,
			DisplayName:   &nodeName,
			Page:          page,
		}
		r, err := func() (core.ListInstancesResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.compute.ListInstances(ctx, listInstancesRequest)
		}()
		if err != nil {
			return nil, err
		}

		for _, i := range r.Items {
			if i.LifecycleState == core.InstanceLifecycleStateRunning {
				running = append(running, i)
			}
		}

		if page = r.OpcNextPage; r.OpcNextPage == nil {
			break
		}
	}

	if len(running) != 1 {
		return nil, fmt.Errorf("expected one instance with display name %q but got %d", nodeName, len(running))
	}

	return &running[0], nil
}

// GetDriverDirectory gets the path for the flexvolume driver either from the
// env or default.
func getCacheDirectory() string {
	path := os.Getenv("OCI_FLEXD_CACHE_DIRECTORY")
	if path != "" {
		return path
	}

	path = os.Getenv("OCI_FLEXD_DRIVER_DIRECTORY")
	if path != "" {
		return path
	}

	return "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci"
}

// GetInstanceByNodeName retrieves the corresponding core.Instance or a
// SearchError if no instance matching the node name is found.
func (c *client) GetInstanceByNodeName(nodeName string) (*core.Instance, error) {
	ociCache, err := cache.Open(fmt.Sprintf("%s/%s", getCacheDirectory(), "nodenamecache.json"))
	if err != nil {
		return nil, err
	}
	defer ociCache.Close()

	// Cache lookup failed so time to refill the cache
	instance, err := c.findInstanceByNodeNameIsDisplayName(nodeName)
	if err != nil {
		log.Printf("Unable to find OCI instance by displayname trying hostname/public ip")
		instance, err = c.findInstanceByNodeNameIsVnic(ociCache, nodeName)
		if err != nil {
			log.Printf("Unable to find OCI instance by hostname/displayname")
		}
	}
	return instance, err
}

// AttachVolume attaches a block storage volume to the specified instance.
func (c *client) AttachVolume(instanceId, volumeId string) (core.VolumeAttachment, int, error) {
	request := core.AttachVolumeRequest{
		AttachVolumeDetails: core.AttachIScsiVolumeDetails{
			InstanceId: &instanceId,
			VolumeId:   &volumeId,
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
func (c *client) DetachVolume(volumeAttachmentId string) error {
	request := core.DetachVolumeRequest{
		VolumeAttachmentId: &volumeAttachmentId,
	}
	err := func() error {
		ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
		defer cancel()
		return c.compute.DetachVolume(ctx, request)
	}()
	if err != nil {
		return err
	}
	return nil
}

// WaitForVolumeDetached polls waiting for a OCI block volume to be in the
// DETACHED state.
func (c *client) WaitForVolumeDetached(volumeAttachmentId string) error {
	// TODO: Replace with "k8s.io/apimachinery/pkg/util/wait".
	request := core.GetVolumeAttachmentRequest{
		VolumeAttachmentId: &volumeAttachmentId,
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
