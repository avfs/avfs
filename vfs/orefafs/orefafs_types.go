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

	"github.com/avfs/avfs"
)

// OrefaFS implements a memory file system using the avfs.VFS interface.
type OrefaFS struct {
	nodes                nodes           // nodes is the map of nodes (files or directories) where the key is the absolute path.
	err                  avfs.Errors     // err regroups errors depending on the OS emulated.
	user                 avfs.UserReader // user is the current user of the file system.
	curDir               string          // curDir is the current directory.
	name                 string          // name is the name of the file system.
	features             avfs.Features   // features defines the list of features available for this file system.
	lastId               uint64          // lastId is the last unique id used to identify files uniquely.
	mu                   sync.RWMutex    // mu is the RWMutex used to access nodes.
	umask                fs.FileMode     // umask is the user file creation mode mask.
	dirMode              fs.FileMode     // dirMode is the default fs.FileMode for a directory.
	fileMode             fs.FileMode     // fileMode is de default fs.FileMode for a file.
	avfs.Utils[*OrefaFS]                 // utils regroups common functions used by emulated file systems.
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
	OSType     avfs.OSType     // OSType defines the operating system type.
	SystemDirs bool            // SystemDirs must be true to create system directories.
}

// nodes is the map of nodes (files or directories) where the key is the absolute path.
type nodes map[string]*node

// children is the map of children (files or directories) of a directory where the key is the relative path.
type children nodes

// node is the common structure of directories and files.
type node struct {
	children children
	data     []byte
	uid      int
	id       uint64
	mtime    int64
	gid      int
	nlink    int
	mu       sync.RWMutex
	mode     fs.FileMode
}

// OrefaInfo is the implementation of fs.FileInfo returned by Stat and Lstat.
type OrefaInfo struct {
	name  string
	id    uint64
	size  int64
	mtime int64
	uid   int
	gid   int
	nlink int
	mode  fs.FileMode
}
