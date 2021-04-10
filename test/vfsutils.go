//
//  Copyright 2021 The AVFS authors
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
	"bytes"
	"crypto/sha512"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfsutils"
)

// TestVFSUtils tests all VFSUtils functions.
func (sfs *SuiteFS) TestVFSUtils(t *testing.T) {
	sfs.RunTests(t, UsrTest,
		sfs.TestCheckPermission,
		sfs.TestCopyFile,
		sfs.TestCreateBaseDirs,
		sfs.TestDirExists,
		sfs.TestHashFile,
		sfs.TestRndTree,
		sfs.TestUMask,
		sfs.TestSegmentPath)
}

// TestCheckPermission tests vfsutils.CheckPermission function.
func (sfs *SuiteFS) TestCheckPermission(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	for perm := os.FileMode(0); perm <= 0o777; perm++ {
		path := fmt.Sprintf("%s/file%03o", testDir, perm)

		err := vfs.WriteFile(path, []byte(path), perm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
		}
	}

	for _, u := range sfs.Users {
		_, err := vfs.User(u.Name())
		if err != nil {
			t.Fatalf("User %s : want error to be nil, got %v", u.Name(), err)
		}

		for perm := os.FileMode(0); perm <= 0o777; perm++ {
			path := fmt.Sprintf("%s/file%03o", testDir, perm)

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

// TestCopyFile tests vfsutils.CopyFile function.
func (sfs *SuiteFS) TestCopyFile(t *testing.T, testDir string) {
	srcFs := sfs.VFSTest()

	dstFs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	rtr, err := vfsutils.NewRndTree(srcFs, &vfsutils.RndTreeParams{
		MinDepth: 1, MaxDepth: 1,
		MinName: 32, MaxName: 32,
		MinFiles: 512, MaxFiles: 512,
		MinFileLen: 0, MaxFileLen: 100 * 1024,
	})
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rtr.CreateTree(testDir)
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	h := sha512.New()

	t.Run("CopyFile_WithHashSum", func(t *testing.T) {
		dstDir, err := dstFs.TempDir("", avfs.Avfs)
		if err != nil {
			t.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		defer dstFs.RemoveAll(dstDir) //nolint:errcheck // Ignore errors.

		for _, srcPath := range rtr.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := vfsutils.Join(dstDir, fileName)

			wantSum, err := vfsutils.CopyFile(dstFs, srcFs, dstPath, srcPath, h)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			gotSum, err := vfsutils.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		dstDir, err := dstFs.TempDir("", avfs.Avfs)
		if err != nil {
			t.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		for _, srcPath := range rtr.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := vfsutils.Join(dstDir, fileName)

			wantSum, err := vfsutils.CopyFile(dstFs, srcFs, dstPath, srcPath, nil)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			if wantSum != nil {
				t.Fatalf("CopyFile (%s)%s, (%s)%s) : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			wantSum, err = vfsutils.HashFile(srcFs, srcPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", srcFs.Type(), srcPath, err)
			}

			gotSum, err := vfsutils.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})
}

// TestCreateBaseDirs tests vfsutils.CreateBaseDirs function.
func (sfs *SuiteFS) TestCreateBaseDirs(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	err := vfsutils.CreateBaseDirs(vfs, testDir)
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

// TestDirExists tests vfsutils.DirExists function.
func (sfs *SuiteFS) TestDirExists(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	existingFile := sfs.EmptyFile(t, testDir)

	t.Run("DirExistsDir", func(t *testing.T) {
		ok, err := vfsutils.DirExists(vfs, testDir)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("DirExists : want DirExists to be true, got false")
		}
	})

	t.Run("DirExistsFile", func(t *testing.T) {
		ok, err := vfsutils.DirExists(vfs, existingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})

	t.Run("DirExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := vfsutils.DirExists(vfs, nonExistingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})
}

// TestHashFile tests vfsutils.HashFile function.
func (sfs *SuiteFS) TestHashFile(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	rtr, err := vfsutils.NewRndTree(vfs, &vfsutils.RndTreeParams{
		MinDepth: 1, MaxDepth: 1,
		MinName: 32, MaxName: 32,
		MinFiles: 100, MaxFiles: 100,
		MinFileLen: 16, MaxFileLen: 100 * 1024,
	})
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rtr.CreateTree(testDir)
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	defer vfs.RemoveAll(testDir) //nolint:errcheck // Ignore errors.

	h := sha512.New()

	for _, fileName := range rtr.Files {
		content, err := vfs.ReadFile(fileName)
		if err != nil {
			t.Fatalf("ReadFile %s : want error to be nil, got %v", fileName, err)
		}

		h.Reset()

		_, err = h.Write(content)
		if err != nil {
			t.Errorf("Write hash : want error to be nil, got %v", err)
		}

		wantSum := h.Sum(nil)

		gotSum, err := vfsutils.HashFile(vfs, fileName, h)
		if err != nil {
			t.Errorf("HashFile %s : want error to be nil, got %v", fileName, err)
		}

		if !bytes.Equal(wantSum, gotSum) {
			t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
		}
	}
}

// TestRndTree tests vfsutils.RndTree function.
func (sfs *SuiteFS) TestRndTree(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	var (
		ErrDepthOutOfRange    = vfsutils.ErrOutOfRange("depth")
		ErrNameOutOfRange     = vfsutils.ErrOutOfRange("name")
		ErrDirsOutOfRange     = vfsutils.ErrOutOfRange("dirs")
		ErrFilesOutOfRange    = vfsutils.ErrOutOfRange("files")
		ErrFileLenOutOfRange  = vfsutils.ErrOutOfRange("file length")
		ErrSymlinksOutOfRange = vfsutils.ErrOutOfRange("symbolic links")
	)

	t.Run("RndTree", func(t *testing.T) {
		rtrTests := []struct {
			params  *vfsutils.RndTreeParams
			wantErr error
		}{
			{params: &vfsutils.RndTreeParams{MinDepth: 0, MaxDepth: 0}, wantErr: ErrDepthOutOfRange},
			{params: &vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 0}, wantErr: ErrDepthOutOfRange},
			{params: &vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 1, MinName: 0, MaxName: 0}, wantErr: ErrNameOutOfRange},
			{params: &vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 0}, wantErr: ErrNameOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinDirs: -1, MaxDirs: 0,
			}, wantErr: ErrDirsOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinDirs: 1, MaxDirs: 0,
			}, wantErr: ErrDirsOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFiles: -1, MaxFiles: 0,
			}, wantErr: ErrFilesOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFiles: 1, MaxFiles: 0,
			}, wantErr: ErrFilesOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFileLen: -1, MaxFileLen: 0,
			}, wantErr: ErrFileLenOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFileLen: 1, MaxFileLen: 0,
			}, wantErr: ErrFileLenOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinSymlinks: -1, MaxSymlinks: 0,
			}, wantErr: ErrSymlinksOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinSymlinks: 1, MaxSymlinks: 0,
			}, wantErr: ErrSymlinksOutOfRange},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 10, MaxName: 20,
				MinDirs: 5, MaxDirs: 10, MinFiles: 5, MaxFiles: 10, MinFileLen: 5, MaxFileLen: 10,
				MinSymlinks: 5, MaxSymlinks: 10,
			}, wantErr: nil},
			{params: &vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 3, MinName: 10, MaxName: 10,
				MinDirs: 3, MaxDirs: 3, MinFiles: 3, MaxFiles: 3, MinFileLen: 3, MaxFileLen: 3,
				MinSymlinks: 3, MaxSymlinks: 3,
			}, wantErr: nil},
		}

		for i, rtrTest := range rtrTests {
			rtr, err := vfsutils.NewRndTree(vfs, rtrTest.params)

			if rtrTest.wantErr == nil {
				if err != nil {
					t.Errorf("NewRndTree %d: want error to be nil, got %v", i, err)

					continue
				}
			} else {
				if err == nil {
					t.Errorf("NewRndTree %d : want error to be %v, got nil", i, rtrTest.wantErr)
				} else if rtrTest.wantErr != err {
					t.Errorf("NewRndTree %d : want error to be %v, got %v", i, rtrTest.wantErr, err)
				}

				continue
			}

			path := vfs.Join(testDir, "Main", strconv.Itoa(i))

			err = vfs.MkdirAll(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("MkdirAll %s : want error to be nil, got %v", path, err)
			}

			err = rtr.CreateTree(path)
			if err != nil {
				t.Errorf("CreateTree : want error to be nil, got %v", err)
			}

			if rtr.MaxDepth == 0 {
				ld := len(rtr.Dirs)
				if ld < rtr.MinDirs || ld > rtr.MaxDirs {
					t.Errorf("CreateTree : want dirs number to to be between %d and %d, got %d",
						rtr.MinDirs, rtr.MaxDirs, ld)
				}

				lf := len(rtr.Files)
				if lf < rtr.MinFiles || lf > rtr.MaxFiles {
					t.Errorf("CreateTree : want files number to to be between %d and %d, got %d",
						rtr.MinFiles, rtr.MaxFiles, lf)
				}

				ls := len(rtr.SymLinks)
				if ls < rtr.MinSymlinks || ls > rtr.MaxSymlinks {
					t.Errorf("CreateTree : want symbolic linls number to to be between %d and %d, got %d",
						rtr.MinSymlinks, rtr.MaxSymlinks, ls)
				}
			}
		}
	})

	t.Run("RndTreeDepth", func(t *testing.T) {
		rtr, err := vfsutils.NewRndTree(vfs, &vfsutils.RndTreeParams{
			MinDepth: 3, MaxDepth: 3,
			MinName: 10, MaxName: 10,
			MinDirs: 2, MaxDirs: 2,
			MinFiles: 1, MaxFiles: 1,
		})
		if err != nil {
			t.Errorf("NewRndTree : want error to be nil, got %v", err)
		}

		path := vfs.Join(testDir, "Depth")

		err = vfs.MkdirAll(path, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("MkdirAll %s : want error to be nil, got %v", path, err)
		}

		err = rtr.CreateTree(path)
		if err != nil {
			t.Errorf("CreateTree : want error to be nil, got %v", err)
		}

		wantDirs := 14
		if len(rtr.Dirs) != wantDirs {
			t.Errorf("CreateTree : want number of directories to be %d, got %d", wantDirs, len(rtr.Dirs))
		}

		wantFiles := 7
		if len(rtr.Files) != wantFiles {
			t.Errorf("CreateTree : want number of directories to be %d, got %d", wantFiles, len(rtr.Files))
		}
	})

	t.Run("RndTreeOutOfRange", func(t *testing.T) {
		parameter := "Some"
		wantErrStr := parameter + " parameter out of range"

		err := vfsutils.ErrOutOfRange(parameter)
		if err.Error() != wantErrStr {
			t.Errorf("ErrOutOfRange : want error to be %s, got %s", wantErrStr, err.Error())
		}
	})
}

// TestUMask tests UMask functions.
func (sfs *SuiteFS) TestUMask(t *testing.T, testDir string) {
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

// TestSegmentPath tests vfsutils.SegmentPath function.
func (sfs *SuiteFS) TestSegmentPath(t *testing.T, testDir string) {
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
