//
//  Copyright 2021 The AVFS authors
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

package avfs_test

import (
	"math"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
)

var (
	// avfs.DummyFS struct implements avfs.VFS interface.
	_ avfs.VFS = &avfs.DummyFS{}

	// avfs.DummyFile struct implements avfs.File interface.
	_ avfs.File = &avfs.DummyFile{}

	// avfs.DummySysStat struct implements avfs.SysStater interface.
	_ avfs.SysStater = &avfs.DummySysStat{}
)

func TestDummyFS(t *testing.T) {
	vfs := avfs.NewDummyFS()

	sfs := test.NewSuiteFS(t, &vfs)
	sfs.TestAll(t)
}

func TestDummyFSConfig(t *testing.T) {
	vfs := avfs.NewDummyFS()

	if vfs.HasFeature(avfs.Feature(math.MaxUint64)) {
		t.Error("HasFeature : want HasFeature(whatever) to be false, got true")
	}

	if vfs.Features() != 0 {
		t.Errorf("Features : want Features to be 0, got %d", vfs.Features())
	}

	if vfs.OSType() != avfs.OsLinux {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.OsLinux, vfs.OSType())
	}
}

func TestDummyFSIdm(t *testing.T) {
	vfs := avfs.NewDummyFS()

	sIdm := test.NewSuiteIdm(t, &vfs)
	sIdm.TestAll(t)
}