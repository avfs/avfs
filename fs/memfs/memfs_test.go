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

package memfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

var (
	// memfs.MemFs struct implements avfs.MemFs interface.
	_ avfs.VFS = &memfs.MemFs{}

	// memfs.MemFile struct implements avfs.File interface.
	_ avfs.File = &memfs.MemFile{}
)

func initTest(t *testing.T) *test.SuiteFs {
	vfsRoot, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFs(t, vfsRoot)

	return sfs
}

func TestMemFs(t *testing.T) {
	sfs := initTest(t)
	sfs.All()
}

func TestMemFsPerm(t *testing.T) {
	sfs := initTest(t)
	sfs.Perm()
}

func TestMemFsOptionError(t *testing.T) {
	_, err := memfs.New(memfs.WithIdm(dummyidm.New()))
	if err != avfs.ErrPermDenied {
		t.Errorf("New : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}
}

// TestMemFsOptionName tests MemFs initialization with or without option name (WithName()).
func TestMemFsOptionName(t *testing.T) {
	const wantName = "whatever"

	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	if vfs.Name() != "" {
		t.Errorf("New : want name to be '', got %s", vfs.Name())
	}

	vfs, err = memfs.New(memfs.WithName(wantName))
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	name := vfs.Name()
	if name != wantName {
		t.Errorf("New : want name to be %s, got %s", wantName, vfs.Name())
	}
}

func TestNilPtrReceiver(t *testing.T) {
	f := (*memfs.MemFile)(nil)

	test.SuiteNilPtrFile(t, f)
}

func TestMemFsFeatures(t *testing.T) {
	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	if vfs.Features()&avfs.FeatIdentityMgr != 0 {
		t.Errorf("Features : want FeatIdentityMgr missing, got present")
	}

	vfs, err = memfs.New(memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	if vfs.Features()&avfs.FeatIdentityMgr == 0 {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

func TestMemFsOSType(t *testing.T) {
	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	ost := vfs.OSType()
	if ost != avfs.OsLinux {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.OsLinux, ost)
	}
}

func BenchmarkMemFsCreate(b *testing.B) {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		b.Fatalf("New : want error to be nil, got %v", err)
	}

	test.BenchmarkCreate(b, vfs)
}
