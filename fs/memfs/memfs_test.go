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

package memfs //nolint:testpackage

import (
	"os"
	"testing"

	"github.com/avfs/avfs"
)

var (
	// dirNode struct implements node interface.
	_ node = &dirNode{}

	// fileNode struct implements node interface.
	_ node = &fileNode{}

	// symlinkNode struct implements node interface.
	_ node = &symlinkNode{}

	// fStat struct implements os.FileInfo interface.
	_ os.FileInfo = &fStat{}
)

func TestSearchNode(t *testing.T) {
	fs, err := New()
	if err != nil {
		t.Fatalf("New : want err to be nil, got %v", err)
	}

	rn := fs.rootNode

	// Directories
	da := fs.createDir(rn, "a", avfs.DefaultDirPerm)
	db := fs.createDir(rn, "b", avfs.DefaultDirPerm)
	dc := fs.createDir(rn, "c", avfs.DefaultDirPerm)
	da1 := fs.createDir(da, "a1", avfs.DefaultDirPerm)
	da2 := fs.createDir(da, "a2", avfs.DefaultDirPerm)
	db1 := fs.createDir(db, "b1", avfs.DefaultDirPerm)
	db1a := fs.createDir(db1, "b1A", avfs.DefaultDirPerm)
	db1b := fs.createDir(db1, "b1B", avfs.DefaultDirPerm)

	// Files
	f1 := fs.createFile(rn, "file1", avfs.DefaultFilePerm)
	fa1 := fs.createFile(da, "afile1", avfs.DefaultFilePerm)
	fa2 := fs.createFile(da, "afile2", avfs.DefaultFilePerm)
	fa3 := fs.createFile(da, "afile3", avfs.DefaultFilePerm)

	// Symlinks
	fs.createSymlink(rn, "lroot", "/")
	fs.createSymlink(rn, "la", "/a")
	fs.createSymlink(db1b, "lb1", "/b/b1")
	fs.createSymlink(dc, "lafile3", "../a/afile3")
	lloop1 := fs.createSymlink(rn, "loop1", "/loop2")
	fs.createSymlink(rn, "loop2", "/loop1")

	tests := []struct {
		path        string
		parent      *dirNode
		child       node
		first, rest string
		err         error
	}{
		// Existing directories
		{path: "/", parent: rn, child: rn, first: "", rest: "", err: avfs.ErrFileExists},
		{path: "/a", parent: rn, child: da, first: "a", rest: "", err: avfs.ErrFileExists},
		{path: "/b", parent: rn, child: db, first: "b", rest: "", err: avfs.ErrFileExists},
		{path: "/c", parent: rn, child: dc, first: "c", rest: "", err: avfs.ErrFileExists},
		{path: "/a/a1", parent: da, child: da1, first: "a1", rest: "", err: avfs.ErrFileExists},
		{path: "/a/a2", parent: da, child: da2, first: "a2", rest: "", err: avfs.ErrFileExists},
		{path: "/b/b1", parent: db, child: db1, first: "b1", rest: "", err: avfs.ErrFileExists},
		{path: "/b/b1/b1A", parent: db1, child: db1a, first: "b1A", rest: "", err: avfs.ErrFileExists},
		{path: "/b/b1/b1B", parent: db1, child: db1b, first: "b1B", rest: "", err: avfs.ErrFileExists},

		// Existing files
		{path: "/file1", parent: rn, child: f1, first: "file1", rest: "", err: avfs.ErrFileExists},
		{path: "/a/afile1", parent: da, child: fa1, first: "afile1", rest: "", err: avfs.ErrFileExists},
		{path: "/a/afile2", parent: da, child: fa2, first: "afile2", rest: "", err: avfs.ErrFileExists},
		{path: "/a/afile3", parent: da, child: fa3, first: "afile3", rest: "", err: avfs.ErrFileExists},

		// Symlinks
		{path: "/lroot", parent: rn, child: rn, first: "", rest: "", err: avfs.ErrFileExists},
		{path: "/lroot/a", parent: rn, child: da, first: "a", rest: "", err: avfs.ErrFileExists},
		{path: "/la/a1", parent: da, child: da1, first: "a1", rest: "", err: avfs.ErrFileExists},
		{path: "/b/b1/b1B/lb1/b1A", parent: db1, child: db1a, first: "b1A", rest: "", err: avfs.ErrFileExists},
		{path: "/c/lafile3", parent: da, child: fa3, first: "afile3", rest: "", err: avfs.ErrFileExists},
		{path: "/loop1", parent: rn, child: lloop1, first: "loop1", rest: "", err: avfs.ErrTooManySymlinks},

		// Non existing
		{path: "/z", parent: rn, first: "z", rest: "", err: avfs.ErrNoSuchFileOrDir},
		{path: "/a/az", parent: da, first: "az", rest: "", err: avfs.ErrNoSuchFileOrDir},
		{path: "/b/b1/b1z", parent: db1, first: "b1z", err: avfs.ErrNoSuchFileOrDir},
		{path: "/b/b1/b1A/b1Az", parent: db1a, first: "b1Az", err: avfs.ErrNoSuchFileOrDir},
		{path: "/b/b1/b1A/b1Az/not/exist", parent: db1a, first: "b1Az", rest: "/not/exist",
			err: avfs.ErrNoSuchFileOrDir},
		{path: "/a/afile1/not/a/dir", parent: da, child: fa1, first: "afile1", rest: "/not/a/dir",
			err: avfs.ErrNotADirectory},
	}

	for _, test := range tests {
		parent, child, absPath, start, end, err := fs.searchNode(test.path, slmEval)

		first := absPath[start:end]
		rest := absPath[end:]

		if test.err != err {
			t.Errorf("%s : want error to be %v, got %v", test.path, test.err, err)
		}

		if test.parent != parent {
			t.Errorf("%s : want parent to be %v, got %v", test.path, test.parent, parent)
		}

		if test.child != child {
			t.Errorf("%s : want child to be %v, got %v", test.path, test.child, child)
		}

		if test.first != first {
			t.Errorf("%s : want first to be %s, got %s", test.path, test.first, first)
		}

		if test.rest != rest {
			t.Errorf("%s : want rest to be %s, got %s", test.path, test.rest, rest)
		}
	}
}
