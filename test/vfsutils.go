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
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfsutils"
)

// TestCopyFile tests vfsutils.CopyFile function.
func (sfs *SuiteFS) TestCopyFile(t *testing.T, testDir string) {
	srcFs := sfs.VFSSetup()

	if !srcFs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	dstFs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	rtParams := &vfsutils.RndTreeParams{
		MinName: 32, MaxName: 32,
		MinFiles: 512, MaxFiles: 512,
		MinFileSize: 0, MaxFileSize: 100 * 1024,
	}

	rt, err := vfsutils.NewRndTree(srcFs, testDir, rtParams)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rt.CreateTree()
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

		for _, srcPath := range rt.Files {
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

		for _, srcPath := range rt.Files {
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
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

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

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

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
		existingFile := sfs.EmptyFile(t, testDir)

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

// TestExists tests vfsutils.Exists function.
func (sfs *SuiteFS) TestExists(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("ExistsDir", func(t *testing.T) {
		ok, err := vfsutils.Exists(vfs, testDir)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := vfsutils.Exists(vfs, existingFile)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := vfsutils.Exists(vfs, nonExistingFile)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})
}

// TestHashFile tests vfsutils.HashFile function.
func (sfs *SuiteFS) TestHashFile(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	rtParams := &vfsutils.RndTreeParams{
		MinName: 32, MaxName: 32,
		MinFiles: 100, MaxFiles: 100,
		MinFileSize: 16, MaxFileSize: 100 * 1024,
	}

	rt, err := vfsutils.NewRndTree(vfs, testDir, rtParams)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rt.CreateTree()
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	defer vfs.RemoveAll(testDir) //nolint:errcheck // Ignore errors.

	h := sha512.New()

	for _, fileName := range rt.Files {
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
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("RndTree", func(t *testing.T) {
		rtTests := []*vfsutils.RndTreeParams{
			{
				MinName: 10, MaxName: 20,
				MinDirs: 5, MaxDirs: 10,
				MinFiles: 5, MaxFiles: 10,
				MinFileSize: 5, MaxFileSize: 10,
				MinSymlinks: 5, MaxSymlinks: 10,
			},
			{
				MinName: 10, MaxName: 10,
				MinDirs: 3, MaxDirs: 3,
				MinFiles: 3, MaxFiles: 3,
				MinFileSize: 3, MaxFileSize: 3,
				MinSymlinks: 3, MaxSymlinks: 3,
			},
		}

		for i, rtTest := range rtTests {
			path := vfs.Join(testDir, "RndTree", strconv.Itoa(i))

			err := vfs.MkdirAll(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("MkdirAll %s :  want error to be nil, got %v", path, err)
			}

			rt, err := vfsutils.NewRndTree(vfs, path, rtTest)
			if err != nil {
				t.Errorf("NewRndTree %d : want error to be nil, got %v", i, err)
			}

			err = rt.CreateTree()
			if err != nil {
				t.Errorf("CreateTree %d : want error to be nil, got %v", i, err)
			}

			nbDirs := len(rt.Dirs)
			if nbDirs < rtTest.MinDirs || nbDirs > rtTest.MaxDirs {
				t.Errorf("Dirs %d : want nb Dirs to be between %d and %d, got %d",
					i, rtTest.MinDirs, rtTest.MaxDirs, nbDirs)
			}

			nbFiles := len(rt.Files)
			if nbFiles < rtTest.MinFiles || nbFiles > rtTest.MaxFiles {
				t.Errorf("Files %d : want nb Files to be between %d and %d, got %d",
					i, rtTest.MinFiles, rtTest.MaxFiles, nbFiles)
			}

			if !vfs.HasFeature(avfs.FeatSymlink) {
				continue
			}

			nbSymlinks := len(rt.SymLinks)
			if nbSymlinks < rtTest.MinSymlinks || nbSymlinks > rtTest.MaxSymlinks {
				t.Errorf("Dirs %d : want nb Dirs to be between %d and %d, got %d",
					i, rtTest.MinSymlinks, rtTest.MaxSymlinks, nbSymlinks)
			}
		}
	})

	t.Run("RndTreeErrors", func(t *testing.T) {
		rtTests := []struct {
			params  *vfsutils.RndTreeParams
			wantErr error
		}{
			{
				params:  &vfsutils.RndTreeParams{MinName: 0, MaxName: 0},
				wantErr: vfsutils.ErrNameOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 0},
				wantErr: vfsutils.ErrNameOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinDirs: -1, MaxDirs: 0},
				wantErr: vfsutils.ErrDirsOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinDirs: 1, MaxDirs: 0},
				wantErr: vfsutils.ErrDirsOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinFiles: -1, MaxFiles: 0},
				wantErr: vfsutils.ErrFilesOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinFiles: 1, MaxFiles: 0},
				wantErr: vfsutils.ErrFilesOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinFileSize: -1, MaxFileSize: 0},
				wantErr: vfsutils.ErrFileSizeOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinFileSize: 1, MaxFileSize: 0},
				wantErr: vfsutils.ErrFileSizeOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinSymlinks: -1, MaxSymlinks: 0},
				wantErr: vfsutils.ErrSymlinksOutOfRange,
			},
			{
				params:  &vfsutils.RndTreeParams{MinName: 1, MaxName: 1, MinSymlinks: 1, MaxSymlinks: 0},
				wantErr: vfsutils.ErrSymlinksOutOfRange,
			},
		}

		for i, rtTest := range rtTests {
			_, err := vfsutils.NewRndTree(vfs, testDir, rtTest.params)
			if rtTest.wantErr != err {
				t.Errorf("NewRndTree %d : want error to be %v, got %v", i, rtTest.wantErr, err)
			}
		}
	})
}

// TestToSysStat tests ToSysStat function.
func (sfs *SuiteFS) TestToSysStat(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		sst := vfsutils.ToSysStat(nil)

		if _, ok := sst.(*vfsutils.DummySysStat); !ok {
			t.Errorf("ToSysStat : want result of type DummySysStat, got %s", reflect.TypeOf(sst).Name())
		}

		return
	}

	existingFile := sfs.EmptyFile(t, testDir)

	fst, err := vfs.Stat(existingFile)
	if err != nil {
		t.Errorf("Stat : want error be nil, got %v", err)
	}

	u := vfs.CurrentUser()
	if vfs.HasFeature(avfs.FeatReadOnly) {
		u = sfs.vfsSetup.CurrentUser()
	}

	wantUid, wantGid := u.Uid(), u.Gid()

	sst := vfsutils.ToSysStat(fst.Sys())

	uid, gid := sst.Uid(), sst.Gid()
	if uid != wantUid || gid != wantGid {
		t.Errorf("ToSysStat : want Uid = %d, Gid = %d, got Uid = %d, Gid = %d",
			wantUid, wantGid, uid, gid)
	}

	wantLink := uint64(1)
	if sst.Nlink() != wantLink {
		t.Errorf("ToSysStat : want Nlink to be %d, got %d", wantLink, sst.Nlink())
	}
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
