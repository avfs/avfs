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
	"sort"
	"time"

	"github.com/avfs/avfs"
)

// split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func split(path string) (dir, file string) {
	i := len(path) - 1
	for i >= 0 && !os.IsPathSeparator(path[i]) {
		i--
	}

	if i == 0 {
		return path[:1], path[1:]
	}

	return path[:i], path[i+1:]
}

// addChild adds a child to a node.
func (nd *node) addChild(name string, child *node) {
	nd.mu.Lock()

	if nd.children == nil {
		nd.children = make(children)
	}

	nd.children[name] = child

	nd.mu.Unlock()
}

// createDir creates a new directory.
func (fs *OrefaFs) createDir(parent *node, absPath, fileName string, perm os.FileMode) *node {
	mode := os.ModeDir | (perm & avfs.FileModeMask &^ os.FileMode(fs.umask))

	return fs.createNode(parent, absPath, fileName, mode)
}

// createFile creates a new file.
func (fs *OrefaFs) createFile(parent *node, absPath, fileName string, perm os.FileMode) *node {
	mode := perm & avfs.FileModeMask &^ os.FileMode(fs.umask)

	return fs.createNode(parent, absPath, fileName, mode)
}

// createNode creates a new node (directory or file).
func (fs *OrefaFs) createNode(parent *node, absPath, fileName string, mode os.FileMode) *node {
	nd := &node{
		mtime: time.Now().UnixNano(),
		mode:  mode,
		nlink: 1,
	}

	parent.addChild(fileName, nd)

	fs.mu.Lock()
	fs.nodes[absPath] = nd
	fs.mu.Unlock()

	return nd
}

// fillStatFrom returns a fStat (implementation of os.FileInfo) from a dirNode dn named name.
func (nd *node) fillStatFrom(name string) fStat {
	nd.mu.RLock()

	fst := fStat{
		name:  name,
		size:  nd.size(),
		mode:  nd.mode,
		mtime: nd.mtime,
	}

	nd.mu.RUnlock()

	return fst
}

// infos returns a slice of FileInfo from a directory ordered by name.
func (nd *node) infos() []os.FileInfo {
	l := len(nd.children)
	if l == 0 {
		return nil
	}

	dirInfos := make([]os.FileInfo, 0, l)
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
func (nd *node) setMode(mode os.FileMode) {
	nd.mu.Lock()

	nd.mode &^= avfs.FileModeMask
	nd.mode |= mode & avfs.FileModeMask

	nd.mu.Unlock()
}

// setModTime sets the modification time of the node.
func (nd *node) setModTime(mtime time.Time) {
	nd.mu.Lock()
	nd.mtime = mtime.UnixNano()
	nd.mu.Unlock()
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
	if nd.data == nil {
		return
	}

	if size == 0 {
		nd.data = nil
		return
	}

	nd.data = nd.data[:size]
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
	return nil
}
