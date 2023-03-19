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
	"bytes"
	"io/fs"
	"sort"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
)

// searchNode search a node from the root of the file system
// where path is the absolute or relative path of the node
// and slMode the behavior of searchNode function relatively to symlinks.
// It returns :
// parent, the parent node of the node if found, the last found node otherwise
// child, the node corresponding to the path or nil if not found
// absPath, the absolute path of the input path
// start and end, the beginning and ending position of the last found segment of absPath
// err, one of the following errors :
//
//	ErrNoSuchFileOrDir when the node is not found
//	ErrFileExists when the node is a file or directory
//	ErrPermDenied when the current user doesn't have permissions on one of the nodes on the path
//	ErrNotADirectory when a file node is found while the path segmentation is not finished
//	ErrTooManySymlinks when more than slCountMax symbolic link resolutions have been performed.
func (vfs *MemFS) searchNode(path string, slMode slMode) (
	parent *dirNode, child node, pi *avfs.PathIterator[*MemFS], err error,
) {
	slCount := 0
	slResolved := false

	absPath, _ := vfs.Abs(path)
	pi = avfs.NewPathIterator[*MemFS](vfs, absPath)

	volNode := vfs.rootNode

	if pi.VolumeNameLen() > 0 {
		nd, ok := vfs.volumes[pi.VolumeName()]
		if !ok {
			err = vfs.err.NoSuchDir

			return
		}

		volNode = nd
	}

	parent = volNode

	for pi.Next() {
		name := pi.Part()

		parent.mu.RLock()
		child = parent.children[name]
		parent.mu.RUnlock()

		if child == nil {
			err = vfs.err.NoSuchDir
			if pi.IsLast() {
				err = vfs.err.NoSuchFile
			}

			return
		}

		switch c := child.(type) {
		case *dirNode:
			if pi.IsLast() {
				err = vfs.err.FileExists

				return
			}

			c.mu.RLock()
			ok := c.checkPermission(avfs.OpenLookup, vfs.user)
			c.mu.RUnlock()

			if !ok {
				err = vfs.err.PermDenied

				return
			}

			parent = c

		case *fileNode:
			// File permissions are checked by the calling function.
			if pi.IsLast() {
				err = vfs.err.FileExists

				return
			}

			err = vfs.err.NotADirectory

			return

		case *symlinkNode:
			// Symlinks mode is always 0o777, no need to check permissions.
			slCount++
			if slCount > slCountMax {
				err = vfs.err.TooManySymlinks

				return
			}

			if pi.IsLast() {
				if slMode == slmLstat {
					err = vfs.err.FileExists

					return
				}

				// if the last part of the path is a symbolic link
				// Stat should return the initial path of the symbolic link
				// and the parent and child nodes of the resolved symbolic link.
				if slMode == slmStat && !slResolved {
					slResolved = true

					defer func(piSymLink avfs.PathIterator[*MemFS]) { //nolint:gocritic // Possible resource leak
						pi = &piSymLink
					}(*pi)
				}
			}

			if pi.ReplacePart(c.link) {
				parent = volNode
			}
		}
	}

	return parent, parent, pi, vfs.err.FileExists
}

// createRootNode creates a root node for a file system.
func (vfs *MemFS) createRootNode() *dirNode {
	u := vfs.User()
	dn := &dirNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  fs.ModeDir | 0o755,
			uid:   u.Uid(),
			gid:   u.Gid(),
		},
	}

	return dn
}

// createDir creates a new directory.
func (vfs *MemFS) createDir(parent *dirNode, name string, perm fs.FileMode) *dirNode {
	child := &dirNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  vfs.memAttrs.dirMode | (perm & avfs.FileModeMask &^ vfs.umask),
			uid:   vfs.user.Uid(),
			gid:   vfs.user.Gid(),
		},
		children: nil,
	}

	parent.addChild(name, child)

	return child
}

// createFile creates a new file.
func (vfs *MemFS) createFile(parent *dirNode, name string, perm fs.FileMode) *fileNode {
	child := &fileNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  vfs.memAttrs.fileMode | (perm & avfs.FileModeMask &^ vfs.umask),
			uid:   vfs.user.Uid(),
			gid:   vfs.user.Gid(),
		},
		id:    atomic.AddUint64(&vfs.memAttrs.lastId, 1),
		nlink: 1,
	}

	parent.addChild(name, child)

	return child
}

// createSymlink creates a new symlink.
func (vfs *MemFS) createSymlink(parent *dirNode, name, link string) *symlinkNode {
	child := &symlinkNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  fs.ModeSymlink | fs.ModePerm,
			uid:   vfs.user.Uid(),
			gid:   vfs.user.Gid(),
		},
		link: link,
	}

	parent.addChild(name, child)

	return child
}

// isNotExist is IsNotExist without unwrapping.
func (vfs *MemFS) isNotExist(err error) bool {
	return err == vfs.err.NoSuchDir || err == vfs.err.NoSuchFile
}

// checkPermission checks if the current user has the desired permissions (perm) on the node.
func (bn *baseNode) checkPermission(perm avfs.OpenMode, u avfs.UserReader) bool {
	const PermRWX = 0o007 // filter all permissions bits.

	if u.IsAdmin() {
		return true
	}

	mode := avfs.OpenMode(bn.mode)

	switch {
	case bn.uid == u.Uid():
		mode >>= 6
	case bn.gid == u.Gid():
		mode >>= 3
	}

	perm &= PermRWX

	return mode&perm == perm
}

