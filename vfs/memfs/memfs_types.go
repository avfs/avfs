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
	// rootNode represent the root directory of the file system.
	rootNode *dirNode

	// MemAttrs represents the file system attributes.
	MemAttrs *MemAttrs

	// User is the current user of the file system.
	user avfs.UserReader

	// curDir is the current directory.
	curDir string
}

// MemAttrs represents the file system attributes for MemFS.
type MemAttrs struct {
	// idm is the identity manager of the file system.
	idm avfs.IdentityMgr

	// name is the name of the file system.
	name string

	// feature defines the list of features available on this file system.
	feature avfs.Feature

	// lastId is the last unique id used to identify files uniquely.
	lastId uint64

	// umask is the user file creation mode mask.
	umask int32
}

// MemFile represents an open file descriptor.
type MemFile struct {
	// nd is node of the file.
	nd node

	// memFS is the memory file system of the file.
	memFS *MemFS

	// name is the name of the file.
	name string

	// dirInfos stores the files informations returned by Readdir function.
	dirInfos []os.FileInfo

	// dirNames stores the names of the file returned by Readdirnames function.
	dirNames []string

	// at is current position in the file used by Read and Write functions.
	at int64

	// dirIndex is the position of the current index for dirInfos ou dirNames slices.
	dirIndex int

	// mu is the RWMutex used to prevent concurrent u
	mu sync.RWMutex

	// wantMode
	wantMode avfs.WantMode
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
	// children are the nodes present in the directory.
	children children

	// baseNode is the common structure of directories, files and symbolic links.
	baseNode
}

// children are the children of a directory.
type children = map[string]node

// fileNode is the structure for a file.
type fileNode struct {
	// data is the file content.
	data []byte

	// baseNode is the common structure of directories, files and symbolic links.
	baseNode

	// id is a unique id to identify a file (used by SameFile function).
	id uint64

	// nlink os the number of hardlinks to this file.
	nlink int
}

// symlinkNode is the structure for a symbolic link.
type symlinkNode struct {
	// link is the symbolic link value.
	link string

	// baseNode is the common structure of directories, files and symbolic links.
	baseNode
}

// baseNode is the common structure of directories, files and symbolic links.
type baseNode struct {
	// mu is the RWMutex used to access the content of the node.
	mu sync.RWMutex

	// mtime is the modification time.
	mtime int64

	// mode represents a file's mode and permission bits.
	mode os.FileMode

	// uid is the user id.
	uid int

	// gid is the group id.
	gid int
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
	name  string
	id    uint64
	size  int64
	mtime int64
	uid   int
	gid   int
	nlink int
	mode  os.FileMode
}

// removeStack is a stack of directories to be removed during tree traversal in RemoveAll function.
type removeStack struct {
	stack []*dirNode
}
