# AVFS
Another Virtual File System for Go

![CI](https://github.com/avfs/avfs/workflows/CI/badge.svg)
[![codecov](https://codecov.io/gh/avfs/avfs/branch/master/graph/badge.svg)](https://codecov.io/gh/avfs/avfs)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/avfs/avfs?tab=doc)
[![Release](https://img.shields.io/github/release/avfs/avfs.svg)](https://github.com/avfs/avfs/releases/latest)
[![License](https://img.shields.io/github/license/avfs/avfs)](/LICENSE)
[![Built with Mage](https://magefile.org/badge.svg)](https://magefile.org)

## Overview
**AVFS** is a virtual file system abstraction, 
inspired mostly by [Afero](http://github.com/spf13/afero) and Go standard library.
It provides an abstraction layer to emulate the behavior of a **Linux file system** and provides several features :
- a set of **constants**, **interfaces** and **types** for all file systems
- a **test suite** for all file systems (emulated or real)
- a very basic **identity manager** allows testing of user related functions (Chown, Lchown) and file system permissions
- all file systems support user file creation mode mask (**Umask**) 
- **symbolic links**, **hard links** and **chroot** are fully supported for some file systems (MemFs, OsFs) 
- some file systems support **multiple users concurrently**  (MemFs)
- each file system has its **own package**

## Installation
This package can be installed with the go get command :
```
go get github.com/avfs/avfs
```
It is only tested with Go modules enabled (`GO111MODULE=on`) and Go version >= 1.13

## Getting started
To make an existing code work with **AVFS** :
- replace all references of `os`, `path/filepath` and `ioutil` 
with the variable used to initialize the file system (`fs` in the following examples)
- replace all references of `os.TempDir()` by `fs.GetTempDir()`.
- import the packages of the file systems and, if necessary, the `avfs` package 
and initialize the file system variable.
- some file systems provide specific options available at initialization.
For instance `MemFs` needs `OptMainDirs` option to create `/home`, `/root` and `/tmp` directories.

## Examples

### Symbolic links
The example below demonstrates the creation of a file, a symbolic link to this file, 
for a different file systems (depending on an environment variable).
Error management has been omitted for the sake of simplicity :
  
```go
package main

import (
    "bytes"
    "log"
    "os"

    "github.com/avfs/avfs"
    "github.com/avfs/avfs/fs/memfs"
    "github.com/avfs/avfs/fs/osfs"
)

func main() {
    var fs avfs.Fs	

    switch os.Getenv("ENV") {
    case "PROD": // The real file system for production.
        fs, _ = osfs.New()
    default: // in memory for tests.
        fs, _ = memfs.New(memfs.OptMainDirs())   
    }
        
    // From this point all references of 'os', 'path/filepath' and 'ioutil' should be replaced by 'fs'
    rootDir, _ := fs.TempDir("", avfs.Avfs)
    defer fs.RemoveAll(rootDir)

    aFilePath := fs.Join(rootDir, "aFile.txt")

    content := []byte("randomContent")
    _ = fs.WriteFile(aFilePath, content, 0o644)
    
    aFilePathSl := fs.Join(rootDir, "aFileSymlink.txt")
    _ = fs.Symlink(aFilePath, aFilePathSl)
    
    gotContentSl, _ := fs.ReadFile(aFilePathSl)
    if !bytes.Equal(content, gotContentSl) {
        log.Fatalf("Symlink %s : want content to be %v, got %v", aFilePathSl, content, gotContentSl)
    }
    
    log.Printf("content from symbolic link %s : %s", aFilePathSl, gotContentSl)
}
```

### Multiple users creating simultaneously directories
The example below demonstrates the concurrent creation of subdirectories under a root directory 
by several users in different goroutines (works only with MemFs) :

```go
package main

import (
	"fmt"
	"sync"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/memidm"
)

func main() {
	const (
		maxUsers  = 100
		groupName = "test_users"
	)

	fs, _ := memfs.New(memfs.OptMainDirs(), memfs.OptIdm(memidm.New()))

	rootDir, _ := fs.TempDir("", avfs.Avfs)
	fs.Chmod(rootDir, 0o777)

	g, _ := fs.GroupAdd(groupName)

	var wg sync.WaitGroup
	wg.Add(maxUsers)

	for i := 0; i < maxUsers; i++ {
		go func(i int) {
			defer wg.Done()

			userName := fmt.Sprintf("user_%08d", i)
			fs.UserAdd(userName, g.Name())

			fsU := fs.Clone()
			fsU.User(userName)

			path := fsU.Join(rootDir, userName)
			fsU.Mkdir(path, avfs.DefaultDirPerm)
		}(i)
	}

	wg.Wait()

	infos, _ := fs.ReadDir(rootDir)

	fmt.Println("number of dirs :", len(infos))
	for _, info := range infos {
		statT := fsutil.AsStatT(info.Sys())
		u, _ := fs.LookupUserId(int(statT.Uid))

		fmt.Println("directory :", info.Name(), ", mode :", info.Mode(), ", owner :", u.Name())
	}
}
```

## Status
Don't be fooled by the coverage percentages, everything is a work in progress.
All the file systems pass the common test suite, but you should not use this 
in a production environment at this time.

## Diagram
The interface diagram below represents the main interfaces, methods and relations between them :

<img src="avfs_diagram.svg" style="max-width:100%;">

## File systems
All file systems implement at least `avfs.Fs` and `avfs.File` interfaces.
By default, each file system supported methods are the most commonly used
from packages `os`, `path/filepath` and `ioutil`.
All methods (except `TempDir` which is found in packages `os` and `ioutil`)
have identical names as their functions counterparts. 
The following file systems are currently available :

File system |Comments
------------|--------
[BasePathFs](fs/basepathfs)|file system that restricts all operations to a given path within a file system
[DummyFs](fs/dummyfs)|Non implemented file system to be used as model
[MemFs](fs/memfs)|In memory file system supporting major features of a linux file system (hard links, symbolic links, chroot, umask)
[OrefaFs](fs/orefafs)|Afero like in memory file system
[OsFs](fs/osfs)|Operating system native file system
[RoFs](fs/rofs)|Read only file system

## Supported methods
File system methods <br> `avfs.Fs`|Comments
----------------------------------|--------
`Abs`|equivalent to `filepath.Abs`
`Base`|equivalent to `filepath.Base`
`Chdir`|equivalent to `os.Chdir`
`Chmod`|equivalent to `os.Chmod`
`Chown`|equivalent to `os.Chown`
`Chroot`|equivalent to `syscall.Chroot`
`Chtimes`|equivalent to `os.Chtimes`
`Clean`|equivalent to `filepath.Clean`
`Clone`| returns a shallow copy of the current file system (see MemFs) or the file system itself
`Create`|equivalent to `os.Create`
`Dir`|equivalent to `filepath.Dir`
`EvalSymlinks`|equivalent to `filepath.EvalSymlinks`
`FromSlash`|equivalent to `filepath.FromSlash`
`Features`| returns the set of features provided by the file system or identity manager
`GetTempDir`|equivalent to `os.TempDir`
`Getwd`|equivalent to `os.Getwd`
`Glob`|equivalent to `filepath.Glob`
`HasFeature`| returns true if the file system or identity manager provides a given feature
`IsAbs`|equivalent to `filepath.IsAbs`
`IsPathSeparator`|equivalent to `filepath.IsPathSeparator`
`Join`|equivalent to `filepath.Join`
`Lchown`|equivalent to `os.Lchown`
`Link`|equivalent to `os.Link`
`Lstat`|equivalent to `os.Lstat`
`Mkdir`|equivalent to `os.Mkdir`
`MkdirAll`|equivalent to `os.MkdirAll`
`Open`|equivalent to `os.Open`
`OpenFile`|equivalent to `os.OpenFile`
`OSType`| returns the operating system type of the file system
`ReadDir`|equivalent to `ioutil.ReadDir`
`ReadFile`|equivalent to `ioutil.ReadFile`
`Readlink`|equivalent to `os.Readlink`
`Rel`|equivalent to `filepath.Rel`
`Remove`|equivalent to `os.Remove`
`RemoveAll`|equivalent to `os.RemoveAll`
`Rename`|equivalent to `os.Rename`
`SameFile`|equivalent to `os.SameFile`
`Split`|equivalent to `filepath.Split`
`Stat`|equivalent to `os.Stat`
`Symlink`|equivalent to `os.Symlink`
`TempDir`|equivalent to `ioutil.TempDir`
`TempFile`|equivalent to `ioutil.TempFile`
`ToSlash`|equivalent to `filepath.ToSlash`
`Truncate`|equivalent to `os.Truncate`
`Walk`|equivalent to `filepath.Walk`
`WriteFile`|equivalent to `ioutil.WriteFile`

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
`Readdir`|equivalent to `os.File.Readdir`
`Readdirnames`|equivalent to `os.File.Readdirnames`
`Seek`|equivalent to `os.File.Seek`
`Stat`|equivalent to `os.File.Stat`
`Truncate`|equivalent to `os.File.Truncate`
`Write`|equivalent to `os.File.Write`
`WriteAt`|equivalent to `os.File.WriteAt`
`WriteString`|equivalent to `os.File.WriteString`

## Identity Managers
Identity managers allow users and groups management.
The ones implemented in `avfs` are just here to allow testing of functions related to users (Chown, Lchown)
and access rights, so they just allow one default group per user.

All file systems supporting identity manager implement by default the identity manager `DummyIdm`
where all functions returns `avfs.ErrPermDenied`. 

Identity Manager |Comments
-----------------|--------
[DummyIdm](idm/dummyidm)|dummy identity manager where all functions are not implemented
[MemIdm](idm/memidm)|In memory identity manager
[OsIdm](idm/osidm)|Identity manager using os functions
[SQLiteIdm](https://github.com/avfs/sqliteidm)|Identity manager backed by a SQLite database

Identity Manager methods <br>`avfs.Fs` <br> `avfs.IdentityMgr`|Comments
--------------------------------------------------------------|--------
`GroupAdd`| adds a new group
`GroupDel`| deletes an existing group
`LookupGroup`| looks up a group by name
`LookupGroupId`| looks up a group by groupid
`LookupUser`| looks up a user by username
`LookupUserId`| looks up a user by userid
`UserAdd`| adds a new user
`UserDel`| deletes an existing user

All the file systems and some Identity managers (see OsFs) provide an additional interface `UserConnecter`

UserConnecter methods <br>`avfs.Fs`|Comments
------------------------------------|--------
`CurrentUser`| returns the current user
`User`| sets and returns the current user

