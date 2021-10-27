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
	user     avfs.UserReader
	nodes    nodes
	curDir   string
	name     string
	features avfs.Features
	lastId   uint64
	mu       sync.RWMutex
	umask    int32
	utils    avfs.Utils
}

// OrefaFile represents an open file descriptor.
type OrefaFile struct {
	orFS       *OrefaFS
	nd         *node
	name       string
	dirEntries []fs.DirEntry
	dirNames   []string
	at         int64
	dirIndex   int
	mu         sync.RWMutex
	permMode   avfs.PermMode
}

// Option defines the option function used for initializing OrefaFS.
type Option func(*OrefaFS)

// nodes is the map of nodes (files or directory) where the key is the absolute path.
type nodes map[string]*node

// children is the map of children (files or directory) of a directory where the key is the relative path.
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
