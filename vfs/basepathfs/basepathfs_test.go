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

// +build !datarace

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
	// basepathfs.BasePathFS struct implements avfs.VFS interface.
	_ avfs.VFS = &basepathfs.BasePathFS{}

	// basepathfs.BasePathFile struct implements avfs.File interface.
	_ avfs.File = &basepathfs.BasePathFile{}
)

func initFS(tb testing.TB) *basepathfs.BasePathFS {
	const basePath = "/base/path"

	baseFs, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		tb.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = baseFs.MkdirAll(basePath, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %v", basePath, err)
	}

	vfs, err := basepathfs.New(baseFs, basePath)
	if err != nil {
		tb.Fatalf("basepathfs.New : want error to be nil, got %v", err)
	}

	return vfs
}

func initTest(t *testing.T) *test.SuiteFS {
	vfs := initFS(t)
	sfs := test.NewSuiteFS(t, vfs)

	return sfs
}

func TestBasePathFS(t *testing.T) {
	sfs := initTest(t)
	sfs.All(t)
}

func TestBasePathFSPerm(t *testing.T) {
	sfs := initTest(t)
	sfs.Perm(t)
}

// TestBasePathFsOptions tests BasePathFS configuration options.
func TestBasePathFSOptions(t *testing.T) {
	const (
		nonExistingDir = "/non/existing/dir"
		existingFile   = "/tmp/existingFile"
	)

	vfs, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = vfs.WriteFile(existingFile, []byte{}, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	_, err = basepathfs.New(vfs, nonExistingDir)
	test.CheckPathError(t, "BasePath", "basepath", nonExistingDir, avfs.ErrNoSuchFileOrDir, err)

	_, err = basepathfs.New(vfs, existingFile)
	test.CheckPathError(t, "BasePath", "basepath", existingFile, avfs.ErrNotADirectory, err)
}

func TestBasePathFSFeatures(t *testing.T) {
	mfs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	if mfs.Features()&avfs.FeatSymlink == 0 {
		t.Errorf("Features : want FeatSymlink present, got missing")
	}

	vfs, err := basepathfs.New(mfs, "/")
	if err != nil {
		t.Fatalf("basepathfs.New : want error to be nil, got %v", err)
	}

	if vfs.Features()&avfs.FeatSymlink != 0 {
		t.Errorf("Features : want FeatSymlink missing, got present")
	}

	if vfs.Features()&avfs.FeatIdentityMgr != 0 {
		t.Errorf("Features : want FeatIdentityMgr missing, got present")
	}

	mfs, err = memfs.New(memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	vfs, err = basepathfs.New(mfs, "/")
	if err != nil {
		t.Fatalf("basepathfs.New : want error to be nil, got %v", err)
	}

	if vfs.Features()&avfs.FeatIdentityMgr == 0 {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

func TestBasepathFSOSType(t *testing.T) {
	vfsBase, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %v", err)
	}

	vfs, err := basepathfs.New(vfsBase, avfs.TmpDir)
	if err != nil {
		t.Fatalf("basepathfs.New : want err to be nil, got %v", err)
	}

	ost := vfs.OSType()
	if ost != vfsBase.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsBase.OSType(), ost)
	}
}
