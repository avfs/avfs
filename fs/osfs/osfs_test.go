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

package osfs_test

import (
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/osfs"
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

var (
	// osfs.OsFs struct implements avfs.Fs interface.
	_ avfs.Fs = &osfs.OsFs{}

	// os.File struct implements avfs.File interface.
	_ avfs.File = &os.File{}
)

func initTest(t *testing.T) *test.ConfigFs {
	fsRoot, err := osfs.New(osfs.OptIdm(osidm.New()))
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	cf := test.NewConfigFs(t, fsRoot)

	return cf
}

func TestOsFs(t *testing.T) {
	cf := initTest(t)
	cf.SuiteAll()
}

func TestOsFsPerm(t *testing.T) {
	cf := initTest(t)
	cf.SuitePerm()
}

func TestNilPtrReceiver(t *testing.T) {
	f := (*os.File)(nil)

	test.SuiteNilPtrFile(t, f)
}

func TestOsFsOSType(t *testing.T) {
	fs, err := osfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	ost := fs.OSType()
	if ost != fsutil.RunTimeOS() {
		t.Errorf("OSType : want os type to be %v, got %v", fsutil.RunTimeOS(), ost)
	}
}

func BenchmarkOsFsCreate(b *testing.B) {
	fs, err := osfs.New()
	if err != nil {
		b.Fatalf("New : want error to be nil, got %v", err)
	}

	test.BenchmarkCreate(b, fs)
}
