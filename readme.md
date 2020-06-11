# AVFS
Another Virtual File System for Go

![CI](https://github.com/avfs/avfs/workflows/CI/badge.svg)
[![codecov](https://codecov.io/gh/avfs/avfs/branch/master/graph/badge.svg)](https://codecov.io/gh/avfs/avfs)
[![GoDoc](https://godoc.org/github.com/avfs/avfs?status.svg)](https://godoc.org/github.com/avfs/avfs) 

## Overview
**AVFS** is a virtual file system abstraction, 
inspired mostly by [Afero](http://github.com/spf13/afero) and Go standard library.
It provides an abstraction layer to emulate the behavior of a **Linux file system**
and provides several features :
- a common set of **interfaces**, **types** and a **test suite** are shared by all file systems (emulated or real)
- a very basic **identity manager** allows testing of user related functions (Chown, Lchown) and file system permissions
- all file systems support user file creation mode mask (**Umask**) 
- **symbolic links**, **hard links** and **chroot** are fully supported for some file systems (MemFs, OsFs) 
- some file systems support **multiple users concurrently**  (MemFs)
- each file system has its **own package**

## Getting started
To make an existing code work with **AVFS** :
- replace all references of `os`, `path/filepath` and `ioutil` with 
the variable used to initialize the file system (`fs` in the following examples)
- replace all references of `os.TempDir()` by `fs.GetTempDir()`
- import the package of the file system and, if necessary, the `avfs` package
```go
import (
    "github.com/avfs/avfs"
    "github.com/avfs/avfs/fs/osfs"
)
```
- initialize the file system variable :  
```go
fs, err := osfs.New()
```
- some file system provide specific options available at initialization.
For instance `MemFs` needs `OptMainDirs` option to create `/home`, `/root` and `/tmp` directories :
```go
fs, err := memfs.New(memfs.OptMainDirs())
```

## Example
The example below demonstrates the creation of memory file system, a file, a symbolic link and a hard link to this file.
Error management has been omitted for the sake of simplicity :

```go
package main

import (
    "bytes"
    "log"
    
    "github.com/avfs/avfs"
    "github.com/avfs/avfs/fs/memfs"
)

func main() {
    fs, _ := memfs.New(memfs.OptMainDirs())
    
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

    aFilePathHl := fs.Join(rootDir, "aFileHardLink.txt")
    _ = fs.Link(aFilePath, aFilePathHl)
    
    gotContentHl, _ := fs.ReadFile(aFilePathHl)
    if !bytes.Equal(content, gotContentHl) {
        log.Fatalf("Hardlink %s : want content to be %v, got %v", aFilePathHl, content, gotContentHl)
    }

    log.Printf("content from hard link %s : %s", aFilePathHl, gotContentHl)
}
```

## Status
Everything is a work in progress, all the file systems pass the common suite test,
but you should not use this in a production environment at this time.

## File systems
The following file systems are available

File system  |Comments
-------------|--------
BasePathFs|file system that restricts all operations to a given path within a file system
DummyFs|Non implemented file system to be used as model
MemFs|In memory file system supporting major features of a linux file system
OsFs|Operating system native file system
ReadOnlyFs|Read only file system

## Supported methods
Supported method are the most commonly used from packages `os`, `path/filepath` and `ioutil`.

File system methods|Comments
---------------------|--------
`Abs`|equivalent to `filepath.Abs`
`Base`|equivalent to `filepath.Base`
`Chdir`|equivalent to `os.Chdir`
`Chmod`|equivalent to `os.Chmod`
`Chown`|equivalent to `os.Chown`
`Chroot`|equivalent to `syscall.Chroot`
`Chtimes`|equivalent to `os.Chtimes`
`Clean`|equivalent to `filepath.Clean`
`Create`|equivalent to `os.Create`
`Dir`|equivalent to `filepath.Dir`
`EvalSymlinks`|equivalent to `filepath.EvalSymlinks`
`GetTempDir`|equivalent to `os.TempDir`
`Getwd`|equivalent to `os.Getwd`
`Glob`|equivalent to `filepath.Glob`
`IsAbs`|equivalent to `filepath.IsAbs
`IsPathSeparator`|equivalent to `filepath.IsPathSeparator`
`Join`|equivalent to `filepath.Join`
`Lchown`|equivalent to `os.Lchown`
`Link`|equivalent to `os.Link`
`Lstat`|equivalent to `os.Lstat`
`Mkdir`|equivalent to `os.Mkdir`
`MkdirAll`|equivalent to `os.MkdirAll`
`Open`|equivalent to `os.Open`
`OpenFile`|equivalent to `os.OpenFile`
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
`Truncate`|equivalent to `os.Truncate`
`Walk`|equivalent to `filepath.Walk`
`WriteFile`|equivalent to `ioutil.WriteFile`

File methods|Comments
--------------|--------
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

Identity Manager methods|Comments
--------------------------|--------
`CurrentUser`| returns the current user
`GroupAdd`| adds a new group
`GroupDel`| deletes an existing group
`LookupGroup`| looks up a group by name
`LookupGroupId`| looks up a group by groupid
`LookupUser`| looks up a user by username
`LookupUserId`| looks up a user by userid
`User`| sets and returns the current user
`UserAdd`| adds a new user
`UserDel`| deletes an existing user

Misc. methods|Comments
---------------|--------
`Clone`| returns a shallow copy of the current file system (see MemFs)
`Features`| returns the set of features provided by the file system or identity manager
`HasFeature`| returns true if the file system or identity manager provides a given feature
