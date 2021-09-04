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
	"io/fs"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
)

const (
	defaultUmask = fs.FileMode(0o22) // defaultUmask is the default umask.
)

// TestCopyFile tests avfs.CopyFile function.
func (sfs *SuiteFS) TestCopyFile(t *testing.T, testDir string) {
	const pattern = "CopyFile"

	srcFs := sfs.VFSSetup()

	if !srcFs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	dstFs := memfs.New(memfs.WithMainDirs())

	rtParams := &avfs.RndTreeParams{
		MinName: 32, MaxName: 32,
		MinFiles: 512, MaxFiles: 512,
		MinFileSize: 0, MaxFileSize: 100 * 1024,
	}

	rt, err := avfs.NewRndTree(srcFs, testDir, rtParams)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rt.CreateTree()
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	h := sha512.New()

	t.Run("CopyFile_WithHashSum", func(t *testing.T) {
		dstDir, err := dstFs.MkdirTemp("", pattern)
		if err != nil {
			t.Fatalf("MkdirTemp : want error to be nil, got %v", err)
		}

		defer dstFs.RemoveAll(dstDir) //nolint:errcheck // Ignore errors.

		for _, srcPath := range rt.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := srcFs.Join(dstDir, fileName)

			wantSum, err := avfs.CopyFile(dstFs, srcFs, dstPath, srcPath, h)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			gotSum, err := avfs.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		dstDir, err := dstFs.MkdirTemp("", pattern)
		if err != nil {
			t.Fatalf("MkdirTemp : want error to be nil, got %v", err)
		}

		for _, srcPath := range rt.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := srcFs.Join(dstDir, fileName)

			wantSum, err := avfs.CopyFile(dstFs, srcFs, dstPath, srcPath, nil)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			if wantSum != nil {
				t.Fatalf("CopyFile (%s)%s, (%s)%s) : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			wantSum, err = avfs.HashFile(srcFs, srcPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", srcFs.Type(), srcPath, err)
			}

			gotSum, err := avfs.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})
}

// TestCreateBaseDirs tests avfs.CreateBaseDirs function.
func (sfs *SuiteFS) TestCreateBaseDirs(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) || avfs.RunTimeOS() == avfs.OsWindows {
		return
	}

	err := avfs.CreateBaseDirs(vfs, testDir)
	if err != nil {
		t.Fatalf("CreateBaseDirs : want error to be nil, got %v", err)
	}

	for _, dir := range avfs.BaseDirs(vfs) {
		info, err := vfs.Stat(dir.Path)
		if err != nil {
			t.Fatalf("CreateBaseDirs : want error to be nil, got %v", err)
		}

		gotMode := info.Mode() & fs.ModePerm
		if gotMode != dir.Perm {
			t.Errorf("CreateBaseDirs %s :  want mode to be %o, got %o", dir.Path, dir.Perm, gotMode)
		}
	}
}

