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
	"sort"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
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
func (fs *MemFs) searchNode(path string, slMode slMode) (
	parent *dirNode, child node, absPath string, start, end int, err error) {
	absPath = path
	if !fs.HasFeature(avfs.FeatAbsPath) {
		absPath, _ = fsutil.Abs(fs, path)
	}

	parent = fs.rootNode
	slCount := 0
	slResolved := false

	isLast := len(absPath) <= 1
	for start, end = 1, 0; !isLast; start = end + 1 {
		end, isLast = fsutil.SegmentPath(absPath, start)
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

			if !c.checkPermissionLck(avfs.WantLookup, fs.user) {
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
			if fsutil.IsAbs(link) {
				absPath = fsutil.Join(link, absPath[end:])
			} else {
				absPath = fsutil.Join(absPath[:start], link, absPath[end:])
			}

			parent = fs.rootNode
			end = 0
			isLast = len(absPath) <= 1
		}
	}

	return parent, parent, absPath, 1, 1, avfs.ErrFileExists
}

// createDir creates a new directory.
func (fs *MemFs) createDir(parent *dirNode, name string, perm os.FileMode) *dirNode {
	child := &dirNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  os.ModeDir | (perm & os.ModePerm &^ os.FileMode(fs.umask)),
			uid:   fs.user.Uid(),
			gid:   fs.user.Gid(),
		},
		children: nil,
	}

	parent.addChild(name, child)

	return child
}

// createFile creates a new file.
func (fs *MemFs) createFile(parent *dirNode, name string, perm os.FileMode) *fileNode {
	child := &fileNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  perm & os.ModePerm &^ os.FileMode(fs.umask),
			uid:   fs.user.Uid(),
			gid:   fs.user.Gid(),
		},
		refCount: 1,
	}

	parent.addChild(name, child)

	return child
}

// createSymlink creates a new symlink.
func (fs *MemFs) createSymlink(parent *dirNode, name, link string) *symlinkNode {
	child := &symlinkNode{
		baseNode: baseNode{
			mtime: time.Now().UnixNano(),
			mode:  os.ModeSymlink | os.ModePerm,
			uid:   fs.user.Uid(),
			gid:   fs.user.Gid(),
		},
		link: link,
	}

	parent.addChild(name, child)

	return child
}

// baseNode

// checkPermissionLck checks if the current user has the wanted permissions (want)
// on the node using a read lock on the node.
func (bn *baseNode) checkPermissionLck(want avfs.WantMode, u avfs.UserReader) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()

	return bn.checkPermission(want, u)
}

// checkPermissionLck checks if the current user has the wanted permissions (want) on the node.
func (bn *baseNode) checkPermission(want avfs.WantMode, u avfs.UserReader) bool {
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

	want &= avfs.WantRWX

	return avfs.WantMode(mode)&want == want
}

// setModTime sets the modification time of the node.
func (bn *baseNode) setModTime(mtime time.Time) {
	bn.mu.Lock()
	bn.mtime = mtime.UnixNano()
	bn.mu.Unlock()
}

// setOwner sets the owner of the node.
func (bn *baseNode) setOwner(uid, gid int) {
	bn.mu.Lock()
	bn.uid = uid
	bn.gid = gid
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

// child returns the child node named name from the parent node dn.
// it returns nil if the child is not found or if there is no children.
func (dn *dirNode) child(name string) node {
	return dn.children[name]
}

// fillStatFrom returns a fStat (implementation of os.FileInfo) from a dirNode dn named name.
func (dn *dirNode) fillStatFrom(name string) fStat {
	dn.mu.RLock()

	fst := fStat{
		name:  name,
		size:  dn.size(),
		mode:  dn.mode,
		mtime: dn.mtime,
		uid:   dn.uid,
		gid:   dn.gid,
	}

	dn.mu.RUnlock()

	return fst
}

// infos returns a slice of FileInfo from a directory ordered by name.
func (dn *dirNode) infos() []os.FileInfo {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	dirInfos := make([]os.FileInfo, 0, l)
	for name, node := range dn.children {
		dirInfos = append(dirInfos, node.fillStatFrom(name))
	}

	sort.Slice(dirInfos, func(i, j int) bool { return dirInfos[i].Name() < dirInfos[j].Name() })

	return dirInfos
}

// names returns a slice of file names from a directory ordered by name.
func (dn *dirNode) names() []string {
	l := len(dn.children)
	if l == 0 {
		return nil
	}

	dirNames := make([]string, 0, l)
	for name := range dn.children {
		dirNames = append(dirNames, name)
	}

	sort.Strings(dirNames)

	return dirNames
}

// setMode sets the permissions of the directory node.
func (dn *dirNode) setMode(mode os.FileMode, u avfs.UserReader) error {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	if dn.uid != u.Uid() && !u.IsRoot() {
		return avfs.ErrOpNotPermitted
	}

	dn.mode = (dn.mode &^ os.ModePerm) | (mode & os.ModePerm)

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
	fn.refCount--
	if fn.refCount == 0 {
		fn.data = nil
	}
}

// fillStatFrom returns a fStat (implementation of os.FileInfo) from a fileNode fn named name.
func (fn *fileNode) fillStatFrom(name string) fStat {
	fn.mu.RLock()

	fst := fStat{
		name:  name,
		size:  fn.size(),
		mode:  fn.mode,
		mtime: fn.mtime,
		uid:   fn.uid,
		gid:   fn.gid,
	}

	fn.mu.RUnlock()

	return fst
}

// setMode sets the permissions of the file node.
func (fn *fileNode) setMode(mode os.FileMode, u avfs.UserReader) error {
	fn.mu.Lock()
	defer fn.mu.Unlock()

	if fn.uid != u.Uid() && !u.IsRoot() {
		return avfs.ErrPermDenied
	}

	fn.mode = (fn.mode &^ os.ModePerm) | (mode & os.ModePerm)

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
	if fn.data == nil {
		return
	}

	if size == 0 {
		fn.data = nil
		return
	}

	fn.data = fn.data[:size]
}

// symlinkNode

// fillStatFrom returns a fStat (implementation of os.FileInfo) from a symlinkNode dn named name.
func (sn *symlinkNode) fillStatFrom(name string) fStat {
	sn.mu.RLock()

	fst := fStat{
		name:  name,
		size:  sn.size(),
		mode:  sn.mode,
		mtime: sn.mtime,
		uid:   sn.uid,
		gid:   sn.gid,
	}

	sn.mu.RUnlock()

	return fst
}

// setMode sets the permissions of the symlink node.
func (sn *symlinkNode) setMode(mode os.FileMode, u avfs.UserReader) error {
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

// fStat is the implementation of FileInfo returned by Stat and Lstat.

// IsDir is the abbreviation for Mode().IsDir().
func (fst fStat) IsDir() bool {
	return fst.mode.IsDir()
}

// Mode returns the file mode bits.
func (fst fStat) Mode() os.FileMode {
	return fst.mode
}

// ModTime returns the modification time.
func (fst fStat) ModTime() time.Time {
	return time.Unix(0, fst.mtime)
}

// Type returns the base name of the file.
func (fst fStat) Name() string {
	return fst.name
}

// Size returns the length in bytes for regular files; system-dependent for others.
func (fst fStat) Size() int64 {
	return fst.size
}

// Sys returns the underlying data source (can return nil).
func (fst fStat) Sys() interface{} {
	return &avfs.StatT{
		Uid: uint32(fst.uid),
		Gid: uint32(fst.gid),
	}
}
