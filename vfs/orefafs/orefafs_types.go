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

package orefafs

import (
	"io/fs"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
)

// Storage holds the content shared by an OrefaFS and the clones made from it.
//
// It is created once, by the constructor, and is never replaced: every clone
// points at the same Storage, so a file created through one view is visible
// from all of them, and the unique file ids stay consistent (see SameFile).
// It is safe for concurrent use: nodes is guarded by mu and lastId is atomic.
type Storage struct {
	nodes  nodes         // nodes is the map of nodes (files or directories) where the key is the absolute path.
	lastId atomic.Uint64 // lastId is the last unique id used to identify files uniquely.
	mu     sync.RWMutex  // mu is the RWMutex used to access nodes.
}

// OrefaFS implements a memory file system using the avfs.VFS interface.
//
// An OrefaFS is immutable once built: its user, its emulated OS type and its
// identity manager are set by the constructor and never change. Use
// CloneWithUser or CloneWithUserName to obtain a view of the same content
// acting as another user or emulating another OS.
type OrefaFS struct {
	err     *avfs.ErrorsForOS // err regroups errors depending on the OS emulated.
	storage *Storage          // storage is the content shared with the clones of this file system.
	name    string            // name is the name of the file system.
	userDir avfs.UserDirMixin // userDir is the identity and the user directories of the file system.
}

// OrefaFile represents an open file descriptor.
type OrefaFile struct {
	vfs        *OrefaFS      // vfs is the memory file system of the file.
	nd         *node         // nd is node of the file.
	name       string        // name is the name of the file.
	dirEntries []fs.DirEntry // dirEntries stores the file information returned by ReadDir function.
	dirNames   []string      // dirNames stores the names of the file returned by Readdirnames function.
	at         int64         // at is current position in the file used by Read and Write functions.
	dirIndex   int           // dirIndex is the position of the current index for dirEntries ou dirNames slices.
	mu         sync.RWMutex  // mu is the RWMutex used to access content of OrefaFile.
	openMode   avfs.OpenMode // OpenMode defines constants used by OpenFile and CheckPermission functions.
}

// Options defines the initialization options of OrefaFS.
type Options struct {
	User       avfs.UserReader // User is the current user of the file system.
	Name       string          // Name is the name of the file system.
	SystemDirs []avfs.DirInfo  // SystemDirs contains data to create system directories.
	OSType     avfs.OSType     // OSType defines the operating system type.
}

// nodes is the map of nodes (files or directories) where the key is the absolute path.
type nodes map[string]*node

// children is the map of children (files or directories) of a directory where the key is the relative path.
type children nodes

// node is the common structure of directories and files.
type node struct {
	mtime    time.Time
	children children
	data     []byte
	uid      int
	id       uint64
	gid      int
	nlink    int
	mu       sync.RWMutex
	mode     fs.FileMode
}

// OrefaInfo is the implementation of fs.FileInfo returned by Stat and Lstat.
type OrefaInfo struct {
	mtime time.Time
	name  string
	id    uint64
	size  int64
	uid   int
	gid   int
	nlink int
	mode  fs.FileMode
}
