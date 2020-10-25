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

// +build !datarace

package fsutil_test

import (
	"os"
	"testing"

	"github.com/avfs/avfs"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fs/orefafs"
	"github.com/avfs/avfs/fs/osfs"
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

func TestAsStatT(t *testing.T) {
	t.Run("StatT MemFs", func(t *testing.T) {
		vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
		if err != nil {
			t.Errorf("memfs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFs(t, vfs)
		sfs.StatT()
	})

	t.Run("StatT OsFs", func(t *testing.T) {
		vfs, err := osfs.New(osfs.WithIdm(osidm.New()))
		if err != nil {
			t.Errorf("osfs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFs(t, vfs)
		sfs.StatT()
	})

	t.Run("StatT OrefaFs", func(t *testing.T) {
		vfs, err := orefafs.New(orefafs.WithMainDirs())
		if err != nil {
			t.Errorf("orefafs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFs(t, vfs)
		sfs.StatT()
	})
}

func TestCreateBaseDirs(t *testing.T) {
	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = fsutil.CreateBaseDirs(vfs, "")
	if err != nil {
		t.Fatalf("CreateBaseDirs : want error to be nil, got %v", err)
	}

	for _, dir := range fsutil.BaseDirs {
		info, err := vfs.Stat(dir.Path)
		if err != nil {
			t.Fatalf("CreateBaseDirs : want error to be nil, got %v", err)
		}

		gotMode := info.Mode() & os.ModePerm
		if gotMode != dir.Perm {
			t.Errorf("CreateBaseDirs %s :  want mode to be %o, got %o", dir.Path, dir.Perm, gotMode)
		}
	}
}

// TestUMask tests Umask and GetYMask functions.
func TestUMask(t *testing.T) {
	umaskOs := os.FileMode(0o22)
	umaskSet := os.FileMode(0o77)
	umaskTest := umaskSet

	if fsutil.RunTimeOS() == avfs.OsWindows {
		umaskTest = umaskOs
	}

	umask := fsutil.UMask.Get()
	if umask != umaskOs {
		t.Errorf("GetUMask : want OS umask %o, got %o", umaskOs, umask)
	}

	fsutil.UMask.Set(umaskSet)

	umask = fsutil.UMask.Get()
	if umask != umaskTest {
		t.Errorf("GetUMask : want test umask %o, got %o", umaskTest, umask)
	}

	fsutil.UMask.Set(umaskOs)

	umask = fsutil.UMask.Get()
	if umask != umaskOs {
		t.Errorf("GetUMask : want OS umask %o, got %o", umaskOs, umask)
	}
}

// TestSegmentPath tests SegmentPath function.
func TestSegmentPath(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{path: "", want: []string{""}},
		{path: "/", want: []string{"", ""}},
		{path: "//", want: []string{"", "", ""}},
		{path: "/a", want: []string{"", "a"}},
		{path: "/b/c/d", want: []string{"", "b", "c", "d"}},
		{path: "/नमस्ते/दुनिया", want: []string{"", "नमस्ते", "दुनिया"}},
	}

	for _, c := range cases {
		for start, end, i, isLast := 0, 0, 0, false; !isLast; start, i = end+1, i+1 {
			end, isLast = fsutil.SegmentPath(c.path, start)
			got := c.path[start:end]

			if i > len(c.want) {
				t.Errorf("%s : want %d parts, got only %d", c.path, i, len(c.want))
				break
			}

			if got != c.want[i] {
				t.Errorf("%s : want part %d to be %s, got %s", c.path, i, c.want[i], got)
			}
		}
	}
}
