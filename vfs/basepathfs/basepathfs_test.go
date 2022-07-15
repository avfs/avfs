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

package basepathfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/basepathfs"
	"github.com/avfs/avfs/vfs/memfs"
)

var (
	// Tests that basepathfs.BasePathFS struct implements avfs.VFS interface.
	_ avfs.VFS = &basepathfs.BasePathFS{}

	// Tests that basepathfs.BasePathFile struct implements avfs.File interface.
	_ avfs.File = &basepathfs.BasePathFile{}
)

func initFS(tb testing.TB) *basepathfs.BasePathFS {
	baseFs := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	basePath := baseFs.FromUnixPath(avfs.DefaultVolume, "/base/path")

	err := baseFs.MkdirAll(basePath, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %v", basePath, err)
	}

	vfs := basepathfs.New(baseFs, basePath)

	return vfs
}

func initTest(t *testing.T) *test.SuiteFS {
	vfs := initFS(t)
	sfs := test.NewSuiteFS(t, vfs)

	return sfs
}

func TestBasePathFS(t *testing.T) {
	sfs := initTest(t)
	sfs.TestAll(t)
}

// TestBasePathFsOptions tests BasePathFS configuration options.
func TestBasePathFSOptions(t *testing.T) {
	vfs := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	nonExistingDir := vfs.FromUnixPath(avfs.DefaultVolume, "/non/existing/dir")

	test.CheckPanic(t, "", func() {
		_ = basepathfs.New(vfs, nonExistingDir)
	})

	existingFile := vfs.Join(vfs.TempDir(), "existing")

	err := vfs.WriteFile(existingFile, []byte{}, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	test.CheckPanic(t, "", func() {
		_ = basepathfs.New(vfs, existingFile)
	})
}

func TestBasePathFSFeatures(t *testing.T) {
	vfs := basepathfs.New(memfs.New(), "/")
	if vfs.HasFeature(avfs.FeatSymlink) {
		t.Errorf("Features : want FeatSymlink missing, got present")
	}

	if vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr missing, got present")
	}

	mfs := memfs.New(memfs.WithIdm(memidm.New()))

	vfs = basepathfs.New(mfs, "/")
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

func TestBasepathFSOSType(t *testing.T) {
	vfsBase := memfs.New(memfs.WithMainDirs())
	vfs := basepathfs.New(vfsBase, vfsBase.TempDir())

	osType := vfs.OSType()
	if osType != vfsBase.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsBase.OSType(), osType)
	}
}
