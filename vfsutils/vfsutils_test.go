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

package vfsutils_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fs/orefafs"
	"github.com/avfs/avfs/fs/osfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfsutils"
)

func TestAsStatT(t *testing.T) {
	t.Run("StatT MemFS", func(t *testing.T) {
		vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
		if err != nil {
			t.Errorf("memfs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFS(t, vfs)
		sfs.StatT()
	})

	t.Run("StatT OsFS", func(t *testing.T) {
		vfs, err := osfs.New(osfs.WithIdm(osidm.New()))
		if err != nil {
			t.Errorf("osfs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFS(t, vfs)
		sfs.StatT()
	})

	t.Run("StatT OrefaFS", func(t *testing.T) {
		vfs, err := orefafs.New(orefafs.WithMainDirs())
		if err != nil {
			t.Errorf("orefafs.New : want error to be nil, got %v", err)
		}

		sfs := test.NewSuiteFS(t, vfs)
		sfs.StatT()
	})
}

func TestCheckPermission(t *testing.T) {
	vfs, err := osfs.New(osfs.WithIdm(osidm.New()))
	if err != nil {
		t.Fatalf("osfs.New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)

	_, rootDir, removeDir := sfs.CreateRootDir(test.UsrTest)
	defer removeDir()

	for perm := os.FileMode(0); perm <= 0o777; perm++ {
		path := fmt.Sprintf("%s/file%03o", rootDir, perm)

		err = vfs.WriteFile(path, []byte(path), perm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
		}
	}

	for _, u := range sfs.Users {
		_, err = vfs.User(u.Name())
		if err != nil {
			t.Fatalf("User %s : want error to be nil, got %v", u.Name(), err)
		}

		for perm := os.FileMode(0); perm <= 0o777; perm++ {
			path := fmt.Sprintf("%s/file%03o", rootDir, perm)

			info, err := vfs.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			u := vfs.CurrentUser()

			wantCheckRead := vfsutils.CheckPermission(info, avfs.WantRead, u)
			gotCheckRead := true

			_, err = vfs.ReadFile(path)
			if err != nil {
				gotCheckRead = false
			}

			if wantCheckRead != gotCheckRead {
				t.Errorf("CheckPermission %s : want read to be %t, got %t", path, wantCheckRead, gotCheckRead)
			}

			wantCheckWrite := vfsutils.CheckPermission(info, avfs.WantWrite, u)
			gotCheckWrite := true

			err = vfs.WriteFile(path, []byte(path), perm)
			if err != nil {
				gotCheckWrite = false
			}

			if wantCheckWrite != gotCheckWrite {
				t.Errorf("CheckPermission %s : want write to be %t, got %t", path, wantCheckWrite, gotCheckWrite)
			}
		}
	}
}

func TestCreateBaseDirs(t *testing.T) {
	vfs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = vfsutils.CreateBaseDirs(vfs, "")
	if err != nil {
		t.Fatalf("CreateBaseDirs : want error to be nil, got %v", err)
	}

	for _, dir := range vfsutils.BaseDirs {
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

	if vfsutils.RunTimeOS() == avfs.OsWindows {
		umaskTest = umaskOs
	}

	umask := vfsutils.UMask.Get()
	if umask != umaskOs {
		t.Errorf("GetUMask : want OS umask %o, got %o", umaskOs, umask)
	}

	vfsutils.UMask.Set(umaskSet)

	umask = vfsutils.UMask.Get()
	if umask != umaskTest {
		t.Errorf("GetUMask : want test umask %o, got %o", umaskTest, umask)
	}

	vfsutils.UMask.Set(umaskOs)

	umask = vfsutils.UMask.Get()
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
			end, isLast = vfsutils.SegmentPath(c.path, start)
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
