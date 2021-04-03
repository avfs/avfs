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

package memfs

import (
	"os"
	"sync"
	"time"

	"github.com/avfs/avfs"
)

const (
	// Maximum number of symlinks in a path.
	slCountMax = 64
)

// MemFS implements a memory file system using the avfs.VFS interface.
type MemFS struct {
	rootNode *dirNode
	fsAttrs  *fsAttrs
	user     avfs.UserReader
	curDir   string
}

// fsAttrs represents the file system attributes for MemFS.
type fsAttrs struct {
	idm     avfs.IdentityMgr
	feature avfs.Feature
	lastId  uint64
	name    string
	umask   int32
}

// MemFile represents an open file descriptor.
type MemFile struct {
	mu       sync.RWMutex
	memFS    *MemFS
	nd       node
	name     string
	at       int64
	wantMode avfs.WantMode
	dirInfos []os.FileInfo
	dirNames []string
	dirIndex int
}

// Option defines the option function used for initializing MemFS.
type Option func(*MemFS) error

// node is the interface implemented by dirNode, fileNode and symlinkNode.
type node interface {
	// checkPermissionLck returns true if the current user has the wanted permissions (want) on the node.
	checkPermissionLck(want avfs.WantMode, u avfs.UserReader) bool

	// fillStatFrom returns a *fStat (implementation of os.FileInfo) from a node named name.
	fillStatFrom(name string) *fStat

	// setMode sets the permissions of the node.
	setMode(mode os.FileMode, u avfs.UserReader) error

	// setModTime sets the modification time of the node.
	setModTime(mtime time.Time)

	// setOwner sets the owner of the node.
	setOwner(uid, git int)

	// size returns the size of the node.
	size() int64
}

// dirNode is the structure for a directory.
type dirNode struct {
	baseNode
	children children
}

// children are the children of a directory.
type children = map[string]node

// fileNode is the structure for a file.
type fileNode struct {
	baseNode
	id    uint64
	data  []byte
	nlink int
}

// symlinkNode is the structure for a symbolic link.
type symlinkNode struct {
	baseNode
	link string
}

// baseNode is the common structure of directories, files and symbolic links.
type baseNode struct {
	mu    sync.RWMutex
	mtime int64
	mode  os.FileMode
	uid   int
	gid   int
}

// slMode defines the behavior of searchNode function relatively to symlinks.
type slMode int

const (
	// slmLstat makes searchNode function follow symbolic links like Lstat.
	slmLstat slMode = iota + 1

	// slmStat makes searchNode function follow symbolic links like Stat.
	slmStat

	// slmEval makes searchNode function follow symbolic links like EvalSymlink.
	slmEval
)

// fStat is the implementation of os.FileInfo returned by Stat and Lstat.
type fStat struct {
	id    uint64
	name  string
	size  int64
	mode  os.FileMode
	mtime int64
	uid   int
	gid   int
	nlink int
}

// removeStack is a stack of directories to be removed during tree traversal in RemoveAll function.
type removeStack struct {
	stack []*dirNode
}