// TestDirExists tests avfs.DirExists function.
func (sfs *SuiteFS) TestDirExists(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("DirExistsDir", func(t *testing.T) {
		ok, err := avfs.DirExists(vfs, testDir)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("DirExists : want DirExists to be true, got false")
		}
	})

	t.Run("DirExistsFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.DirExists(vfs, existingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})

	t.Run("DirExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.DirExists(vfs, nonExistingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})
}

// TestExists tests avfs.Exists function.
func (sfs *SuiteFS) TestExists(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("ExistsDir", func(t *testing.T) {
		ok, err := avfs.Exists(vfs, testDir)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.Exists(vfs, existingFile)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.Exists(vfs, nonExistingFile)
		if err != nil {
			t.Errorf("Exists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})

	t.Run("ExistsInvalidPath", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		invalidPath := vfs.Join(existingFile, defaultFile)

		ok, err := avfs.Exists(vfs, invalidPath)
		CheckPathError(t, err).Op("stat").Path(invalidPath).Err(avfs.ErrNotADirectory)

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})
}

// TestIsDir tests avfs.IsDir function.
func (sfs *SuiteFS) TestIsDir(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("IsDir", func(t *testing.T) {
		existingDir := sfs.ExistingDir(t, testDir)

		ok, err := avfs.IsDir(vfs, existingDir)
		if err != nil {
			t.Errorf("IsDir : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("IsDir : want IsDir to be true, got false")
		}
	})

	t.Run("IsDirFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.IsDir(vfs, existingFile)
		if err != nil {
			t.Errorf("IsDirFile : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("IsDirFile : want DirExists to be false, got true")
		}
	})

	t.Run("IsDirNonExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.IsDir(vfs, nonExistingFile)
		CheckPathError(t, err).Op("stat").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)

		if ok {
			t.Error("IsDirNonExisting : want DirExists to be false, got true")
		}
	})
}

// TestIsEmpty tests avfs.IsEmpty function.
func (sfs *SuiteFS) TestIsEmpty(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("IsEmptyFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.IsEmpty(vfs, existingFile)
		if err != nil {
			t.Errorf("IsEmpty : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDirEmpty", func(t *testing.T) {
		emptyDir := sfs.ExistingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, emptyDir)
		if err != nil {
			t.Errorf("IsEmpty : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDir", func(t *testing.T) {
		sfs.ExistingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, testDir)
		if err != nil {
			t.Errorf("IsEmpty : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("IsEmpty : want IsEmpty to be false, got true")
		}
	})

	t.Run("IsEmptyNonExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		wantErr := fmt.Errorf("%q path does not exist", nonExistingFile)

		ok, err := avfs.IsEmpty(vfs, nonExistingFile)
		if err.Error() != wantErr.Error() {
			t.Errorf("IsEmpty : want error to be %v, got %v", wantErr, err)
		}

		if ok {
			t.Error("IsEmpty : want IsEmpty to be false, got true")
		}
	})
}

// TestHashFile tests avfs.HashFile function.
func (sfs *SuiteFS) TestHashFile(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	rtParams := &avfs.RndTreeParams{
		MinName: 32, MaxName: 32,
		MinFiles: 100, MaxFiles: 100,
		MinFileSize: 16, MaxFileSize: 100 * 1024,
	}

	rt, err := avfs.NewRndTree(vfs, testDir, rtParams)
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

		gotSum, err := avfs.HashFile(vfs, fileName, h)
		if err != nil {
			t.Errorf("HashFile %s : want error to be nil, got %v", fileName, err)
		}

		if !bytes.Equal(wantSum, gotSum) {
			t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
		}
	}
}

// TestRndTree tests avfs.RndTree function.
func (sfs *SuiteFS) TestRndTree(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	t.Run("RndTree", func(t *testing.T) {
		rtTests := []*avfs.RndTreeParams{
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

			sfs.CreateDir(t, path, avfs.DefaultDirPerm)

			rt, err := avfs.NewRndTree(vfs, path, rtTest)
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
			params  *avfs.RndTreeParams
			wantErr error
		}{
			{
				params:  &avfs.RndTreeParams{MinName: 0, MaxName: 0},
				wantErr: avfs.ErrNameOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 0},
				wantErr: avfs.ErrNameOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinDirs: -1, MaxDirs: 0},
				wantErr: avfs.ErrDirsOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinDirs: 1, MaxDirs: 0},
				wantErr: avfs.ErrDirsOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinFiles: -1, MaxFiles: 0},
				wantErr: avfs.ErrFilesOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinFiles: 1, MaxFiles: 0},
				wantErr: avfs.ErrFilesOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinFileSize: -1, MaxFileSize: 0},
				wantErr: avfs.ErrFileSizeOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinFileSize: 1, MaxFileSize: 0},
				wantErr: avfs.ErrFileSizeOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinSymlinks: -1, MaxSymlinks: 0},
				wantErr: avfs.ErrSymlinksOutOfRange,
			},
			{
				params:  &avfs.RndTreeParams{MinName: 1, MaxName: 1, MinSymlinks: 1, MaxSymlinks: 0},
				wantErr: avfs.ErrSymlinksOutOfRange,
			},
		}

		for i, rtTest := range rtTests {
			_, err := avfs.NewRndTree(vfs, testDir, rtTest.params)
			if rtTest.wantErr != err {
				t.Errorf("NewRndTree %d : want error to be %v, got %v", i, rtTest.wantErr, err)
			}
		}
	})
}

// TestUMask tests UMask functions.
func (sfs *SuiteFS) TestUMask(t *testing.T, testDir string) {
	const umaskSet = fs.FileMode(0o77)

	umaskTest := umaskSet

	if avfs.RunTimeOS() == avfs.OsWindows {
		umaskTest = defaultUmask
	}

	umask := avfs.UMask.Get()
	if umask != defaultUmask {
		t.Errorf("GetUMask : want OS umask %o, got %o", defaultUmask, umask)
	}

	avfs.UMask.Set(umaskSet)

	umask = avfs.UMask.Get()
	if umask != umaskTest {
		t.Errorf("GetUMask : want test umask %o, got %o", umaskTest, umask)
	}

	avfs.UMask.Set(defaultUmask)

	umask = avfs.UMask.Get()
	if umask != defaultUmask {
		t.Errorf("GetUMask : want OS umask %o, got %o", defaultUmask, umask)
	}
}

// TestSegmentPath tests avfs.SegmentPath function.
func (sfs *SuiteFS) TestSegmentPath(t *testing.T, testDir string) {
	vfs := sfs.VFSTest()

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
			end, isLast = avfs.SegmentPath(vfs.PathSeparator(), c.path, start)
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
