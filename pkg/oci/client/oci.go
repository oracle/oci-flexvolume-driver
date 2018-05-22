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
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/oracle/oci-flexvolume-driver/pkg/oci/client/cache"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/filestorage"
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

	// GetInstanceByNodeName retrieves the oci.Instance corresponding or
	// a SearchError if no instance matching the node name is found.
	GetInstanceByNodeName(name string) (*core.Instance, error)

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

	// GetMountTargetForAD returns a mount target for a given AD
	GetMountTargetForAD(AvailabilityDomain string) (*filestorage.MountTarget, error)

	// GetFilesystem returns the filesystem for a given id
	GetFileSystem(ocid string) (*filestorage.FileSystem, error)

	// AttachFileSystemToMountTarget attaches the filesystem to the mount target
	AttachFileSystemToMountTarget(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) error

	// DetachFileSystemToMountTarget detaches the filesystem from the mount target.
	// trjl: TODO rename? -> DetachFileSystemFromMountTarget
	DetachFileSystemToMountTarget(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) error

	// GetMountTargetIPS gets the mount target private ip addresses
	GetMountTargetIPS(mountTarget *filestorage.MountTarget) ([]core.PrivateIp, error)

	// GetMountTargetIPs checks to see if the filesystem is attached to the mounttarget
	IsFileSystemAttached(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) (bool, error)
}

// client contains all the clients,config and the default context and timeout
type client struct {
	compute     *core.ComputeClient
	network     *core.VirtualNetworkClient
	filestorage *filestorage.FileStorageClient
	config      *Config
	ctx         context.Context
	timeout     time.Duration
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
		&config.Auth.Passphrase,
	)
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

	filesystemStorageClient, err := filestorage.NewFileStorageClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	return &client{
		compute:     &computeClient,
		network:     &virtualNetworkClient,
		filestorage: &filesystemStorageClient,
		config:      config,
		ctx:         context.Background(),
		timeout:     time.Minute}, nil
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
			CompartmentId: &c.config.Auth.CompartmentOCID,
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

		subnetList = append(subnetList, subnets...)
		if page = r.OpcNextPage; r.OpcNextPage == nil {
			break
		}
	}
	return &subnetList, nil
}

