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
	"io/fs"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
)

const (
	// Maximum number of symlinks in a path.
	slCountMax = 64
)

// MemIOFS implements a memory file system using the avfs.IOFS interface.
type MemIOFS struct {
	MemFS
}

// userDirKey identifies an identity view of a Storage: a user, by name and by
// uid, emulating an OS type.
//
// The uid is part of the key because a deleted user can be created again with
// the same name and a different uid: such a user is a different identity, and
// must not inherit the directories of its predecessor.
type userDirKey struct {
	name string      // name is the name of the user.
	uid  int         // uid is the user id of the user.
	ost  avfs.OSType // ost is the OS type emulated by the view.
}

// Storage holds the state shared by a MemFS and the clones made from it.
//
// It is created once, by the constructor, and is never replaced: every clone
// points at the same Storage, so a file created through one view is visible
// from all of them, and the unique file ids stay consistent (see SameFile).
// The nodes are guarded by their own locks, lastId is atomic, and volumes and
// userDirs are guarded by volMu and udMu, so a Storage is safe for concurrent
// use.
type Storage struct {
	rootNode *dirNode                          // rootNode is the root directory of the file system.
	lastId   atomic.Uint64                     // lastId is the last unique id used to identify files uniquely.
	volumes  volumes                           // volumes contains the volume names (for Windows only).
	volMu    sync.RWMutex                      // volMu is the RWMutex used to access volumes.
	name     string                            // name is the name of the file system.
	userDirs map[userDirKey]*avfs.UserDirMixin // userDirs are the identity views already built, by user and OS type.
	udMu     sync.Mutex                        // udMu is the Mutex used to access userDirs.
}

// userDirFor returns the identity view of the storage for user emulating ost,
// building it if it does not exist yet.
//
// The views are shared: cloning twice for the same user and OS type returns two
// file systems whose identity is the same object, so they also share their
// current directory. Chdir on one of them is visible in the others.
func (s *Storage) userDirFor(
	idm avfs.IdmMgr, user avfs.UserReader, ost avfs.OSType, features avfs.Features, umask fs.FileMode,
) (*avfs.UserDirMixin, error) {
	key := userDirKey{name: user.Name(), uid: user.Uid(), ost: ost}

	s.udMu.Lock()
	defer s.udMu.Unlock()

	ud, ok := s.userDirs[key]
	if ok {
		return ud, nil
	}

	ud = &avfs.UserDirMixin{}

	// A view whose OS type can't be set is not registered: the next call must
	// try again rather than hand out a view of the wrong OS.
	err := ud.Init(ost, idm, user, "")
	if err != nil {
		return nil, err
	}

	// The features and the umask describe the content and the creation policy,
	// not the identity: a view gets those of the file system it is cloned from.
	_ = ud.SetFeatures(features)
	_ = ud.SetUMask(umask)

	if s.userDirs == nil {
		s.userDirs = make(map[userDirKey]*avfs.UserDirMixin)
	}

	s.userDirs[key] = ud

	return ud, nil
}

// MemFS implements a memory file system using the avfs.VFS interface.
//
// A MemFS is immutable once built: its user, its emulated OS type and its
// identity manager are set by the constructor and never change. Use
// CloneWithUser or CloneWithUserName to obtain a view of the same content
// acting as another user or emulating another OS.
type MemFS struct {
	err     *avfs.ErrorsForOS  // err regroups errors depending on the OS emulated.
	storage *Storage           // storage is the state shared with the clones of this file system.
	userDir *avfs.UserDirMixin // userDir is the identity and the user directories of the file system.
}

// MemFile represents an open file descriptor.
type MemFile struct {
	nd         node          // nd is node of the file.
	vfs        *MemFS        // vfs is the memory file system of the file.
	name       string        // name is the name of the file.
	dirEntries []fs.DirEntry // dirEntries stores the file information returned by ReadDir function.
	dirNames   []string      // dirNames stores the names of the file returned by Readdirnames function.
	at         int64         // at is the current position in the file used by Read and Write functions.
	dirIndex   int           // dirIndex is the position of the current index for dirEntries ou dirNames slices.
	mu         sync.RWMutex  // mu is the RWMutex used to access content of MemFile.
	openMode   avfs.OpenMode // openMode defines the permissions to check for OpenFile and CheckPermission functions.
}

// Options defines the initialization options of MemFS.
type Options struct {
	Idm        avfs.IdmMgr     // Idm is the identity manager of the file system.
	User       avfs.UserReader // User is the current user of the file system.
	Name       string          // Name is the name of the file system.
	SystemDirs []avfs.DirInfo  // SystemDirs contains data to create system directories.
	OSType     avfs.OSType     // OSType defines the operating system type.
}

// node is the interface implemented by dirNode, fileNode and symlinkNode.
type node interface {
	sync.Locker

	// checkPermission returns true if the current user has the desired permissions (perm) on the node.
	checkPermission(perm avfs.OpenMode, u avfs.UserReader) bool

	// delete removes all information from the node.
	delete()

	// fillStatFrom returns a *MemInfo (implementation of fs.FileInfo) from a node named name.
	fillStatFrom(name string) *MemInfo

	// setMode sets the permissions of the node.
	setMode(mode fs.FileMode, u avfs.UserReader) bool

	// setModTime sets the modification time of the node.
	setModTime(mtime time.Time, u avfs.UserReader) bool

	// setOwner sets the owner of the node.
	setOwner(uid, gid int)

	// size returns the size of the node.
	size() int64
}

// volumes are the volumes names for Windows.
type volumes map[string]*dirNode

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
	mtime time.Time    // mtime is the modification time.
	uid   int          // uid is the user id.
	gid   int          // gid is the group id.
	mu    sync.RWMutex // mu is the RWMutex used to access the content of the node.
	mode  fs.FileMode  // mode represents a file's mode and permission bits.
}

// slMode defines the behavior of the searchNode function relatively to symlinks.
type slMode int

const (
	slmLstat slMode = iota + 1 // slmLstat makes the searchNode function follow symbolic links like Lstat.
	slmStat                    // slmStat makes the searchNode function follow symbolic links like Stat.
	slmEval                    // slmEval makes the searchNode function follow symbolic links like EvalSymlink.
)

// MemInfo is the implementation of fs.DirEntry (returned by ReadDir) and fs.FileInfo (returned by Stat and Lstat).
type MemInfo struct {
	mtime time.Time   // mtime is the modification time.
	name  string      // name is the name of the file.
	id    uint64      // id is a unique id to identify a file (used by SameFile function).
	size  int64       // size is the size of the file.
	uid   int         // uid is the user id.
	gid   int         // gid is the group id.
	nlink int         // nlink is the number of hardlinks to this fileNode.
	mode  fs.FileMode // mode represents a file's mode and permission bits.
}
