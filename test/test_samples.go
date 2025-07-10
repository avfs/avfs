//
//  Copyright 2023 The AVFS authors
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

package test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
)

// dirInfo contains the sample directories.
type dirInfo struct {
	Path      string
	WantModes []fs.FileMode
	Mode      fs.FileMode
}

// sampleDirs returns the sample directories used by Mkdir function.
func (ts *Suite) sampleDirs(testDir string) []*dirInfo {
	dirs := []*dirInfo{
		{Path: "/A", Mode: 0o777, WantModes: []fs.FileMode{0o777}},
		{Path: "/B", Mode: 0o755, WantModes: []fs.FileMode{0o755}},
		{Path: "/B/1", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o755}},
		{Path: "/B/1/D", Mode: 0o700, WantModes: []fs.FileMode{0o755, 0o755, 0o700}},
		{Path: "/B/1/E", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o755, 0o755}},
		{Path: "/B/2", Mode: 0o750, WantModes: []fs.FileMode{0o755, 0o750}},
		{Path: "/B/2/F", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o750, 0o755}},
		{Path: "/B/2/F/3", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755}},
		{Path: "/B/2/F/3/G", Mode: 0o777, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755, 0o777}},
		{Path: "/B/2/F/3/G/4", Mode: 0o777, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755, 0o777, 0o777}},
		{Path: "/C", Mode: 0o750, WantModes: []fs.FileMode{0o750}},
		{Path: "/C/5", Mode: 0o750, WantModes: []fs.FileMode{0o750, 0o750}},
	}

	for i, dir := range dirs {
		dirs[i].Path = ts.vfsTest.Join(testDir, dir.Path)
	}

	return dirs
}

// sampleDirsAll returns the sample directories used by MkdirAll function.
func (ts *Suite) sampleDirsAll(testDir string) []*dirInfo {
	dirs := []*dirInfo{
		{Path: "/H/6", Mode: 0o750, WantModes: []fs.FileMode{0o750, 0o750}},
		{Path: "/H/6/I/7", Mode: 0o755, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755}},
		{Path: "/H/6/I/7/J/8", Mode: 0o777, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755, 0o777, 0o777}},
	}

	for i, dir := range dirs {
		dirs[i].Path = ts.vfsTest.Join(testDir, dir.Path)
	}

	return dirs
}

// createSampleDirs creates and returns sample directories and testDir directory if necessary.
func (ts *Suite) createSampleDirs(tb testing.TB, testDir string) []*dirInfo {
	vfs := ts.vfsSetup

	err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
	RequireNoError(tb, err, "MkdirAll %s", testDir)

	dirs := ts.sampleDirs(testDir)
	for _, dir := range dirs {
		err = vfs.Mkdir(dir.Path, dir.Mode)
		RequireNoError(tb, err, "Mkdir %s", dir.Path)
	}

	return dirs
}

// fileInfo contains the sample files.
type fileInfo struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

