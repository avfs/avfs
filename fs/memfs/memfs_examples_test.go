package memfs_test

import (
	"fmt"
	"log"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
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

func ExampleMemFs_Clone() {
	vfsRoot, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = vfsRoot.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	vfs := vfsRoot.Clone()

	_, err = vfs.User(test.UsrTest)
	if err != nil {
		log.Fatalf("User : want error to be nil, got %v", err)
	}

	fmt.Println(vfsRoot.CurrentUser().Name())
	fmt.Println(vfs.CurrentUser().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFs_User() {
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

func ExampleMemFs_UserAdd() {
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
