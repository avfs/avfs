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

package memfs_test

import (
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

func ExampleNew() {
	vfs := memfs.New(memfs.WithOSType(avfs.OsLinux))
	tmpDir := vfs.TempDir()

	_, err := vfs.Stat(tmpDir)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("%s does not exist", tmpDir)
	}

	// Output: /tmp does not exist
}

func ExampleWithSystemDirs() {
	vfs := memfs.New(memfs.WithSystemDirs(), memfs.WithOSType(avfs.OsLinux))

	info, err := vfs.Stat(vfs.TempDir())
	if err != nil {
		log.Fatalf("stat : want error to be nil, got %v", err)
	}

	fmt.Println(info.Name())

	// Output: tmp
}

func ExampleWithIdm() {
	idm := memidm.New(memidm.WithOSType(avfs.OsLinux))
	vfs := memfs.New(memfs.WithIdm(idm), memfs.WithSystemDirs(), memfs.WithOSType(avfs.OsLinux))

	fmt.Println(vfs.User().Name())

	// Output: root
}

func ExampleMemFS_Sub() {
	idm := memidm.New(memidm.WithOSType(avfs.OsLinux))
	vfsSrc := memfs.New(memfs.WithSystemDirs(), memfs.WithIdm(idm), memfs.WithOSType(avfs.OsLinux))

	_, err := vfsSrc.Idm().UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	vfsSub, err := vfsSrc.Sub("/")
	if err != nil {
		log.Fatalf("Sub : want error to be nil, got %v", err)
	}

	_, err = vfsSub.SetUser(test.UsrTest)
	if err != nil {
		log.Fatalf("SetUser : want error to be nil, got %v", err)
	}

	fmt.Println(vfsSrc.User().Name())
	fmt.Println(vfsSub.User().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFS_SetUser() {
	idm := memidm.New(memidm.WithOSType(avfs.OsLinux))
	vfs := memfs.New(memfs.WithSystemDirs(), memfs.WithIdm(idm), memfs.WithOSType(avfs.OsLinux))

	_, err := vfs.Idm().UserAdd(test.UsrTest, idm.AdminGroup().Name())
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.User().Name())

	_, err = vfs.SetUser(test.UsrTest)
	if err != nil {
		log.Fatalf("SetUser : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.User().Name())

	// Output:
	// root
	// UsrTest
}
