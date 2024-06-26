//
//  Copyright 2020 The AVFS authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//  	http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

//go:build !avfs_race

package rofs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/rofs"
)

var (
	// Tests that rofs.RoFS struct implements avfs.VFS interface.
	_ avfs.VFS = &rofs.RoFS{}

	// Tests that rofs.RoFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &rofs.RoFS{}

	// Tests that rofs.RoFile struct implements avfs.File interface.
	_ avfs.File = &rofs.RoFile{}
)

func initTest(t *testing.T) *test.Suite {
	vfsSetup := memfs.New()
	vfs := rofs.New(vfsSetup)

	ts := test.NewSuiteFS(t, vfsSetup, vfs)

	return ts
}

func TestRoFS(t *testing.T) {
	ts := initTest(t)
	ts.TestVFSAll(t)
}

func TestRoFSConfig(t *testing.T) {
	vfsWrite := memfs.New()
	vfs := rofs.New(vfsWrite)

	wantFeatures := vfs.Features()&^avfs.FeatIdentityMgr | avfs.FeatReadOnly
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	osType := vfs.OSType()
	if osType != vfsWrite.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsWrite.OSType(), osType)
	}
}
