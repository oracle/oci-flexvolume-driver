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
	"fmt"
	"log"
	"strings"
	"time"

	baremetal "github.com/oracle/bmcs-go-sdk"
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
	FindVolumeAttachment(vID string) (*baremetal.VolumeAttachment, error)

	// WaitForVolumeAttached polls waiting for a OCI block volume to be in the
	// ATTACHED state.
	WaitForVolumeAttached(id string) (*baremetal.VolumeAttachment, error)

	// GetInstanceByNodeName retrieves the baremetal.Instance corresponding or
	// a SearchError if no instance matching the node name is found.
	GetInstanceByNodeName(name string) (*baremetal.Instance, error)

	// GetConfig returns the Config associated with the OCI API client.
	GetConfig() *Config

	// AttachVolume attaches a block storage volume to the specified instance.
	// See https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/VolumeAttachment/AttachVolume
	AttachVolume(
		attachmentType,
		instanceID,
		volumeID string,
		opts *baremetal.CreateOptions,
	) (*baremetal.VolumeAttachment, error)

	// DetachVolume detaches a storage volume from the specified instance.
	// See: https://docs.us-phoenix-1.oraclecloud.com/api/#/en/iaas/20160918/Volume/DetachVolume
	DetachVolume(id string, opts *baremetal.IfMatchOptions) error
}

// client extends a barmetal.Client.
type client struct {
	*baremetal.Client
	config *Config
}

// New initialises a OCI API client from a config file.
func New(configPath string) (Interface, error) {
	config, err := ConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	baseClient, err := baremetal.NewClient(
		config.UserID,
		config.TenancyID,
		config.Fingerprint,
		baremetal.PrivateKeyFilePath(config.PrivateKeyFile),
		baremetal.Region(config.Region))
	if err != nil {
		return nil, err
	}

	return &client{Client: baseClient, config: config}, nil
}

// WaitForVolumeAttached polls waiting for a OCI block volume to be in the
// ATTACHED state.
func (c *client) WaitForVolumeAttached(id string) (*baremetal.VolumeAttachment, error) {
	// TODO: Replace with "k8s.io/apimachinery/pkg/util/wait".
	for i := 0; i < ociMaxRetries; i++ {
		at, err := c.GetVolumeAttachment(id)
		if err != nil {
			return nil, err
		}

		switch at.State {
		case baremetal.ResourceAttaching:
			time.Sleep(ociWaitDuration)
		case baremetal.ResourceAttached:
			return at, nil
		default:
			return nil, fmt.Errorf("unexpected state %q while wating for volume attach", at.State)
		}
	}
	return nil, fmt.Errorf("maximum number of retries (%d) exceeed attaching volume", ociMaxRetries)
}

// FindVolumeAttachment searches for a volume attachment in either the state of
// ATTACHING or ATTACHED and returns the first volume attachment found.
func (c *client) FindVolumeAttachment(vID string) (*baremetal.VolumeAttachment, error) {
	opts := &baremetal.ListVolumeAttachmentsOptions{VolumeID: vID}

	for {
		r, err := c.ListVolumeAttachments(c.config.CompartmentID, opts)
		if err != nil {
			return nil, err
		}

		for _, attachment := range r.VolumeAttachments {
			if attachment.State == baremetal.ResourceAttaching ||
				attachment.State == baremetal.ResourceAttached {
				return &attachment, nil
			}
		}

		if hasNextPage := SetNextPageOption(r.NextPage, &opts.ListOptions.PageListOptions); !hasNextPage {
			break
		}
	}

	return nil, fmt.Errorf("failed to find volume attachment for %q", vID)
}

// findInstanceByNodeNameIsVnic try to find the BM Instance
// // it makes the assumption that he nodename has to be resolvable
// https://kubernetes.io/docs/concepts/architecture/nodes/#management
// So if the displayname doesn't match the nodename then
// 1) get the IP of the node name doing a reverse lookup and see if we can find it.
// I'm leaving the DNS lookup till later as the options below fix the OKE issue
// 2) see if the nodename is equal to the hostname label
// 3) see if the nodename is an ip
func (c *client) findInstanceByNodeNameIsVnic(nodeName string) (*baremetal.Instance, error) {
	var running []baremetal.Instance
	opts := &baremetal.ListVnicAttachmentsOptions{}
	for {
		vnicAttachments, err := c.ListVnicAttachments(c.config.CompartmentID, opts)
		if err != nil {
			return nil, err
		}
		for _, attachment := range vnicAttachments.Attachments {
			if attachment.State == baremetal.ResourceAttached {
				vnic, err := c.GetVnic(attachment.VnicID)
				if err != nil {
					log.Printf("Unable to get vnic for attachment:%s", attachment.ID)
				}

				if vnic.PublicIPAddress == nodeName ||
					(vnic.HostnameLabel != "" && strings.HasPrefix(nodeName, vnic.HostnameLabel)) {
					instance, err := c.GetInstance(attachment.InstanceID)
					if err != nil {
						log.Printf("Error getting instance for attachment:%s", attachment.InstanceID)
						return nil, err
					}
					if instance.State == baremetal.ResourceRunning {
						log.Printf("Adding instace %#v due to vnic %#v matching %s", *instance, *vnic, nodeName)
						running = append(running, *instance)
					}
				}
			}
		}
		if hasNextPage := SetNextPageOption(vnicAttachments.NextPage, &opts.ListOptions.PageListOptions); !hasNextPage {
			break
		}
	}
	if len(running) != 1 {
		return nil, fmt.Errorf("expected one instance vnic ip/hostname '%s' but got %d", nodeName, len(running))
	}

	return &running[0], nil
}

func (c *client) findInstanceByNodeNameIsDisplayName(nodeName string) (*baremetal.Instance, error) {
	opts := &baremetal.ListInstancesOptions{
		DisplayNameListOptions: baremetal.DisplayNameListOptions{
			DisplayName: nodeName,
		},
	}

	var running []baremetal.Instance
	for {
		r, err := c.ListInstances(c.config.CompartmentID, opts)
		if err != nil {
			return nil, err
		}

		for _, i := range r.Instances {
			if i.State == baremetal.ResourceRunning {
				running = append(running, i)
			}
		}

		if hasNexPage := SetNextPageOption(r.NextPage, &opts.ListOptions.PageListOptions); !hasNexPage {
			break
		}
	}

	if len(running) != 1 {
		return nil, fmt.Errorf("expected one instance with display name %q but got %d", nodeName, len(running))
	}

	return &running[0], nil
}

// GetInstanceByNodeName retrieves the baremetal.Instance corresponding or a
// SearchError if no instance matching the node name is found.
func (c *client) GetInstanceByNodeName(nodeName string) (*baremetal.Instance, error) {
	instance, err := c.findInstanceByNodeNameIsDisplayName(nodeName)
	if err != nil {
		log.Printf("Unable to find OCI instance by displayname trying hostname/public ip")
		instance, err = c.findInstanceByNodeNameIsVnic(nodeName)
		if err != nil {
			log.Printf("Unable to find OCI instance by hostname/displayname")
		}
	}
	return instance, err
}

// GetConfig returns the Config associated with the OCI API client.
func (c *client) GetConfig() *Config {
	return c.config
}