// sampleFiles returns the sample files.
func (ts *Suite) sampleFiles(testDir string) []*fileInfo {
	files := []*fileInfo{
		{Path: "/file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("file")},
		{Path: "/A/afile1.txt", Mode: 0o777, Content: []byte("afile1")},
		{Path: "/A/afile2.txt", Mode: avfs.DefaultFilePerm, Content: []byte("afile2")},
		{Path: "/A/afile3.txt", Mode: 0o600, Content: []byte("afile3")},
		{Path: "/B/1/1file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("1file")},
		{Path: "/B/1/E/efile.txt", Mode: avfs.DefaultFilePerm, Content: []byte("efile")},
		{Path: "/B/2/F/3/3file1.txt", Mode: 0o640, Content: []byte("3file1")},
		{Path: "/B/2/F/3/3file2.txt", Mode: avfs.DefaultFilePerm, Content: []byte("3file2")},
		{Path: "/B/2/F/3/G/4/4file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("4file")},
		{Path: "/C/cfile.txt", Mode: avfs.DefaultFilePerm, Content: []byte("cfile")},
	}

	for i, file := range files {
		files[i].Path = ts.vfsTest.Join(testDir, file.Path)
	}

	return files
}

// createSampleFiles creates and returns the sample files.
func (ts *Suite) createSampleFiles(tb testing.TB, testDir string) []*fileInfo {
	vfs := ts.vfsSetup

	files := ts.sampleFiles(testDir)
	for _, file := range files {
		err := vfs.WriteFile(file.Path, file.Content, file.Mode)
		RequireNoError(tb, err, "WriteFile %s", file.Path)
	}

	return files
}

// symlinkInfo contains sample symbolic links.
type symlinkInfo struct {
	NewPath string
	OldPath string
}

// sampleSymlinks returns the sample symbolic links.
func (ts *Suite) sampleSymlinks(testDir string) []*symlinkInfo {
	vfs := ts.vfsTest
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sls := []*symlinkInfo{
		{NewPath: "/A/lroot", OldPath: "/"},
		{NewPath: "/lC", OldPath: "/C"},
		{NewPath: "/B/1/lafile2.txt", OldPath: "/A/afile2.txt"},
		{NewPath: "/B/2/lf", OldPath: "/B/2/F"},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/lf"},
		{NewPath: "/C/lllf", OldPath: "/B/2/F/3/llf"},
		{NewPath: "/A/l3file2.txt", OldPath: "/C/lllf/3/3file2.txt"},
		{NewPath: "/C/lNonExist", OldPath: "/A/path/to/a/non/existing/file"},
	}

	for i, sl := range sls {
		sls[i].NewPath = vfs.Join(testDir, sl.NewPath)
		sls[i].OldPath = vfs.Join(testDir, sl.OldPath)
	}

	return sls
}

// symlinkEvalInfo contains the data to evaluate the sample symbolic links.
type symlinkEvalInfo struct {
	NewPath   string      // Name of the symbolic link.
	OldPath   string      // Value of the symbolic link.
	IsSymlink bool        //
	Mode      fs.FileMode //
}

// sampleSymlinksEval returns the sample symbolic links to evaluate.
func (ts *Suite) sampleSymlinksEval(testDir string) []*symlinkEvalInfo {
	vfs := ts.vfsTest
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sles := []*symlinkEvalInfo{
		{NewPath: "/A/lroot/", OldPath: "/", IsSymlink: true, Mode: fs.ModeDir | 0o777},
		{NewPath: "/A/lroot/B", OldPath: "/B", IsSymlink: false, Mode: fs.ModeDir | 0o755},
		{NewPath: "/", OldPath: "/", IsSymlink: false, Mode: fs.ModeDir | 0o777},
		{NewPath: "/lC", OldPath: "/C", IsSymlink: true, Mode: fs.ModeDir | 0o750},
		{NewPath: "/B/1/lafile2.txt", OldPath: "/A/afile2.txt", IsSymlink: true, Mode: 0o644},
		{NewPath: "/B/2/lf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/C/lllf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/C/lllf/3/3file1.txt", OldPath: "/B/2/F/3/3file1.txt", IsSymlink: false, Mode: 0o640},
	}

	for i, sle := range sles {
		sles[i].NewPath = vfs.Join(testDir, sle.NewPath)
		sles[i].OldPath = vfs.Join(testDir, sle.OldPath)
	}

	return sles
}

// createSampleSymlinks creates the sample symbolic links.
func (ts *Suite) createSampleSymlinks(tb testing.TB, testDir string) []*symlinkInfo {
	vfs := ts.vfsSetup

	symlinks := ts.sampleSymlinks(testDir)
	for _, sl := range symlinks {
		err := vfs.Symlink(sl.OldPath, sl.NewPath)
		RequireNoError(tb, err, "Symlink %s %s", sl.OldPath, sl.NewPath)
	}

	return symlinks
}
