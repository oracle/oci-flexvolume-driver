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
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/core"
	"github.com/stretchr/testify/assert"
)

func TempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
}

func TestCache(t *testing.T) {
	cacheName := TempFileName("/tmp", "TestCache.json")
	t.Logf("Testcache: %s", cacheName)
	cache, err := Open(cacheName)
	if err != nil {
		t.Error(err)
	}
	defer cache.Close()

	id := "test"
	var testVnic = core.Vnic{Id: &id}
	cache.SetVnic("test", &testVnic)
	value, ok := cache.GetVnic("test")
	if !ok {
		t.Error("test not found")
	}
	assert.Equal(t, *value, testVnic, "Key not equal to test")
	err = cache.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestCacheMultipleClose(t *testing.T) {
	cacheName := TempFileName("/tmp", "TestCache.json")
	t.Logf("Testcache: %s", cacheName)
	cache, err := Open(cacheName)
	if err != nil {
		t.Error(err)
	}

	err = cache.Close()
	if err != nil {
		t.Error(err)
	}
	err = cache.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestCacheLoadSave(t *testing.T) {
	cacheName := TempFileName("/tmp", "TestCache.json")
	t.Logf("Testcache: %s", cacheName)
	firstCache, err := Open(cacheName)
	if err != nil {
		t.Error(err)
	}
	defer firstCache.Close()
	id := "test"
	var testVnic = core.Vnic{Id: &id}
	firstCache.SetVnic("test", &testVnic)
	value, ok := firstCache.GetVnic("test")
	if !ok {
		t.Error("test not found")
	}
	assert.Equal(t, *value, testVnic, "Key not equal to test")
	err = firstCache.Close()
	if err != nil {
		t.Error(err)
	}
	otherCache, err := Open(cacheName)
	if err != nil {
		t.Error(err)
	}
	value, ok = otherCache.GetVnic("test")
	if !ok {
		t.Error("test not found")
	}

	if !reflect.DeepEqual(*value, testVnic) {
		t.Error("Key not equal to test")
	}
	err = otherCache.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestCacheParallel(t *testing.T) {
	cacheName := TempFileName("/tmp", "TestCache.json")
	t.Logf("Testcache: %s", cacheName)
	use := func() {
		cache, err := Open(cacheName)
		if err != nil {
			t.Error(err)
		}
		defer cache.Close()
		id := "test"
		var testVnic = core.Vnic{Id: &id}
		cache.SetVnic("test", &testVnic)
		value, ok := cache.GetVnic("test")
		if !ok {
			t.Error("test not found")
		}

		assert.Equal(t, *value, testVnic, "Key not equal to test")
		err = cache.Close()
		if err != nil {
			t.Error(err)
		}
	}

	for i := 0; i < 4000; i++ {
		go use()
	}
}
