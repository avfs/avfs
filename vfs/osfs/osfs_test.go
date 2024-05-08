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

package osfs_test

import (
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/osfs"
)

var (
	// Tests that osfs.OsFS struct implements avfs.VFS interface.
	_ avfs.VFS = &osfs.OsFS{}

	// Tests that osfs.OsFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &osfs.OsFS{}

	// Tests that os.File struct implements avfs.File interface.
	_ avfs.File = &os.File{}
)

func TestOsFS(t *testing.T) {
	vfs := osfs.New()

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestOsFSWithNoIdm(t *testing.T) {
	vfs := osfs.NewWithNoIdm()

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestOsFSNilPtrFile(t *testing.T) {
	f := (*os.File)(nil)

	test.FileNilPtr(t, f)
}

func TestOsFSConfig(t *testing.T) {
	vfs := osfs.New()

	wantFeatures := avfs.FeatHardlink | avfs.FeatRealFS | avfs.FeatSymlink
	if vfs.OSType() == avfs.OsLinux {
		wantFeatures |= avfs.FeatIdentityMgr
	}

	if !vfs.User().IsAdmin() && vfs.OSType() != avfs.OsWindows {
		wantFeatures |= avfs.FeatReadOnlyIdm
	}

	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	ostType := vfs.OSType()
	if ostType != avfs.CurrentOSType() {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.CurrentOSType(), ostType)
	}
}

func BenchmarkOsFSAll(b *testing.B) {
	vfs := osfs.New()

	ts := test.NewSuiteFS(b, vfs, vfs)
	ts.BenchAll(b)
}
