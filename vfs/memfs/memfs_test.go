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

//go:build !datarace

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
	//  Tests that memfs.MemFS struct implements avfs.VFS interface.
	_ avfs.VFS = &memfs.MemFS{}

	//  Tests that memfs.MemFile struct implements avfs.File interface.
	_ avfs.File = &memfs.MemFile{}

	//  Tests that memfs.MemInfo struct implements fs.DirEntry interface.
	_ fs.DirEntry = &memfs.MemInfo{}

	//  Tests that memfs.MemInfo struct implements fs.FileInfo interface.
	_ fs.FileInfo = &memfs.MemInfo{}

	//  Tests that memfs.MemInfo struct implements avfs.SysStater interface.
	_ avfs.SysStater = &memfs.MemInfo{}
)

func TestMemFS(t *testing.T) {
	vfs := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs(), memfs.WithOSType(avfs.OSUtils.OSType()))

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestMemFSWithNoIdm(t *testing.T) {
	vfs := memfs.New(memfs.WithMainDirs(), memfs.WithOSType(avfs.OSUtils.OSType()))

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestMemFSOptionUser(t *testing.T) {
	idm := memidm.New()

	groupName := "aGroup"
	_, err := idm.GroupAdd(groupName)
	test.CheckNoError(t, "GroupAdd "+groupName, err)

	userName := "aUser"
	u, err := idm.UserAdd(userName, groupName)
	test.CheckNoError(t, "UserAdd "+userName, err)

	vfs := memfs.New(memfs.WithUser(u))

	dir := "test"
	err = vfs.Mkdir(dir, avfs.DefaultDirPerm)
	test.CheckNoError(t, "Mkdir "+dir, err)

	info, err := vfs.Stat(dir)
	test.CheckNoError(t, "Stat "+dir, err)

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

	vfs = memfs.New(memfs.WithName(wantName))

	name := vfs.Name()
	if name != wantName {
		t.Errorf("New : want name to be %s, got %s", wantName, vfs.Name())
	}
}

func TestMemFSNilPtrFile(t *testing.T) {
	f := (*memfs.MemFile)(nil)

	test.FileNilPtr(t, f)
}

func TestMemFSConfig(t *testing.T) {
	vfs := memfs.New()

	wantFeatures := avfs.FeatBasicFs | avfs.FeatHardlink | avfs.FeatSymlink
	if vfs.OSType() == avfs.OsLinux {
		wantFeatures |= avfs.FeatChroot
	}

	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	vfs = memfs.New(memfs.WithIdm(memidm.New()))

	wantFeatures = avfs.FeatBasicFs | avfs.FeatHardlink | avfs.FeatIdentityMgr | avfs.FeatSymlink
	if vfs.OSType() == avfs.OsLinux {
		wantFeatures |= avfs.FeatChroot
	}

	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	ost := vfs.OSType()
	if ost != avfs.OSUtils.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.OSUtils.OSType(), ost)
	}
}

func BenchmarkMemFSAll(b *testing.B) {
	vfs := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())

	sfs := test.NewSuiteFS(b, vfs)
	sfs.BenchAll(b)
}
