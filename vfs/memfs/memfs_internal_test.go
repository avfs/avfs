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

//go:build !datarace

package memfs

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
)

var (
	// Tests that dirNode struct implements node interface.
	_ node = &dirNode{}

	// Tests that fileNode struct implements node interface.
	_ node = &fileNode{}

	// Tests that symlinkNode struct implements node interface.
	_ node = &symlinkNode{}

	// Tests that MemInfo struct implements fs.FileInfo interface.
	_ fs.FileInfo = &MemInfo{}
)

func TestSearchNode(t *testing.T) {
	vfs := New()
	rn := vfs.rootNode

	// Directories
	da := vfs.createDir(rn, "a", avfs.DefaultDirPerm)
	db := vfs.createDir(rn, "b", avfs.DefaultDirPerm)
	dc := vfs.createDir(rn, "c", avfs.DefaultDirPerm)
	da1 := vfs.createDir(da, "a1", avfs.DefaultDirPerm)
	da2 := vfs.createDir(da, "a2", avfs.DefaultDirPerm)
	db1 := vfs.createDir(db, "b1", avfs.DefaultDirPerm)
	db1a := vfs.createDir(db1, "b1A", avfs.DefaultDirPerm)
	db1b := vfs.createDir(db1, "b1B", avfs.DefaultDirPerm)

	// Files
	f1 := vfs.createFile(rn, "file1", avfs.DefaultFilePerm)
	fa1 := vfs.createFile(da, "afile1", avfs.DefaultFilePerm)
	fa2 := vfs.createFile(da, "afile2", avfs.DefaultFilePerm)
	fa3 := vfs.createFile(da, "afile3", avfs.DefaultFilePerm)

	// Symlinks
	vfs.createSymlink(rn, "lroot", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "/"))
	vfs.createSymlink(rn, "la", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "/a"))
	vfs.createSymlink(db1b, "lb1", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "/b/b1"))
	vfs.createSymlink(dc, "lafile3", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "../a/afile3"))
	lloop1 := vfs.createSymlink(rn, "loop1", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "/loop2"))
	vfs.createSymlink(rn, "loop2", vfs.Utils.FromUnixPath(avfs.DefaultVolume, "/loop1"))

	cases := []struct { //nolint:govet // no fieldalignment for test structs
		path        string
		parent      *dirNode
		child       node
		first, rest string
		err         error
	}{
		// Existing directories
		{path: "/", parent: rn, child: rn, first: "", rest: "", err: vfs.err.FileExists},
		{path: "/a", parent: rn, child: da, first: "a", rest: "", err: vfs.err.FileExists},
		{path: "/b", parent: rn, child: db, first: "b", rest: "", err: vfs.err.FileExists},
		{path: "/c", parent: rn, child: dc, first: "c", rest: "", err: vfs.err.FileExists},
		{path: "/a/a1", parent: da, child: da1, first: "a1", rest: "", err: vfs.err.FileExists},
		{path: "/a/a2", parent: da, child: da2, first: "a2", rest: "", err: vfs.err.FileExists},
		{path: "/b/b1", parent: db, child: db1, first: "b1", rest: "", err: vfs.err.FileExists},
		{path: "/b/b1/b1A", parent: db1, child: db1a, first: "b1A", rest: "", err: vfs.err.FileExists},
		{path: "/b/b1/b1B", parent: db1, child: db1b, first: "b1B", rest: "", err: vfs.err.FileExists},

		// Existing files
		{path: "/file1", parent: rn, child: f1, first: "file1", rest: "", err: vfs.err.FileExists},
		{path: "/a/afile1", parent: da, child: fa1, first: "afile1", rest: "", err: vfs.err.FileExists},
		{path: "/a/afile2", parent: da, child: fa2, first: "afile2", rest: "", err: vfs.err.FileExists},
		{path: "/a/afile3", parent: da, child: fa3, first: "afile3", rest: "", err: vfs.err.FileExists},

		// Non existing
		{path: "/z", parent: rn, first: "z", rest: "", err: vfs.err.NoSuchFile},
		{path: "/a/az", parent: da, first: "az", rest: "", err: vfs.err.NoSuchFile},
		{path: "/b/b1/b1z", parent: db1, first: "b1z", err: vfs.err.NoSuchFile},
		{path: "/b/b1/b1A/b1Az", parent: db1a, first: "b1Az", err: vfs.err.NoSuchFile},
		{path: "/b/b1/b1A/b1Az/not/exist", parent: db1a, first: "b1Az", rest: "/not/exist", err: vfs.err.NoSuchDir},
		{path: "/a/afile1/not/a/dir", parent: da, child: fa1, first: "afile1", rest: "/not/a/dir", err: vfs.err.NotADirectory},

		// Symlinks
		{path: "/lroot", parent: rn, child: rn, first: "", rest: "", err: vfs.err.FileExists},
		{path: "/lroot/a", parent: rn, child: da, first: "a", rest: "", err: vfs.err.FileExists},
		{path: "/la/a1", parent: da, child: da1, first: "a1", rest: "", err: vfs.err.FileExists},
		{path: "/b/b1/b1B/lb1/b1A", parent: db1, child: db1a, first: "b1A", rest: "", err: vfs.err.FileExists},
		{path: "/c/lafile3", parent: da, child: fa3, first: "afile3", rest: "", err: vfs.err.FileExists},
		{path: "/loop1", parent: rn, child: lloop1, first: "loop1", rest: "", err: vfs.err.TooManySymlinks},
	}

	for _, c := range cases {
		path := vfs.FromUnixPath(avfs.DefaultVolume, c.path)
		wantRest := vfs.FromSlash(c.rest)

		parent, child, pi, err := vfs.searchNode(path, slmEval)
		first := pi.Part()
		rest := pi.Right()

		if c.err != err {
			t.Errorf("%s : want error to be %v, got %v", path, c.err, err)
		}

		if c.parent != parent {
			t.Errorf("%s : want parent to be %v, got %v", path, c.parent, parent)
		}

		if c.child != child {
			t.Errorf("%s : want child to be %v, got %v", path, c.child, child)
		}

		if c.first != first {
			t.Errorf("%s : want first to be %s, got %s", path, c.first, first)
		}

		if wantRest != rest {
			t.Errorf("%s : want rest to be %s, got %s", path, wantRest, rest)
		}
	}
}
