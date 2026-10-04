//
//  Copyright 2026 The AVFS authors
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

package ostestfs_test

import (
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/ostestfs"
)

var (
	// Ensures that osfs.OsFS implements avfs.VFS interface.
	_ avfs.VFS = &ostestfs.OsTestFS{}

	// Ensures that osfs.OsFS implements avfs.VFSBase interface.
	_ avfs.VFSBase = &ostestfs.OsTestFS{}

	// Ensures that os.File implements avfs.File interface.
	_ avfs.File = &os.File{}

	// Ensures that os.File implements the io.ReaderFrom interface.
	_ io.ReaderFrom = &os.File{}

	// Ensures that os.File implements the io.WriterTo interface.
	_ io.WriterTo = &os.File{}

	// Ensures that os.File implements the syscall.Conn interface.
	_ syscall.Conn = &os.File{}
)

func TestOsTestFS(t *testing.T) {
	vfs, err := ostestfs.New()
	if err != nil {
		t.Fatal(err)
	}

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestOsTestFSConfig(t *testing.T) {
	vfs, err := ostestfs.New()
	if err != nil {
		t.Fatal(err)
	}

	wantFeatures := avfs.FeatHardlink | avfs.FeatRealFS | avfs.FeatSymlink
	if vfs.OSType() != avfs.OsWindows {
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

func BenchmarkOsTestFSAll(b *testing.B) {
	vfs, err := ostestfs.New()
	if err != nil {
		b.Fatal(err)
	}

	ts := test.NewSuiteFS(b, vfs, vfs)
	ts.BenchAll(b)
}
