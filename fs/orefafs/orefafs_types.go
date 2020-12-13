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
	"os"
	"sync"

	"github.com/avfs/avfs"
)

// OrefaFs implements a memory file system using the avfs.VFS interface.
type OrefaFs struct {
	mu      sync.RWMutex
	nodes   nodes
	curDir  string
	name    string
	feature avfs.Feature
	umask   int32
}

// OrefaFile represents an open file descriptor.
type OrefaFile struct {
	mu       sync.RWMutex
	vFs      *OrefaFs
	nd       *node
	name     string
	at       int64
	wantMode avfs.WantMode
	dirInfos []os.FileInfo
	dirNames []string
	dirIndex int
}

// Option defines the option function used for initializing OrefaFs.
type Option func(*OrefaFs) error

// nodes is the map of nodes (files or directory) where the key is the absolute path.
type nodes map[string]*node

// children is the map of children (files or directory) of a directory where the key is the relative path.
type children nodes

// node is the common structure of directories and files.
type node struct {
	mu       sync.RWMutex
	mtime    int64
	mode     os.FileMode
	children children
	data     []byte
	nlink    int
}

// fStat is the implementation of os.FileInfo returned by Stat and Lstat.
type fStat struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime int64
}
