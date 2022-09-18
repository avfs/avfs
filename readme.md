# AVFS

Another Virtual File System for Go

![CI](https://github.com/avfs/avfs/workflows/CI/badge.svg)
[![codecov](https://codecov.io/gh/avfs/avfs/branch/master/graph/badge.svg?token=z325AezWts)](https://codecov.io/gh/avfs/avfs)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/avfs/avfs)](https://pkg.go.dev/github.com/avfs/avfs)
[![Release](https://img.shields.io/github/release/avfs/avfs.svg)](https://github.com/avfs/avfs/releases/latest)
[![License](https://img.shields.io/github/license/avfs/avfs)](/LICENSE)
[![Built with Mage](https://magefile.org/badge.svg)](https://magefile.org)

## Overview

**AVFS** is a virtual file system abstraction, inspired mostly
by [Afero](http://github.com/spf13/afero) and Go standard library. It provides
an abstraction layer to emulate the behavior of a file system that provides several features :
- a set of **constants**, **interfaces** and **types** for all file systems
- a **test suite** for all file systems (emulated or real)
- each file system has its **own package**
- a very basic **identity manager** allows testing of user related functions
(Chown, Lchown) and file system permissions

Additionally, some file systems support :
- user file creation mode mask (**Umask**) (MemFS, OrefaFS)
- **chroot** (OSFS on Linux)
- **hard links** (MemFS, OrefaFS)
- **symbolic links** (MemFS)
- **multiple users concurrently** (MemFS)
- **Linux** and **Windows** emulation regardless of host operating system (MemFS, OrefaFS)

## Installation

This package can be installed with the go install command :

```
go install github.com/avfs/avfs@latest
```

It is only tested with Go version >= 1.18

## Getting started

To make an existing code work with **AVFS** :

- replace all references of `os`, `path/filepath`
  with the variable used to initialize the file system (`vfs` in the following
  examples)
- import the packages of the file systems and, if necessary, the `avfs` package
  and initialize the file system variable.
- some file systems provide specific options available at initialization. For
  instance `MemFS` needs `WithSystemDirs` option to create `/home`, `/root`
  and `/tmp` directories.

## Examples

### Symbolic links

The example below demonstrates the creation of a file, a symbolic link to this
file, for a different file systems (depending on an environment variable). Error
management has been omitted for the sake of simplicity :

```go
package main

import (
	"bytes"
	"log"
	"os"
	
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

func main() {
	var vfs avfs.VFS

	switch os.Getenv("ENV") {
	case "PROD": // The real file system for production.
		vfs = osfs.NewWithNoIdm()
	default: // in memory for tests.
		vfs = memfs.New()
	}

	// From this point all references of 'os', 'path/filepath'
	// should be replaced by 'vfs'
	rootDir, _ := vfs.MkdirTemp("", "avfs")
	defer vfs.RemoveAll(rootDir)

	aFilePath := vfs.Join(rootDir, "aFile.txt")

	content := []byte("randomContent")
	_ = vfs.WriteFile(aFilePath, content, 0o644)

	aFilePathSl := vfs.Join(rootDir, "aFileSymlink.txt")
	_ = vfs.Symlink(aFilePath, aFilePathSl)

	gotContentSl, _ := vfs.ReadFile(aFilePathSl)
	if !bytes.Equal(content, gotContentSl) {
		log.Fatalf("Symlink %s : want content to be %v, got %v",
			aFilePathSl, content, gotContentSl)
	}

	log.Printf("content from symbolic link %s : %s", aFilePathSl, gotContentSl)
}
```

### Multiple users creating simultaneously directories

The example below demonstrates the concurrent creation of subdirectories under a
root directory by several users in different goroutines (works only with
MemFS) :

```go
package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/vfs/memfs"
)

func main() {
	const (
		maxUsers  = 100
		groupName = "test_users"
	)

	idm := memidm.New()
	vfs := memfs.New()

	rootDir, _ := vfs.MkdirTemp("", "avfs")
	vfs.Chmod(rootDir, 0o777)

	g, _ := idm.GroupAdd(groupName)

	var wg sync.WaitGroup
	wg.Add(maxUsers)

	for i := 0; i < maxUsers; i++ {
		go func(i int) {
			defer wg.Done()

			userName := fmt.Sprintf("user_%08d", i)
			idm.UserAdd(userName, g.Name())

			vfsU, _ := vfs.Sub("/")
			vfsU.SetUser(userName)

			path := vfsU.Join(rootDir, userName)
			vfsU.Mkdir(path, avfs.DefaultDirPerm)
		}(i)
	}

	wg.Wait()

	entries, _ := vfs.ReadDir(rootDir)

	log.Println("number of dirs :", len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		sst := vfs.ToSysStat(info)

		u, _ := idm.LookupUserId(sst.Uid())

		log.Println("dir :", info.Name(),
			", mode :", info.Mode(),
			", owner :", u.Name())
	}
}
```

## Status

Almost ready for Windows.

## File systems

All file systems implement at least `avfs.FS` and `avfs.File` interfaces. 
By default, each file system supported methods are the most commonly used from
packages `os` and `path/filepath`. All methods have identical names as their 
functions counterparts.
The following file systems are currently available :

File system |Comments
------------|--------
[BasePathFS](vfs/basepathfs)|file system that restricts all operations to a given path within a file system
[MemFS](vfs/memfs)|In memory file system supporting major features of a linux file system (hard links, symbolic links, chroot, umask)
[OrefaFS](vfs/orefafs)|Afero like in memory file system
[OsFS](vfs/osfs)|Operating system native file system
[RoFS](vfs/rofs)|Read only file system

## Supported methods

File system methods <br> `avfs.VFS`|Comments
-----------------------------------|--------
`Abs`|equivalent to `filepath.Abs`
`Base`|equivalent to `filepath.Base`
`Chdir`|equivalent to `os.Chdir`
`Chmod`|equivalent to `os.Chmod`
`Chown`|equivalent to `os.Chown`
`Chtimes`|equivalent to `os.Chtimes`
`Clean`|equivalent to `filepath.Clean`
`Create`|equivalent to `os.Create`
`CreateTemp`|equivalent to `os.CreateTemp`
`Dir`|equivalent to `filepath.Dir`
`EvalSymlinks`|equivalent to `filepath.EvalSymlinks`
`FromSlash`|equivalent to `filepath.FromSlash`
`Features`|returns the set of features provided by the file system or identity manager
`Getwd`|equivalent to `os.Getwd`
`Glob`|equivalent to `filepath.Glob`
`HasFeature`|returns true if the file system or identity manager provides a given feature
`Idm`|returns the identity manager of the file system
`IsAbs`|equivalent to `filepath.IsAbs`
`IsPathSeparator`|equivalent to `filepath.IsPathSeparator`
`Join`|equivalent to `filepath.Join`
`Lchown`|equivalent to `os.Lchown`
`Link`|equivalent to `os.Link`
`Lstat`|equivalent to `os.Lstat`
`Match`|equivalent to `filepath.Match`
`Mkdir`|equivalent to `os.Mkdir`
`MkdirAll`|equivalent to `os.MkdirAll`
`MkdirTemp`|equivalent to `os.MkdirTemp`
`Open`|equivalent to `os.Open`
`OpenFile`|equivalent to `os.OpenFile`
`OSType`|returns the operating system type of the file system
`PathSeparator`|equivalent to `os.PathSeparator`
`ReadDir`|equivalent to `os.ReadDir`
`ReadFile`|equivalent to `os.ReadFile`
`Readlink`|equivalent to `os.Readlink`
`Rel`|equivalent to `filepath.Rel`
`Remove`|equivalent to `os.Remove`
`RemoveAll`|equivalent to `os.RemoveAll`
`Rename`|equivalent to `os.Rename`
`SameFile`|equivalent to `os.SameFile`
`SetUMask`|sets the file mode creation mask
`SetUser`|sets and returns the current user
`Split`|equivalent to `filepath.Split`
`Stat`|equivalent to `os.Stat`
`Sub`|equivalent to `fs.Sub`
`Symlink`|equivalent to `os.Symlink`
`TempDir`|equivalent to `os.TempDir`
`ToSlash`|equivalent to `filepath.ToSlash`
`ToSysStat`|takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater
`Truncate`|equivalent to `os.Truncate`
`UMask`|returns the file mode creation mask
`User`|returns the current user
`Utils`|returns the file utils of the current file system
`WalkDir`|equivalent to `filepath.WalkDir`
`WriteFile`|equivalent to `os.WriteFile`

File methods <br> `avfs.File`|Comments
-----------------------------|--------
`Chdir`|equivalent to `os.File.Chdir`
`Chmod`|equivalent to `os.File.Chmod`
`Chown`|equivalent to `os.File.Chown`
`Close`|equivalent to `os.File.Close`
`Fd`|equivalent to `os.File.Fd`
`Name`|equivalent to `os.File.Name`
`Read`|equivalent to `os.File.Read`
`ReadAt`|equivalent to `os.File.ReadAt`
`ReadDir`|equivalent to `os.File.ReadDir`
`Readdirnames`|equivalent to `os.File.Readdirnames`
`Seek`|equivalent to `os.File.Seek`
`Stat`|equivalent to `os.File.Stat`
`Truncate`|equivalent to `os.File.Truncate`
`Write`|equivalent to `os.File.Write`
`WriteAt`|equivalent to `os.File.WriteAt`
`WriteString`|equivalent to `os.File.WriteString`

## Identity Managers

Identity managers allow users and groups management. The ones implemented
in `avfs` are just here to allow testing of functions related to users (Chown,
Lchown) and access rights, so they just allow one default group per user.

All file systems supporting identity manager implement by default the identity
manager `DummyIdm`
where all functions returns `avfs.ErrPermDenied`.

Identity Manager |Comments
-----------------|--------
[DummyIdm](dummyidm.go)|dummy identity manager where all functions are not implemented
[MemIdm](idm/memidm)|In memory identity manager
[OsIdm](idm/osidm)|Identity manager using os functions
[SQLiteIdm](https://github.com/avfs/sqliteidm)|Identity manager backed by a SQLite database

Identity Manager methods <br>`avfs.FS` <br> `avfs.IdentityMgr`|Comments
--------------------------------------------------------------|--------
`AdminGroup`|returns the administrator group (root for Linux)
`AdminUser`|returns the administrator user (root for Linux)
`GroupAdd`| adds a new group
`GroupDel`| deletes an existing group
`LookupGroup`| looks up a group by name
`LookupGroupId`| looks up a group by groupid
`LookupUser`| looks up a user by username
`LookupUserId`| looks up a user by userid
`UserAdd`| adds a new user
`UserDel`| deletes an existing user
