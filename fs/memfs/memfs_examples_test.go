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
	fs, err := memfs.New()
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = fs.Stat(avfs.TmpDir)
	if fs.IsNotExist(err) {
		fmt.Printf("%s does not exist", avfs.TmpDir)
	}

	// Output: /tmp does not exist
}

func ExampleOptMainDirs() {
	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	info, err := fs.Stat(avfs.TmpDir)
	if err != nil {
		log.Fatalf("stat : want error to be nil, got %v", err)
	}

	fmt.Println(info.Name())

	// Output: tmp
}

func ExampleOptIdm() {
	idm := memidm.New()

	fs, err := memfs.New(memfs.OptIdm(idm), memfs.OptMainDirs())
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	fmt.Println(fs.CurrentUser().Name())

	// Output: root
}

func ExampleMemFs_Clone() {
	fsRoot, err := memfs.New(memfs.OptMainDirs(), memfs.OptIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = fsRoot.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fs := fsRoot.Clone()

	_, err = fs.User(test.UsrTest)
	if err != nil {
		log.Fatalf("User : want error to be nil, got %v", err)
	}

	fmt.Println(fsRoot.CurrentUser().Name())
	fmt.Println(fs.CurrentUser().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFs_User() {
	fs, err := memfs.New(memfs.OptMainDirs(), memfs.OptIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	_, err = fs.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fmt.Println(fs.CurrentUser().Name())

	_, err = fs.User(test.UsrTest)
	if err != nil {
		log.Fatalf("User : want error to be nil, got %v", err)
	}

	fmt.Println(fs.CurrentUser().Name())

	// Output:
	// root
	// UsrTest
}

func ExampleMemFs_UserAdd() {
	fs, err := memfs.New(memfs.OptMainDirs(), memfs.OptIdm(memidm.New()))
	if err != nil {
		log.Fatalf("new : want error to be nil, got %v", err)
	}

	u, err := fs.UserAdd(test.UsrTest, "root")
	if err != nil {
		log.Fatalf("UserAdd : want error to be nil, got %v", err)
	}

	fmt.Println(u.Name())
	fmt.Println(u.Gid())

	// Output:
	// UsrTest
	// 0
}
