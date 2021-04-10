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
	rootNode *dirNode        // rootNode represent the root directory of the file system.
	memAttrs *memAttrs       // memAttrs represents the file system attributes.
	user     avfs.UserReader // User is the current user of the file system.
	curDir   string          // curDir is the current directory.
}

// memAttrs represents the file system attributes for MemFS.
type memAttrs struct {
	idm     avfs.IdentityMgr // idm is the identity manager of the file system.
	name    string           // name is the name of the file system.
	feature avfs.Feature     // feature defines the list of features available on this file system.
	lastId  uint64           // lastId is the last unique id used to identify files uniquely.
	umask   int32            // umask is the user file creation mode mask.
}

// MemFile represents an open file descriptor.
type MemFile struct {
	nd       node          // nd is node of the file.
	memFS    *MemFS        // memFS is the memory file system of the file.
	name     string        // name is the name of the file.
	dirInfos []os.FileInfo // dirInfos stores the files informations returned by Readdir function.
	dirNames []string      // dirNames stores the names of the file returned by Readdirnames function.
	at       int64         // at is current position in the file used by Read and Write functions.
	dirIndex int           // dirIndex is the position of the current index for dirInfos ou dirNames slices.
	mu       sync.RWMutex  // mu is the RWMutex used to access content of MemFile.
	wantMode avfs.WantMode // wantMode
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
	children children // children are the nodes present in the directory.
	baseNode          // baseNode is the common structure of directories, files and symbolic links.
}

// children are the children of a directory.
type children = map[string]node

// fileNode is the structure for a file.
type fileNode struct {
	data     []byte // data is the file content.
	baseNode        // baseNode is the common structure of directories, files and symbolic links.
	id       uint64 // id is a unique id to identify a file (used by SameFile function).
	nlink    int    // nlink is the number of hardlinks to this fileNode.
}

// symlinkNode is the structure for a symbolic link.
type symlinkNode struct {
	link     string // link is the symbolic link value.
	baseNode        // baseNode is the common structure of directories, files and symbolic links.
}

// baseNode is the common structure of directories, files and symbolic links.
type baseNode struct {
	mu    sync.RWMutex // mu is the RWMutex used to access the content of the node.
	mtime int64        // mtime is the modification time.
	mode  os.FileMode  // mode represents a file's mode and permission bits.
	uid   int          // uid is the user id.
	gid   int          // gid is the group id.
}

// slMode defines the behavior of searchNode function relatively to symlinks.
type slMode int

const (
	slmLstat slMode = iota + 1 // slmLstat makes searchNode function follow symbolic links like Lstat.
	slmStat                    // slmStat makes searchNode function follow symbolic links like Stat.
	slmEval                    // slmEval makes searchNode function follow symbolic links like EvalSymlink.
)

// fStat is the implementation of os.FileInfo returned by Stat and Lstat.
type fStat struct {
	name  string      // name is the name of the file.
	id    uint64      // id is a unique id to identify a file (used by SameFile function).
	size  int64       // size is the size of the file.
	mtime int64       // mtime is the modification time.
	uid   int         // uid is the user id.
	gid   int         // gid is the group id.
	nlink int         // nlink is the number of hardlinks to this fileNode.
	mode  os.FileMode // mode represents a file's mode and permission bits.
}

// removeStack is a stack of directories to be removed during tree traversal in RemoveAll function.
type removeStack struct {
	stack []*dirNode
}
