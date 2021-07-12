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
	"bytes"
	"io/fs"
	"sort"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func split(path string) (dir, file string) {
	i := len(path) - 1
	for i >= 0 && !vfsutils.IsPathSeparator(path[i]) {
		i--
	}

	if i == 0 {
		return path[:1], path[1:]
	}

	return path[:i], path[i+1:]
}

// addChild adds a child to a node.
func (nd *node) addChild(name string, child *node) {
	if nd.children == nil {
		nd.children = make(children)
	}

	nd.children[name] = child
}

// createDir creates a new directory.
func (vfs *OrefaFS) createDir(parent *node, absPath, fileName string, perm fs.FileMode) *node {
	mode := fs.ModeDir | (perm & avfs.FileModeMask &^ fs.FileMode(vfs.umask))

	return vfs.createNode(parent, absPath, fileName, mode)
}

// createFile creates a new file.
func (vfs *OrefaFS) createFile(parent *node, absPath, fileName string, perm fs.FileMode) *node {
	mode := perm & avfs.FileModeMask &^ fs.FileMode(vfs.umask)

	return vfs.createNode(parent, absPath, fileName, mode)
}

// createNode creates a new node (directory or file).
func (vfs *OrefaFS) createNode(parent *node, absPath, fileName string, mode fs.FileMode) *node {
	nd := &node{
		id:    atomic.AddUint64(&vfs.lastId, 1),
		mtime: time.Now().UnixNano(),
		mode:  mode,
		uid:   vfs.currentUser.Uid(),
		gid:   vfs.currentUser.Gid(),
		nlink: 1,
	}

	parent.mu.Lock()
	parent.addChild(fileName, nd)
	parent.mu.Unlock()

	vfs.mu.Lock()
	vfs.nodes[absPath] = nd
	vfs.mu.Unlock()

	return nd
}

// fillStatFrom returns a OrefaInfo (implementation of fs.FileInfo) from a dirNode dn named name.
func (nd *node) fillStatFrom(name string) *OrefaInfo {
	nd.mu.RLock()

	fst := &OrefaInfo{
		id:    nd.id,
		name:  name,
		size:  nd.size(),
		mode:  nd.mode,
		mtime: nd.mtime,
		uid:   nd.uid,
		gid:   nd.gid,
		nlink: nd.nlink,
	}

	nd.mu.RUnlock()

	return fst
}

// infos returns a slice of FileInfo from a directory ordered by name.
func (nd *node) infos() []fs.FileInfo {
	l := len(nd.children)
	if l == 0 {
		return nil
	}

	dirInfos := make([]fs.FileInfo, 0, l)
	for name, cn := range nd.children {
		dirInfos = append(dirInfos, cn.fillStatFrom(name))
	}

	sort.Slice(dirInfos, func(i, j int) bool { return dirInfos[i].Name() < dirInfos[j].Name() })

	return dirInfos
}

// names returns a slice of file names from a directory ordered by name.
func (nd *node) names() []string {
	l := len(nd.children)
	if l == 0 {
		return nil
	}

	dirNames := make([]string, 0, l)
	for name := range nd.children {
		dirNames = append(dirNames, name)
	}

	sort.Strings(dirNames)

	return dirNames
}

// remove deletes the content of a node.
func (nd *node) remove() {
	nd.children = nil

	nd.nlink--
	if nd.nlink == 0 {
		nd.data = nil
	}
}

// setMode sets the permissions of the file node.
func (nd *node) setMode(mode fs.FileMode) {
	nd.mode &^= avfs.FileModeMask
	nd.mode |= mode & avfs.FileModeMask
}

// setModTime sets the modification time of the node.
func (nd *node) setModTime(mtime time.Time) {
	nd.mtime = mtime.UnixNano()
}

// setOwner sets the user and group id.
func (nd *node) setOwner(uid, gid int) {
	nd.uid = uid
	nd.gid = gid
}

// size returns the size of the file.
func (nd *node) size() int64 {
	if nd.mode.IsDir() {
		return int64(len(nd.children))
	}

	return int64(len(nd.data))
}

// Size returns the size of the file.
func (nd *node) Size() int64 {
	nd.mu.RLock()
	s := nd.size()
	nd.mu.RUnlock()

	return s
}

// truncate truncates the file.
func (nd *node) truncate(size int64) {
	if size == 0 {
		nd.data = nil

		return
	}

	diff := int(size) - len(nd.data)
	if diff > 0 {
		nd.data = append(nd.data, bytes.Repeat([]byte{0}, diff)...)

		return
	}

	nd.data = nd.data[:size]
}
