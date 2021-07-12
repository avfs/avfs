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
// +build !datarace

package memfs_test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

var (
	// memfs.MemFS struct implements avfs.VFS interface.
	_ avfs.VFS = &memfs.MemFS{}

	// memfs.MemFile struct implements avfs.File interface.
	_ avfs.File = &memfs.MemFile{}

	// memfs.MemStat struct implements fs.DirEntry interface.
	_ fs.DirEntry = &memfs.MemStat{}

	// memfs.MemStat struct implements fs.FileInfo interface.
	_ fs.FileInfo = &memfs.MemStat{}
)

func TestMemFS(t *testing.T) {
	vfs, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestMemFSWithNoIdm(t *testing.T) {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestMemFSOptionError(t *testing.T) {
	_, err := memfs.New(memfs.WithIdm(dummyidm.New()))
	if err != avfs.ErrPermDenied {
		t.Errorf("New : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}
}

// TestMemFsOptionName tests MemFS initialization with or without option name (WithName()).
func TestMemFSOptionName(t *testing.T) {
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

func TestMemFSNilPtrFile(t *testing.T) {
	f := (*memfs.MemFile)(nil)

	test.FileNilPtr(t, f)
}

func TestMemFSConfig(t *testing.T) {
	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	wantFeatures := avfs.FeatBasicFs | avfs.FeatChroot | avfs.FeatHardlink | avfs.FeatSymlink
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	vfs, err = memfs.New(memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	wantFeatures = avfs.FeatBasicFs | avfs.FeatChroot | avfs.FeatHardlink | avfs.FeatIdentityMgr | avfs.FeatSymlink
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	ost := vfs.OSType()
	if ost != avfs.OsLinux {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.OsLinux, ost)
	}
}

func BenchmarkMemFSAll(b *testing.B) {
	vfs, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		b.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(b, vfs)
	sfs.BenchAll(b)
}
