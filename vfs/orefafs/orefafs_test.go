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

package orefafs_test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/orefafs"
)

var (
	// Tests that orefafs.OrefaFS struct implements avfs.VFS interface.
	_ avfs.VFS = &orefafs.OrefaFS{}

	// Tests that orefafs.OrefaFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &orefafs.OrefaFS{}

	// Tests that orefafs.OrefaFile struct implements avfs.File interface.
	_ avfs.File = &orefafs.OrefaFile{}

	// Tests that orefafs.OrefaInfo struct implements fs.DirEntry interface.
	_ fs.DirEntry = &orefafs.OrefaInfo{}

	// Tests that orefafs.OrefaInfo struct implements fs.FileInfo interface.
	_ fs.FileInfo = &orefafs.OrefaInfo{}

	// Tests that orefafs.OrefaInfo struct implements avfs.SysStater interface.
	_ avfs.SysStater = &orefafs.OrefaInfo{}
)

func TestOrefaFS(t *testing.T) {
	vfs := orefafs.New()

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestOrefaFSConfig(t *testing.T) {
	vfs := orefafs.New()

	wantFeatures := avfs.FeatHardlink | avfs.BuildFeatures()
	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures.String(), vfs.Features())
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

func BenchmarkOrefaFSAll(b *testing.B) {
	vfs := orefafs.New()

	ts := test.NewSuiteFS(b, vfs, vfs)
	ts.BenchAll(b)
}
