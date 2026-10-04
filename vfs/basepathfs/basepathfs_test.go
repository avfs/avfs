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

//go:build !avfs_race

package basepathfs_test

import (
	"io"
	"strings"
	"syscall"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/basepathfs"
	"github.com/avfs/avfs/vfs/memfs"
)

var (
	// Ensures that basepathfs.BasePathFS implements avfs.VFS interface.
	_ avfs.VFS = &basepathfs.BasePathFS{}

	// Ensures that basepathfs.BasePathFS implements avfs.VFSBase interface.
	_ avfs.VFSBase = &basepathfs.BasePathFS{}

	// Ensures that basepathfs.BasePathFile implements avfs.File interface.
	_ avfs.File = &basepathfs.BasePathFile{}

	// Ensures that basepathfs.BasePathFile implements the io.ReaderFrom interface.
	_ io.ReaderFrom = &basepathfs.BasePathFile{}

	// Ensures that basepathfs.BasePathFile implements the io.WriterTo interface.
	_ io.WriterTo = &basepathfs.BasePathFile{}

	// Ensures that basepathfs.BasePathFile implements the syscall.Conn interface.
	_ syscall.Conn = &basepathfs.BasePathFile{}
)

func initFS(tb testing.TB) (vfs *basepathfs.BasePathFS, basePath string) {
	baseFS, err := memfs.New()
	if err != nil {
		tb.Fatalf("Can't create base file system : %v", err)
	}

	basePath = avfs.FromUnixPath(baseFS, "/base/testpath")

	err = baseFS.MkdirAll(basePath, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("Can't create base directory %s : %v", basePath, err)
	}

	dirs := avfs.SystemDirs(baseFS)

	err = avfs.MkDirs(baseFS, dirs, basePath)
	if err != nil {
		tb.Fatalf("Can't create system directories %v", err)
	}

	vfs, err = basepathfs.New(baseFS, basePath)
	if err != nil {
		tb.Fatalf("Can't create base path file system : %v", err)
	}

	return vfs, basePath
}

func initTest(t *testing.T) *test.Suite {
	vfs, _ := initFS(t)
	ts := test.NewSuiteFS(t, vfs, vfs)

	return ts
}

func TestBasePathFS(t *testing.T) {
	ts := initTest(t)
	ts.TestVFSAll(t)
}

// TestBasePathFsOptions tests BasePathFS configuration options.
func TestBasePathFSOptions(t *testing.T) {
	vfs, err := memfs.New()
	test.RequireNoError(t, err, "New")

	nonExistingDir := avfs.FromUnixPath(vfs, "/non/existing/dir")

	_, err = basepathfs.New(vfs, nonExistingDir)
	test.RequireError(t, err, "New nonExistingDir")

	existingFile := vfs.Join(vfs.TempDir(), "existing")

	err = vfs.WriteFile(existingFile, []byte{}, avfs.DefaultFilePerm)
	test.RequireNoError(t, err, "WriteFile %s", existingFile)

	_, err = basepathfs.New(vfs, existingFile)
	test.RequireError(t, err, "New existingFile")
}

func TestBasePathFSFeatures(t *testing.T) {
	baseFS, err := memfs.New()
	test.RequireNoError(t, err, "New")

	vfs, err := basepathfs.New(baseFS, "/")
	test.RequireNoError(t, err, "New basePathFS")

	if vfs.HasFeature(avfs.FeatSymlink) {
		t.Errorf("Features : want FeatSymlink missing, got present")
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}

	mfs, err := memfs.New()
	test.RequireNoError(t, err, "New")

	vfs, err = basepathfs.New(mfs, "/")
	test.RequireNoError(t, err, "New basePathFS")

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

func TestBasePathFSOSType(t *testing.T) {
	vfsBase, err := memfs.New()
	test.RequireNoError(t, err, "New")

	vfs, err := basepathfs.New(vfsBase, vfsBase.TempDir())
	test.RequireNoError(t, err, "New basePathFS")

	osType := vfs.OSType()
	if osType != vfsBase.OSType() {
		t.Errorf("OSType : want os type to be %v, got %v", vfsBase.OSType(), osType)
	}
}

func TestBasePathFSToBasePath(t *testing.T) {
	vfs, basePath := initFS(t)

	toTests := []struct{ path, result string }{
		{path: "", result: basePath},
		{path: "/", result: basePath},
		{path: "/tmp", result: basePath + "/tmp"},
		{path: "/tmp/avfs", result: basePath + "/tmp/avfs"},
	}

	for _, tt := range toTests {
		path := avfs.FromUnixPath(vfs, tt.path)

		want := avfs.FromUnixPath(vfs, tt.result)
		got := vfs.ToBasePath(path)

		if got != want {
			t.Errorf("ToBasePath %s : want path to be %s, got %s", path, want, got)
		}
	}
}

func TestBasePathFSFromBasePath(t *testing.T) {
	vfs, basePath := initFS(t)

	fromTests := []struct{ path, result string }{
		{path: "/another/path", result: ""},
		{path: basePath, result: "/"},
		{path: basePath + "/tmp", result: "/tmp"},
		{path: basePath + "/tmp/avfs", result: "/tmp/avfs"},
	}

	for _, ft := range fromTests {
		path := avfs.FromUnixPath(vfs, ft.path)

		if !strings.HasPrefix(path, basePath) {
			test.AssertPanic(t, "", func() {
				path = vfs.FromBasePath(path)
			})
		} else {
			want := avfs.FromUnixPath(vfs, ft.result)
			got := vfs.FromBasePath(path)

			if got != want {
				t.Errorf("FromBasePath %s : want path to be %s, got %s", ft.path, want, got)
			}
		}
	}
}
