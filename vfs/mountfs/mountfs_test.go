//
//  Copyright 2022 The AVFS authors
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

package mountfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/mountfs"
)

var (
	// Tests that mountfs.MountFS struct implements avfs.VFS interface.
	_ avfs.VFS = &mountfs.MountFS{}

	// Tests that mountfs.MountFile struct implements avfs.File interface.
	_ avfs.File = &mountfs.MountFile{}
)

func initFS(tb testing.TB) *mountfs.MountFS {
	idm := memidm.New()
	rootFS := memfs.NewWithOptions(&memfs.Options{Idm: idm, Name: "rootFS", SystemDirs: true})
	tmpFS := memfs.NewWithOptions(&memfs.Options{Idm: idm, Name: "tmpFS"})

	vfs := mountfs.New(rootFS, "")

	err := vfs.Mount(tmpFS, "/tmp", "")
	if err != nil {
		tb.Fatalf("Can't mount /tmp file system : %v", err)
	}

	return vfs
}

func TestMountFS(t *testing.T) {
	vfs := initFS(t)
	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}
