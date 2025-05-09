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

package memfs_test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

var (
	// Tests that memfs.MemFS struct implements avfs.VFS interface.
	_ avfs.VFS = &memfs.MemFS{}

	// Tests that memfs.MemFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &memfs.MemFS{}

	// Tests that memfs.MemFile struct implements avfs.File interface.
	_ avfs.File = &memfs.MemFile{}

	// Tests that memfs.MemInfo struct implements fs.DirEntry interface.
	_ fs.DirEntry = &memfs.MemInfo{}

	// Tests that memfs.MemInfo struct implements fs.FileInfo interface.
	_ fs.FileInfo = &memfs.MemInfo{}

	// Tests that memfs.MemInfo struct implements avfs.SysStater interface.
	_ avfs.SysStater = &memfs.MemInfo{}

	// Tests that memfs.MemIOFS struct implements avfs.IOFS interface.
	_ avfs.IOFS = &memfs.MemIOFS{}
)

func TestMemFS(t *testing.T) {
	vfs := memfs.New()

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestMemFSWithNoIdm(t *testing.T) {
	vfs := memfs.NewWithOptions(&memfs.Options{Idm: avfs.NotImplementedIdm})

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestMemFSOptionUser(t *testing.T) {
	idm := memidm.New()

	groupName := "aGroup"
	_, err := idm.AddGroup(groupName)
	test.RequireNoError(t, err, "AddGroup %s", groupName)

	userName := "aUser"
	u, err := idm.AddUser(userName, groupName)
	test.RequireNoError(t, err, "AddUser %s", userName)

	vfs := memfs.NewWithOptions(&memfs.Options{User: u})

	dir := "test"
	err = vfs.Mkdir(dir, avfs.DefaultDirPerm)
	test.RequireNoError(t, err, "Mkdir %s", dir)

	info, err := vfs.Stat(dir)
	test.RequireNoError(t, err, "Stat %s", dir)

	sst := vfs.ToSysStat(info)
	if sst.Uid() != u.Uid() {
		t.Errorf("want Uid to be %d, got %d", u.Uid(), sst.Uid())
	}

	if sst.Gid() != u.Gid() {
		t.Errorf("want Uid to be %d, got %d", u.Gid(), sst.Gid())
	}
}

// TestMemFsOptionName tests MemFS initialization with or without option name (WithName()).
func TestMemFSOptionName(t *testing.T) {
	const wantName = "whatever"

	vfs := memfs.New()
	if vfs.Name() != "" {
		t.Errorf("New : want name to be '', got %s", vfs.Name())
	}

	vfs = memfs.NewWithOptions(&memfs.Options{Name: wantName})

	name := vfs.Name()
	if name != wantName {
		t.Errorf("New : want name to be %s, got %s", wantName, vfs.Name())
	}
}

func TestMemFSConfig(t *testing.T) {
	vfs := memfs.New()

	wantFeatures := avfs.FeatHardlink | avfs.FeatSubFS | avfs.FeatSymlink | avfs.FeatIdentityMgr |
		avfs.BuildFeatures()
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	osType := vfs.OSType()
	if osType != avfs.CurrentOSType() {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.CurrentOSType(), osType)
	}
}

func BenchmarkMemFSAll(b *testing.B) {
	vfs := memfs.New()

	ts := test.NewSuiteFS(b, vfs, vfs)
	ts.BenchAll(b)
}
