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

package rofs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/rofs"
)

var (
	// rofs.RoFS struct implements avfs.VFS interface.
	_ avfs.VFS = &rofs.RoFS{}

	// rofs.RoFile struct implements avfs.File interface.
	_ avfs.File = &rofs.RoFile{}
)

func initTest(t *testing.T) *test.SuiteFS {
	vfsWrite, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %v", err)
	}

	vfs := rofs.New(vfsWrite)

	sfs := test.NewSuiteFS(t, vfsWrite, test.WithVFSRead(vfs))

	return sfs
}

func TestRoFS(t *testing.T) {
	sfs := initTest(t)
	sfs.TestRead(t)
	sfs.TestWriteOnReadOnly(t)
	sfs.TestEvalSymlink(t)
	sfs.TestPath(t)
}

func TestRoFSPerm(t *testing.T) {
	sfs := initTest(t)
	sfs.TestPermRead(t)
}

func TestRoFSOSType(t *testing.T) {
	vfsWrite, err := memfs.New()
	if err != nil {
		t.Fatalf("New : want err to be nil, got %v", err)
	}

	vfs := rofs.New(vfsWrite)

	ost := vfs.OSType()
	if ost != vfsWrite.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsWrite.OSType(), ost)
	}
}
