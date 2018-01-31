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

package cache

import (
	"encoding/json"
	"log"
	"os"
	"syscall"

	"github.com/oracle/oci-go-sdk/core"
)

type OCICache struct {
	vnics    map[string]core.Vnic
	file     *os.File
	filename string
}

// Open opens the cache and returns the cache handle
func Open(filename string) (*OCICache, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Failed to open cache: %v", err)
		return nil, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		log.Printf("Failed to lock cache: %v", err)
		return nil, err
	}
	var vnicCache = map[string]core.Vnic{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&vnicCache)
	if err != nil {
		log.Printf("Failed to decode cache: %v", err)
	}
	return &OCICache{file: file, vnics: vnicCache, filename: filename}, nil
}

// GetVnic looks up the vnic id in the cache
func (nc *OCICache) GetVnic(id string) (*core.Vnic, bool) {
	value, ok := nc.vnics[id]
	return &value, ok
}

// SetVnic adds a vnic to the cache
func (nc *OCICache) SetVnic(id string, value *core.Vnic) {
	nc.vnics[id] = *value
}

// Close closes the vnic cache saving to disk and unlocking the file
func (nc *OCICache) Close() error {
	if nc.file != nil {
		defer func() {
			nc.file.Close()
			nc.file = nil

		}()
		_, err := nc.file.Seek(0, 0)
		if err != nil {
			log.Printf("Error marshalling OCICache to JSON: %v", err)
			return err
		}

		encoder := json.NewEncoder(nc.file)
		err = encoder.Encode(&nc.vnics)
		if err != nil {
			log.Printf("Error marshalling OCICache to JSON: %v", err)
			return err
		}

		if err := syscall.Flock(int(nc.file.Fd()), syscall.LOCK_UN); err != nil {
			log.Printf("Error marshalling OCICache to JSON: %v", err)
			return err
		}
	}
	return nil
}
