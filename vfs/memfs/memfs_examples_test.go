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

//go:build avfs_setostype

package memfs_test

import (
	"fmt"
	"log"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

// ExampleNewWithOptions should produce the same results independently of the host OS.
func ExampleNewWithOptions() {
	idm := memidm.NewWithOptions(&memidm.Options{OSType: avfs.OsLinux})
	vfs := memfs.NewWithOptions(&memfs.Options{Idm: idm, OSType: avfs.OsLinux})

	fmt.Println(vfs.Features())
	fmt.Println(vfs.User().Name())
	fmt.Println(vfs.TempDir())

	homeDir, _ := vfs.UserHomeDir()
	fmt.Println(homeDir)

	_, err := vfs.Stat(homeDir)
	if err == nil {
		fmt.Printf("%s exists", homeDir)
	}

	// Output:
	// Features(Hardlink|IdentityMgr|SetOSType|SubFS|Symlink)
	// root
	// /tmp
	// /root
	// /root exists
}

func ExampleNewWithOptions_noSystemDirs() {
	vfs := memfs.NewWithOptions(&memfs.Options{OSType: avfs.OsLinux})
	tmpDir := vfs.TempDir()
	fmt.Println(tmpDir)

	_, err := vfs.Stat(tmpDir)
	if err != nil {
		fmt.Printf("%s does not exist", tmpDir)
	}

	// Output: /tmp
}

func ExampleNewWithOptions_noIdm() {
	vfs := memfs.NewWithOptions(&memfs.Options{Idm: avfs.DefaultIdm, OSType: avfs.OsLinux})
	fmt.Println(vfs.User().Name())

	// Output: Default
}

func ExampleMemFS_Sub() {
	idm := memidm.NewWithOptions(&memidm.Options{OSType: avfs.OsLinux})
	vfsSrc := memfs.NewWithOptions(&memfs.Options{Idm: idm, OSType: avfs.OsLinux})

	vfsSub, err := vfsSrc.Sub("/")
	if err != nil {
		log.Fatalf("Sub : want error to be nil, got %v", err)
	}

	// The subtree shares the content of the file system it was taken from: a
	// file created through one of them is visible from the other.
	path := "/file.txt"

	err = vfsSrc.WriteFile(path, []byte("content"), avfs.DefaultFilePerm)
	if err != nil {
		log.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
	}

	content, err := vfsSub.ReadFile(path)
	if err != nil {
		log.Fatalf("ReadFile %s : want error to be nil, got %v", path, err)
	}

	fmt.Println(string(content))

	// Output:
	// content
}

func ExampleMemFS_SetUserByName() {
	idm := memidm.NewWithOptions(&memidm.Options{OSType: avfs.OsLinux})
	vfs := memfs.NewWithOptions(&memfs.Options{Idm: idm, OSType: avfs.OsLinux})

	_, err := vfs.Idm().AddUser(test.UsrTest, idm.AdminGroup().Name())
	if err != nil {
		log.Fatalf("AddUser : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.User().Name())

	err = vfs.SetUserByName(test.UsrTest)
	if err != nil {
		log.Fatalf("SetUser : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.User().Name())

	// Output:
	// root
	// UsrTest
}
