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

package osfs_test

import (
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/avfs/avfs/vfsutils"
)

var (
	// osfs.OsFS struct implements avfs.VFS interface.
	_ avfs.VFS = &osfs.OsFS{}

	// os.File struct implements avfs.File interface.
	_ avfs.File = &os.File{}
)

func initTest(tb testing.TB) *test.SuiteFS {
	vfs, err := osfs.New(osfs.WithIdm(osidm.New()))
	if err != nil {
		tb.Fatalf("New : want err to be nil, got %s", err)
	}

	sfs := test.NewSuiteFS(tb, vfs)

	return sfs
}

func TestOsFS(t *testing.T) {
	sfs := initTest(t)
	sfs.TestAll(t)
}

func TestOsFSWithNoIdm(t *testing.T) {
	vfs, err := osfs.New()
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestOsFSNilPtrFile(t *testing.T) {
	f := (*os.File)(nil)

	test.FileNilPtr(t, f)
}

func TestOsFSConfig(t *testing.T) {
	vfs, err := osfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	wantFeatures := avfs.FeatBasicFs | avfs.FeatRealFS | avfs.FeatMainDirs

	switch vfs.OSType() {
	case avfs.OsLinux:
		wantFeatures |= avfs.FeatChroot | avfs.FeatHardlink | avfs.FeatSymlink
	case avfs.OsDarwin:
		wantFeatures |= avfs.FeatHardlink | avfs.FeatSymlink
	}

	if vfs.Features() != wantFeatures {
		t.Errorf("Features : want Features to be %s, got %s", wantFeatures, vfs.Features())
	}

	name := vfs.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %v", name)
	}

	ost := vfs.OSType()
	if ost != vfsutils.RunTimeOS() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsutils.RunTimeOS(), ost)
	}
}

func BenchmarkOsFSAll(b *testing.B) {
	sfs := initTest(b)
	sfs.BenchAll(b)
}