// Lock locks the node.
func (bn *baseNode) Lock() {
	bn.mu.Lock()
}

// setModTime sets the modification time of the node.
func (bn *baseNode) setModTime(mtime time.Time, u avfs.UserReader) bool {
	if bn.uid != u.Uid() && !u.IsAdmin() {
		return false
	}

	bn.mtime = mtime.UnixNano()

	return true
}

// setOwner sets the owner of the node.
func (bn *baseNode) setOwner(uid, gid int) {
	bn.uid = uid
	bn.gid = gid
}

// Unlock unlocks the node.
func (bn *baseNode) Unlock() {
	bn.mu.Unlock()
}

// dirNode

// addChild adds a child to a dirNode.
func (dn *dirNode) addChild(name string, child node) {
	if dn.children == nil {
		dn.children = make(children)
	}

	dn.children[name] = child
}

// removeChild removes the child from the parent dirNode.
func (dn *dirNode) removeChild(name string) {
	delete(dn.children, name)
}

// delete removes all information from the node.
func (dn *dirNode) delete() {
	dn.children = nil
}

// fillStatFrom returns a MemInfo (implementation of fs.FileInfo) from a dirNode dn named name.
func (dn *dirNode) fillStatFrom(name string) *MemInfo {
	dn.mu.RLock()

	fst := &MemInfo{
		name:  name,
		size:  dn.size(),
		mode:  dn.mode,
		mtime: dn.mtime,
		uid:   dn.uid,
		gid:   dn.gid,
		nlink: 0,
	}

	dn.mu.RUnlock()

	return fst
}

// dirEntries returns a slice of fs.DirEntry from a directory ordered by name.
func (dn *dirNode) dirEntries() []fs.DirEntry {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	entries := make([]fs.DirEntry, l)
	i := 0

	for name, nd := range dn.children {
		entries[i] = nd.fillStatFrom(name)
		i++
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return entries
}

// dirNames returns a slice of file names from a directory ordered by name.
func (dn *dirNode) dirNames() []string {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	names := make([]string, l)
	i := 0

	for name := range dn.children {
		names[i] = name
		i++
	}

	sort.Strings(names)

	return names
}

// setMode sets the permissions of the directory node.
func (dn *dirNode) setMode(mode fs.FileMode, u avfs.UserReader) bool {
	if dn.uid != u.Uid() && !u.IsAdmin() {
		return false
	}

	dn.mode &^= avfs.FileModeMask
	dn.mode |= mode & avfs.FileModeMask

	return true
}

// size returns the size of the dirNode : number of children.
func (dn *dirNode) size() int64 {
	return int64(len(dn.children))
}

// fileNode

// delete removes all information from the node, decrements the reference counter of the fileNode.
// If there is no more references, the data is deleted.
func (fn *fileNode) delete() {
	fn.nlink--
	if fn.nlink == 0 {
		fn.data = nil
	}
}

// fillStatFrom returns a MemInfo (implementation of fs.FileInfo) from a fileNode fn named name.
func (fn *fileNode) fillStatFrom(name string) *MemInfo {
	fn.mu.RLock()

	fst := &MemInfo{
		id:    fn.id,
		name:  name,
		size:  fn.size(),
		mode:  fn.mode,
		mtime: fn.mtime,
		uid:   fn.uid,
		gid:   fn.gid,
		nlink: fn.nlink,
	}

	fn.mu.RUnlock()

	return fst
}

// setMode sets the permissions of the file node.
func (fn *fileNode) setMode(mode fs.FileMode, u avfs.UserReader) bool {
	if fn.uid != u.Uid() && !u.IsAdmin() {
		return false
	}

	fn.mode &^= avfs.FileModeMask
	fn.mode |= mode & avfs.FileModeMask

	return true
}

// size returns the size of the file.
func (fn *fileNode) size() int64 {
	return int64(len(fn.data))
}

// truncate truncates the file.
func (fn *fileNode) truncate(size int64) {
	if size == 0 {
		fn.data = nil

		return
	}

	diff := int(size) - len(fn.data)
	if diff > 0 {
		fn.data = append(fn.data, bytes.Repeat([]byte{0}, diff)...)

		return
	}

	fn.data = fn.data[:size]
}

// symlinkNode

// delete removes all information from the node.
func (sn *symlinkNode) delete() {
	sn.link = ""
}

// fillStatFrom returns a MemInfo (implementation of fs.FileInfo) from a symlinkNode named name.
func (sn *symlinkNode) fillStatFrom(name string) *MemInfo {
	sn.mu.RLock()

	fst := &MemInfo{
		name:  name,
		size:  sn.size(),
		mode:  sn.mode,
		mtime: sn.mtime,
		uid:   sn.uid,
		gid:   sn.gid,
		nlink: 0,
	}

	sn.mu.RUnlock()

	return fst
}

// setMode sets the permissions of the symlink node.
func (sn *symlinkNode) setMode(mode fs.FileMode, u avfs.UserReader) bool {
	return false
}

func (sn *symlinkNode) size() int64 {
	return 1
}
