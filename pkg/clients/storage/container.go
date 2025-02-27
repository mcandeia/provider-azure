/*
Copyright 2019 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"

	azure "github.com/crossplane-contrib/provider-azure/pkg/clients"
)

// ContainerOperations interface to perform operations on Container resources
type ContainerOperations interface {
	Create(ctx context.Context, publicAccessType azblob.PublicAccessType, metadata azblob.Metadata) error
	Update(ctx context.Context, publicAccessType azblob.PublicAccessType, metadata azblob.Metadata) error
	Get(ctx context.Context) (*azblob.PublicAccessType, azblob.Metadata, error)
	Delete(ctx context.Context) error
}

// ContainerHandle implements ContainerOperations
type ContainerHandle struct {
	azblob.ContainerURL
	PublicAccessType azblob.PublicAccessType
}

var _ ContainerOperations = &ContainerHandle{}

const blobFormatString = `https://%s.blob.core.windows.net`

// NewContainerHandle creates a new instance of ContainerHandle for given storage account and given container name
func NewContainerHandle(accountName, accountKey, containerName string) (*ContainerHandle, error) {
	c, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	p := azblob.NewPipeline(c, azblob.PipelineOptions{
		Telemetry: azblob.TelemetryOptions{Value: azure.UserAgent},
	})

	u, _ := url.Parse(fmt.Sprintf(blobFormatString, accountName))
	service := azblob.NewServiceURL(*u, p)

	return &ContainerHandle{
		ContainerURL: service.NewContainerURL(containerName),
	}, nil
}

// Create container resource
func (a *ContainerHandle) Create(ctx context.Context, publicAccessType azblob.PublicAccessType, metadata azblob.Metadata) error {
	_, err := a.ContainerURL.Create(ctx, azblob.Metadata{}, publicAccessType)
	return err
}

// Update container resource
func (a *ContainerHandle) Update(ctx context.Context, publicAccessType azblob.PublicAccessType, metadata azblob.Metadata) error {
	if _, err := a.ContainerURL.SetMetadata(ctx, metadata, azblob.ContainerAccessConditions{}); err != nil {
		return err
	}
	_, err := a.ContainerURL.SetAccessPolicy(ctx, publicAccessType, nil, azblob.ContainerAccessConditions{})
	return err
}

// Get resource information
func (a *ContainerHandle) Get(ctx context.Context) (*azblob.PublicAccessType, azblob.Metadata, error) {
	rs, err := a.ContainerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if err != nil {
		return nil, nil, err
	}
	publicAccess := rs.BlobPublicAccess()
	return &publicAccess, emtpyMetaToNil(rs.NewMetadata()), nil
}

// Delete deletes the named container.
func (a *ContainerHandle) Delete(ctx context.Context) error {
	_, err := a.ContainerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	return err
}

func emtpyMetaToNil(m azblob.Metadata) azblob.Metadata {
	if len(m) == 0 {
		return nil
	}
	return m
}

// IsNotFoundError tests for azblob not found error
func IsNotFoundError(err error) bool {
	storageErr, ok := err.(azblob.StorageError)
	if !ok {
		return false
	}

	return storageErr.Response().StatusCode == http.StatusNotFound // nolint: bodyclose
}
