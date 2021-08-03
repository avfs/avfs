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

package orefafs_test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/orefafs"
)

var (
	// orefafs.OrefaFS struct implements avfs.VFS interface.
	_ avfs.VFS = &orefafs.OrefaFS{}

	// orefafs.OrefaFile struct implements avfs.File interface.
	_ avfs.File = &orefafs.OrefaFile{}

	// orefafs.OrefaInfo struct implements fs.DirEntry interface.
	_ fs.DirEntry = &orefafs.OrefaInfo{}

	// orefafs.OrefaInfo struct implements fs.FileInfo interface.
	_ fs.FileInfo = &orefafs.OrefaInfo{}

	// orefafs.OrefaInfo struct implements avfs.SysStater interface.
	_ avfs.SysStater = &orefafs.OrefaInfo{}
)

func initTest(tb testing.TB) *test.SuiteFS {
	vfs, err := orefafs.New(orefafs.WithMainDirs())
	if err != nil {
		tb.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(tb, vfs)

	return sfs
}

func TestOrefaFS(t *testing.T) {
	sfs := initTest(t)
	sfs.TestAll(t)
}

func TestOrefaFSNilPtrFile(t *testing.T) {
	f := (*orefafs.OrefaFile)(nil)

	test.FileNilPtr(t, f)
}

func TestOrefaFSConfig(t *testing.T) {
	vfs, err := orefafs.New()
	if err != nil {
		t.Fatalf("orefaFs.New : want error to be nil, got %v", err)
	}

	wantFeatures := avfs.FeatBasicFs | avfs.FeatHardlink
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %d, got %d", wantFeatures, vfs.Features())
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

func BenchmarkOrefaFSAll(b *testing.B) {
	sfs := initTest(b)
	sfs.BenchAll(b)
}
