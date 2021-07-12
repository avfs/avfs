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
	"github.com/avfs/avfs/vfsutils"
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
//  ErrNoSuchFileOrDir when the node is not found
//  ErrFileExists when the node is a file or directory
//  ErrPermDenied when the current user doesn't have permissions on one of the nodes on the path
//  ErrNotADirectory when a file node is found while the path segmentation is not finished
//  ErrTooManySymlinks when more than slCountMax symbolic link resolutions have been performed.
func (vfs *MemFS) searchNode(path string, slMode slMode) ( //nolint:gocritic // consider to simplify the function
	parent *dirNode, child node, absPath string, start, end int, err error) {
	absPath = path
	if !vfs.HasFeature(avfs.FeatAbsPath) {
		absPath, _ = vfsutils.Abs(vfs, path)
	}

	rootNode := vfs.rootNode
	parent = rootNode
	slCount := 0
	slResolved := false

	isLast := len(absPath) <= 1
	for start, end = 1, 0; !isLast; start = end + 1 {
		end, isLast = vfsutils.SegmentPath(absPath, start)
		name := absPath[start:end]

		parent.mu.RLock()
		child = parent.child(name)
		parent.mu.RUnlock()

		if child == nil {
			err = avfs.ErrNoSuchFileOrDir

			return
		}

		switch c := child.(type) {
		case *dirNode:
			if isLast {
				err = avfs.ErrFileExists

				return
			}

			c.mu.RLock()
			ok := c.checkPermission(avfs.PermLookup, vfs.user)
			c.mu.RUnlock()

			if !ok {
				err = avfs.ErrPermDenied

				return
			}

			parent = c

		case *fileNode:
			// File permissions are checked by the calling function.
			if isLast {
				err = avfs.ErrFileExists

				return
			}

			err = avfs.ErrNotADirectory

			return

		case *symlinkNode:
			// Symlinks mode is always 0o777, no need to check permissions.
			slCount++
			if slCount > slCountMax {
				err = avfs.ErrTooManySymlinks

				return
			}

			if isLast {
				if slMode == slmLstat {
					err = avfs.ErrFileExists

					return
				}

				// if the last part of the path is a symbolic link
				// Stat should return the link and not the resolved path.
				if slMode == slmStat && !slResolved {
					slResolved = true

					defer func(ap string, s, e int) {
						absPath, start, end = ap, s, e
					}(absPath, start, end)
				}
			}

			link := c.link
			if vfsutils.IsAbs(link) {
				absPath = vfsutils.Join(link, absPath[end:])
			} else {
				absPath = vfsutils.Join(absPath[:start], link, absPath[end:])
			}

			parent = rootNode
			end = 0
			isLast = len(absPath) <= 1
		}
	}

	return parent, parent, absPath, 1, 1, avfs.ErrFileExists
}

// createDir creates a new directory.
func (vfs *MemFS) createDir(parent *dirNode, name string, perm fs.FileMode) *dirNode {
	child := &dirNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  fs.ModeDir | (perm & avfs.FileModeMask &^ fs.FileMode(vfs.memAttrs.umask)),
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
			mode:  perm & avfs.FileModeMask &^ fs.FileMode(vfs.memAttrs.umask),
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

// base returns the baseNode.
func (bn *baseNode) base() *baseNode {
	return bn
}

// checkPermission checks if the current user has the desired permissions (perm) on the node.
func (bn *baseNode) checkPermission(perm avfs.PermMode, u avfs.UserReader) bool {
	if u.IsRoot() {
		return true
	}

	mode := bn.mode

	switch {
	case bn.uid == u.Uid():
		mode >>= 6
	case bn.gid == u.Gid():
		mode >>= 3
	}

	perm &= avfs.PermRWX

	return avfs.PermMode(mode)&perm == perm
}

// permMode returns the access mode of the node bn.
func (bn *baseNode) permMode(u avfs.UserReader) avfs.PermMode {
	if u.IsRoot() {
		return avfs.PermRWX
	}

	mode := bn.mode

	switch {
	case bn.uid == u.Uid():
		mode >>= 6
	case bn.gid == u.Gid():
		mode >>= 3
	}

	return avfs.PermMode(mode)
}

// setModTime sets the modification time of the node.
func (bn *baseNode) setModTime(mtime time.Time, u avfs.UserReader) error {
	if bn.uid != u.Uid() && !u.IsRoot() {
		return avfs.ErrOpNotPermitted
	}

	bn.mtime = mtime.UnixNano()

	return nil
}

// setOwner sets the owner of the node.
func (bn *baseNode) setOwner(uid, gid int) {
	bn.uid = uid
	bn.gid = gid
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

// child returns the child node named name from the parent node dn.
// it returns nil if the child is not found or if there is no children.
func (dn *dirNode) child(name string) node {
	return dn.children[name]
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

// entries returns a slice of fs.DirEntry from a directory ordered by name.
func (dn *dirNode) entries() []fs.DirEntry {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	entries := make([]fs.DirEntry, l)
	i := 0

	for name, node := range dn.children {
		entries[i] = node.fillStatFrom(name)
		i++
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return entries
}

// names returns a slice of file names from a directory ordered by name.
func (dn *dirNode) names() []string {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	dirNames := make([]string, l)
	i := 0

	for name := range dn.children {
		dirNames[i] = name
		i++
	}

	sort.Strings(dirNames)

	return dirNames
}

// setMode sets the permissions of the directory node.
func (dn *dirNode) setMode(mode fs.FileMode, u avfs.UserReader) error {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	if dn.uid != u.Uid() && !u.IsRoot() {
		return avfs.ErrOpNotPermitted
	}

	dn.mode &^= avfs.FileModeMask
	dn.mode |= mode & avfs.FileModeMask

	return nil
}

// size returns the size of the dirNode : number of children.
func (dn *dirNode) size() int64 {
	return int64(len(dn.children))
}

// fileNode

// deleteData decrements the reference counter of the fileNode.
// if there is no more references, the data is deleted.
func (fn *fileNode) deleteData() {
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
func (fn *fileNode) setMode(mode fs.FileMode, u avfs.UserReader) error {
	fn.mu.Lock()
	defer fn.mu.Unlock()

	if fn.uid != u.Uid() && !u.IsRoot() {
		return avfs.ErrPermDenied
	}

	fn.mode &^= avfs.FileModeMask
	fn.mode |= mode & avfs.FileModeMask

	return nil
}

// size returns the size of the file.
func (fn *fileNode) size() int64 {
	return int64(len(fn.data))
}

// Size returns the size of the file.
func (fn *fileNode) Size() int64 {
	fn.mu.RLock()
	s := fn.size()
	fn.mu.RUnlock()

	return s
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

// fillStatFrom returns a MemInfo (implementation of fs.FileInfo) from a symlinkNode dn named name.
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
func (sn *symlinkNode) setMode(mode fs.FileMode, u avfs.UserReader) error {
	return avfs.ErrOpNotPermitted
}

func (sn *symlinkNode) size() int64 {
	return 1
}

// removeStack

func (rs *removeStack) push(parent *dirNode) {
	rs.stack = append(rs.stack, parent)
}

func (rs *removeStack) pop() *dirNode {
	n := len(rs.stack) - 1
	parent := rs.stack[n]
	rs.stack = rs.stack[:n]

	return parent
}

func (rs *removeStack) len() int {
	return len(rs.stack)
}
