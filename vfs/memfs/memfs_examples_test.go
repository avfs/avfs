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
	"fmt"
	"log"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

func ExampleNew() {
	vfs, err := memfs.New()
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = vfs.Stat(avfs.TmpDir)
	if vfs.IsNotExist(err) {
		fmt.Printf("%s does not exist", avfs.TmpDir)
	}

	// Output: /tmp does not exist
}

func ExampleWithMainDirs() {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	info, err := vfs.Stat(avfs.TmpDir)
	if err != nil {
		log.Fatalf("stat : want error to be nil, got %v", err)
	}

	fmt.Println(info.Name())

	// Output: tmp
}

func ExampleWithIdm() {
	idm := memidm.New()

	vfs, err := memfs.New(memfs.WithIdm(idm), memfs.WithMainDirs())
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.CurrentUser().Name())

	// Output: root
}

func ExampleMemFS_Clone() {
	vfsSrc, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = vfsSrc.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	vfsCloned := vfsSrc.Clone()

	_, err = vfsCloned.User(test.UsrTest)
	if err != nil {
		log.Fatalf("User : want error to be nil, got %v", err)
	}

	fmt.Println(vfsSrc.CurrentUser().Name())
	fmt.Println(vfsCloned.CurrentUser().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFS_User() {
	vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = vfs.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.CurrentUser().Name())

	_, err = vfs.User(test.UsrTest)
	if err != nil {
		log.Fatalf("User : want error to be nil, got %v", err)
	}

	fmt.Println(vfs.CurrentUser().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFS_UserAdd() {
	vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	u, err := vfs.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fmt.Println(u.Name())
	fmt.Println(u.Gid())

	// Output:
	// UsrTest
	// 0
}
