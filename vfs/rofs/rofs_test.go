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
	"io"
	"syscall"
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

	// Ensures that rofs.RoFile implements the syscall.Conn interface.
	_ syscall.Conn = &rofs.RoFile{}

	// Ensures that rofs.RoFile implements the io.ReaderFrom interface.
	_ io.ReaderFrom = &rofs.RoFile{}

	// Ensures that rofs.RoFile implements the io.WriterTo interface.
	_ io.WriterTo = &rofs.RoFile{}
)

func initTest(t *testing.T) *test.Suite {
	vfsSetup, err := memfs.New()
	if err != nil {
		t.Fatal(err)
	}

	vfs, err := rofs.New(vfsSetup)
	if err != nil {
		t.Fatal(err)
	}

	ts := test.NewSuiteFS(t, vfsSetup, vfs)

	return ts
}

func TestRoFS(t *testing.T) {
	ts := initTest(t)
	ts.TestVFSAll(t)
}

func TestRoFSConfig(t *testing.T) {
	vfsWrite, err := memfs.New()
	if err != nil {
		t.Fatal(err)
	}

	vfs, err := rofs.New(vfsWrite)
	if err != nil {
		t.Fatal(err)
	}

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