func (c *client) isVnicAttachmentInSubnets(vnicAttachment *core.VnicAttachment, subnets *[]core.Subnet) bool {
	for _, subnet := range *subnets {
		if *vnicAttachment.SubnetId == *subnet.Id {
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
	if len(*subnets) == 0 {
		return nil, fmt.Errorf("no subnets defined for VCN: %s", c.config.Auth.VcnOCID)
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
						log.Printf("Error getting Vnic for attachment: %s(%v)", attachment, err)
						continue
					}
					vnic = &vnicResponse.Vnic
					cache.SetVnic(*attachment.VnicId, vnic)
				}
				if (vnic.PublicIp != nil && *vnic.PublicIp == nodeName) ||
					(vnic.HostnameLabel != nil && (*vnic.HostnameLabel != "" && strings.HasPrefix(nodeName, *vnic.HostnameLabel))) {
					instanceRequest := core.GetInstanceRequest{
						InstanceId: attachment.InstanceId,
					}
					instanceResponse, err := func() (core.GetInstanceResponse, error) {
						ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
						defer cancel()
						return c.compute.GetInstance(ctx, instanceRequest)
					}()
					if err != nil {
						log.Printf("Error getting instance for attachment: %s", attachment)
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
	log.Printf("GetInstanceByNodeName:%s", nodeName)
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

func (c *client) getMountTargetOCIDForAD(AvailabilityDomain string) *string {
	if strings.HasSuffix(AvailabilityDomain, "AD-1") {
		return &c.config.Storage.MountTargetAd1OCID
	}
	if strings.HasSuffix(AvailabilityDomain, "AD-2") {
		return &c.config.Storage.MountTargetAd2OCID
	}
	if strings.HasSuffix(AvailabilityDomain, "AD-3") {
		return &c.config.Storage.MountTargetAd3OCID
	}
	return nil
}

func (c *client) GetMountTargetForAD(AvailabilityDomain string) (*filestorage.MountTarget, error) {
	mountTargetOCID := c.getMountTargetOCIDForAD(AvailabilityDomain)
	if mountTargetOCID == nil {
		return nil, fmt.Errorf("Unable to get mount target for AD:%s", AvailabilityDomain)
	}
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	response, err := c.filestorage.GetMountTarget(ctx, filestorage.GetMountTargetRequest{MountTargetId: mountTargetOCID})
	if err != nil {
		return nil, err
	}
	return &response.MountTarget, nil
}

func (c *client) GetFileSystem(ocid string) (*filestorage.FileSystem, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	response, err := c.filestorage.GetFileSystem(ctx,
		filestorage.GetFileSystemRequest{FileSystemId: common.String(ocid)})
	if err != nil {
		return nil, err
	}
	return &response.FileSystem, nil
}

func (c *client) listExports(fileSystem *filestorage.FileSystem,
	mountTarget *filestorage.MountTarget) (*[]filestorage.ExportSummary, error) {
	request := filestorage.ListExportsRequest{
		CompartmentId: fileSystem.CompartmentId,
		FileSystemId:  fileSystem.Id,
		ExportSetId:   mountTarget.ExportSetId,
	}

	var exports []filestorage.ExportSummary
	for {
		response, err := func() (filestorage.ListExportsResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.filestorage.ListExports(ctx, request)
		}()
		if err != nil {
			return nil, err
		}

		exports = append(exports, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		request.Page = response.OpcNextPage
	}
	return &exports, nil
}

func (c *client) findExport(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) (*filestorage.ExportSummary, error) {
	exports, err := c.listExports(fileSystem, mountTarget)
	if err != nil {
		return nil, err
	}

	for _, export := range *exports {
		if *export.Path == path {
			return &export, nil
		}
	}
	return nil, nil
}

func (c *client) IsFileSystemAttached(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) (bool, error) {
	exportSummary, err := c.findExport(fileSystem, mountTarget, path)
	if err != nil {
		log.Printf("Error in IsAttached findexports")
		return false, err
	}
	if exportSummary != nil {
		return true, nil
	}
	return false, nil
}

func (c *client) AttachFileSystemToMountTarget(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) error {
	exportSummary, err := c.findExport(fileSystem, mountTarget, path)
	if err != nil {
		return err
	}
	if exportSummary != nil {
		log.Printf("Found export %s", *exportSummary.Id)
		log.Printf("FileSystem:%s already mounted on MountTarget %s at %s", *fileSystem.Id, *mountTarget.Id, path)
		return nil
	}
	response, err := func() (filestorage.CreateExportResponse, error) {
		ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
		defer cancel()
		return c.filestorage.CreateExport(ctx,
			filestorage.CreateExportRequest{
				CreateExportDetails: filestorage.CreateExportDetails{
					ExportSetId:  mountTarget.ExportSetId,
					FileSystemId: fileSystem.Id,
					Path:         common.String(path),
				},
			})
	}()
	if err != nil {
		return err
	}
	log.Printf("Filesystem Exported %s at %s(%s) %s", *fileSystem.Id, *mountTarget.Id, path, *response.Export.Id)

	export := response.Export

	for {
		log.Printf("Export State:(%s)%s", *export.Id, export.LifecycleState)
		if export.LifecycleState == filestorage.ExportLifecycleStateActive {
			break
		}

		response, err := func() (filestorage.GetExportResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.filestorage.GetExport(ctx, filestorage.GetExportRequest{
				ExportId: export.Id,
			})
		}()
		if err != nil {
			return err
		}
		export = response.Export
		time.Sleep(time.Second * 1)
	}

	return nil
}

func (c *client) DetachFileSystemToMountTarget(fileSystem *filestorage.FileSystem, mountTarget *filestorage.MountTarget, path string) error {
	export, err := c.findExport(fileSystem, mountTarget, path)
	if err != nil {
		return err
	}
	if export == nil {
		log.Printf("FileSystem:%s not mounted on MountTarget %s at %s", *fileSystem.Id, *mountTarget.Id, path)
		return nil
	}
	log.Printf("Found export %s", *export.Id)

	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	_, err = c.filestorage.DeleteExport(ctx, filestorage.DeleteExportRequest{
		ExportId: export.Id,
	})
	if err != nil {
		return err
	}

	log.Printf("Deleted export %s", *export.Id)

	/*exportid := export.Id

	for {
		response, err := func() (filestorage.GetExportResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.filestorage.GetExport(ctx, filestorage.GetExportRequest{
				ExportId: exportid,
			})
		}()
		if err != nil {
			return err
		}
		log.Printf("Export State:(%s)%s", *export.Id, export.LifecycleState)
		if export.LifecycleState == filestorage.ExportSummaryLifecycleStateDeleted {
			break
		}
		time.Sleep(time.Second * 1)

		exportid = response.Export.Id

	}*/

	return nil
}

func (c *client) GetMountTargetIPS(mountTarget *filestorage.MountTarget) ([]core.PrivateIp, error) {
	var privateIps []core.PrivateIp
	for _, PrivateIPID := range mountTarget.PrivateIpIds {
		response, err := func() (core.GetPrivateIpResponse, error) {
			ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
			defer cancel()
			return c.network.GetPrivateIp(ctx, core.GetPrivateIpRequest{
				PrivateIpId: &PrivateIPID,
			})
		}()
		if err != nil {
			log.Printf("GetMountTargetIPS failed to get private IP for %s", PrivateIPID)
			return nil, err
		}
		privateIps = append(privateIps, response.PrivateIp)
	}
	return privateIps, nil
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
