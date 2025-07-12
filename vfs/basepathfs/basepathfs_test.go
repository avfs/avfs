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
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/basepathfs"
	"github.com/avfs/avfs/vfs/memfs"
)

var (
	// Tests that basepathfs.BasePathFS struct implements avfs.VFS interface.
	_ avfs.VFS = &basepathfs.BasePathFS{}

	// Tests that basepathfs.BasePathFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &basepathfs.BasePathFS{}

	// Tests that basepathfs.BasePathFile struct implements avfs.File interface.
	_ avfs.File = &basepathfs.BasePathFile{}
)

func initFS(tb testing.TB) (vfs *basepathfs.BasePathFS, basePath string) {
	baseFS := memfs.New()
	basePath = avfs.FromUnixPath(baseFS, "/base/testpath")

	err := baseFS.MkdirAll(basePath, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("Can't create base directory %s : %v", basePath, err)
	}

	dirs := avfs.SystemDirs(baseFS, basePath)

	err = avfs.MkSystemDirs(baseFS, dirs)
	if err != nil {
		tb.Fatalf("Can't create system directories %v", err)
	}

	vfs = basepathfs.New(baseFS, basePath)

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
	vfs := memfs.New()
	nonExistingDir := avfs.FromUnixPath(vfs, "/non/existing/dir")

	test.AssertPanic(t, "", func() {
		_ = basepathfs.New(vfs, nonExistingDir)
	})

	existingFile := vfs.Join(vfs.TempDir(), "existing")

	err := vfs.WriteFile(existingFile, []byte{}, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	test.AssertPanic(t, "", func() {
		_ = basepathfs.New(vfs, existingFile)
	})
}

func TestBasePathFSFeatures(t *testing.T) {
	vfs := basepathfs.New(memfs.New(), "/")
	if vfs.HasFeature(avfs.FeatSymlink) {
		t.Errorf("Features : want FeatSymlink missing, got present")
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}

	mfs := memfs.New()

	vfs = basepathfs.New(mfs, "/")
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

func TestBasePathFSOSType(t *testing.T) {
	vfsBase := memfs.New()
	vfs := basepathfs.New(vfsBase, vfsBase.TempDir())

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
