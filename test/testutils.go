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
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
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
		MinFiles: 32, MaxFiles: 32,
		MinFileSize: 0, MaxFileSize: 100 * 1024,
	}

	rt, err := avfs.NewRndTree(srcFs, testDir, rtParams)
	if !CheckNoError(t, "NewRndTree", err) {
		return
	}

	err = rt.CreateTree()
	if !CheckNoError(t, "CreateTree", err) {
		return
	}

	h := sha512.New()

	t.Run("CopyFile_WithHashSum", func(t *testing.T) {
		dstDir, err := dstFs.MkdirTemp("", pattern)
		if !CheckNoError(t, "MkdirTemp", err) {
			return
		}

		defer dstFs.RemoveAll(dstDir) //nolint:errcheck // Ignore errors.

		for _, srcFile := range rt.Files {
			fileName := srcFs.Base(srcFile.Name)
			dstPath := dstFs.Join(dstDir, fileName)
			copyName := fmt.Sprintf("CopyFile (%s)%s, (%s)%s", dstFs.Type(), dstPath, srcFs.Type(), srcFile.Name)

			wantSum, err := avfs.CopyFile(dstFs, srcFs, dstPath, srcFile.Name, h)
			CheckNoError(t, copyName, err)

			gotSum, err := avfs.HashFile(dstFs, dstPath, h)
			CheckNoError(t, fmt.Sprintf("HashFile (%s)%s", dstFs.Type(), dstPath), err)

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		dstDir, err := dstFs.MkdirTemp("", pattern)
		if !CheckNoError(t, "MkdirTemp "+pattern, err) {
			return
		}

		for _, srcFile := range rt.Files {
			fileName := srcFs.Base(srcFile.Name)
			dstPath := dstFs.Join(dstDir, fileName)
			copyName := fmt.Sprintf("CopyFile (%s)%s, (%s)%s", dstFs.Type(), dstPath, srcFs.Type(), srcFile.Name)

			wantSum, err := avfs.CopyFile(dstFs, srcFs, dstPath, srcFile.Name, nil)
			CheckNoError(t, copyName, err)

			if wantSum != nil {
				t.Fatalf("%s : want hash sum to be nil, got %v", copyName, err)
			}

			wantSum, err = avfs.HashFile(srcFs, srcFile.Name, h)
			CheckNoError(t, fmt.Sprintf("HashFile (%s)%s", srcFs.Type(), srcFile.Name), err)

			gotSum, err := avfs.HashFile(dstFs, dstPath, h)
			CheckNoError(t, fmt.Sprintf("HashFile (%s)%s", dstFs.Type(), dstPath), err)

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})
}

// TestCreateBaseDirs tests avfs.CreateBaseDirs function.
func (sfs *SuiteFS) TestCreateBaseDirs(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := avfs.CreateBaseDirs(vfs, testDir)
	if !CheckNoError(t, "CreateBaseDirs", err) {
		return
	}

	for _, dir := range avfs.BaseDirs(vfs) {
		info, err := vfs.Stat(dir.Path)
		if !CheckNoError(t, "Stat", err) {
			continue
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
		CheckNoError(t, "DirExists "+testDir, err)

		if !ok {
			t.Error("DirExists : want DirExists to be true, got false")
		}
	})

	t.Run("DirExistsFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.DirExists(vfs, existingFile)
		CheckNoError(t, "DirExists "+testDir, err)

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})

	t.Run("DirExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.DirExists(vfs, nonExistingFile)
		CheckNoError(t, "DirExists "+nonExistingFile, err)

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
		CheckNoError(t, "Exists "+testDir, err)

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.Exists(vfs, existingFile)
		CheckNoError(t, "DirExists "+existingFile, err)

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsNotExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.Exists(vfs, nonExistingFile)
		CheckNoError(t, "Exists "+nonExistingFile, err)

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})

	t.Run("ExistsInvalidPath", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		invalidPath := vfs.Join(existingFile, defaultFile)

		ok, err := avfs.Exists(vfs, invalidPath)
		CheckPathError(t, err).OpStat(vfs).Path(invalidPath).Err(avfs.ErrNotADirectory)

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
		CheckNoError(t, "IsDir "+existingDir, err)

		if !ok {
			t.Error("IsDir : want IsDir to be true, got false")
		}
	})

	t.Run("IsDirFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		ok, err := avfs.IsDir(vfs, existingFile)
		CheckNoError(t, "IsDir "+existingFile, err)

		if ok {
			t.Error("IsDirFile : want DirExists to be false, got true")
		}
	})

	t.Run("IsDirNonExisting", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		ok, err := avfs.IsDir(vfs, nonExistingFile)
		CheckPathError(t, err).OpStat(vfs).Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)

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
		CheckNoError(t, "IsEmpty "+existingFile, err)

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDirEmpty", func(t *testing.T) {
		emptyDir := sfs.ExistingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, emptyDir)
		CheckNoError(t, "IsEmpty "+emptyDir, err)

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDir", func(t *testing.T) {
		sfs.ExistingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, testDir)
		CheckNoError(t, "IsEmpty "+testDir, err)

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
	if !CheckNoError(t, "NewRndTree", err) {
		return
	}

	err = rt.CreateTree()
	if !CheckNoError(t, "CreateTree", err) {
		return
	}

	defer vfs.RemoveAll(testDir) //nolint:errcheck // Ignore errors.

	h := sha512.New()

	for _, file := range rt.Files {
		content, err := vfs.ReadFile(file.Name)
		if !CheckNoError(t, "ReadFile", err) {
			continue
		}

		h.Reset()

		_, err = h.Write(content)
		CheckNoError(t, "Write", err)

		wantSum := h.Sum(nil)

		gotSum, err := avfs.HashFile(vfs, file.Name, h)
		CheckNoError(t, "HashFile", err)

		if !bytes.Equal(wantSum, gotSum) {
			t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", file.Name, wantSum, gotSum)
		}
	}
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}

	return e.Error()
}

func (sfs *SuiteFS) TestMatch(t *testing.T, testDir string) {
	ut := avfs.NewUtils(avfs.CurrentOSType())

	matchTests := []struct { //nolint:govet // no fieldalignment for simple structs
		pattern, s string
		match      bool
		err        error
	}{
		{"abc", "abc", true, nil},
		{"*", "abc", true, nil},
		{"*c", "abc", true, nil},
		{"a*", "a", true, nil},
		{"a*", "abc", true, nil},
		{"a*", "ab/c", false, nil},
		{"a*/b", "abc/b", true, nil},
		{"a*/b", "a/c/b", false, nil},
		{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
		{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
		{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
		{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
		{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
		{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
		{"ab[c]", "abc", true, nil},
		{"ab[b-d]", "abc", true, nil},
		{"ab[e-g]", "abc", false, nil},
		{"ab[^c]", "abc", false, nil},
		{"ab[^b-d]", "abc", false, nil},
		{"ab[^e-g]", "abc", true, nil},
		{"a\\*b", "a*b", true, nil},
		{"a\\*b", "ab", false, nil},
		{"a?b", "a☺b", true, nil},
		{"a[^a]b", "a☺b", true, nil},
		{"a???b", "a☺b", false, nil},
		{"a[^a][^a][^a]b", "a☺b", false, nil},
		{"[a-ζ]*", "α", true, nil},
		{"*[a-ζ]", "A", false, nil},
		{"a?b", "a/b", false, nil},
		{"a*b", "a/b", false, nil},
		{"[\\]a]", "]", true, nil},
		{"[\\-]", "-", true, nil},
		{"[x\\-]", "x", true, nil},
		{"[x\\-]", "-", true, nil},
		{"[x\\-]", "z", false, nil},
		{"[\\-x]", "x", true, nil},
		{"[\\-x]", "-", true, nil},
		{"[\\-x]", "a", false, nil},
		{"[]a]", "]", false, filepath.ErrBadPattern},
		{"[-]", "-", false, filepath.ErrBadPattern},
		{"[x-]", "x", false, filepath.ErrBadPattern},
		{"[x-]", "-", false, filepath.ErrBadPattern},
		{"[x-]", "z", false, filepath.ErrBadPattern},
		{"[-x]", "x", false, filepath.ErrBadPattern},
		{"[-x]", "-", false, filepath.ErrBadPattern},
		{"[-x]", "a", false, filepath.ErrBadPattern},
		{"\\", "a", false, filepath.ErrBadPattern},
		{"[a-b-c]", "a", false, filepath.ErrBadPattern},
		{"[", "a", false, filepath.ErrBadPattern},
		{"[^", "a", false, filepath.ErrBadPattern},
		{"[^bc", "a", false, filepath.ErrBadPattern},
		{"a[", "a", false, filepath.ErrBadPattern},
		{"a[", "ab", false, filepath.ErrBadPattern},
		{"a[", "x", false, filepath.ErrBadPattern},
		{"a/b[", "x", false, filepath.ErrBadPattern},
		{"*x", "xxx", true, nil},
	}

	for _, tt := range matchTests {
		pattern := tt.pattern
		s := tt.s

		if ut.OSType() == avfs.OsWindows {
			if strings.Contains(pattern, "\\") {
				// no escape allowed on windows.
				continue
			}

			pattern = ut.Clean(pattern)
			s = ut.Clean(s)
		}

		ok, err := ut.Match(pattern, s)
		if ok != tt.match || err != tt.err {
			t.Errorf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}

// TestRndTree tests avfs.RndTree function.
func (sfs *SuiteFS) TestRndTree(t *testing.T, testDir string) {
	const randSeed = 42

	vfs := sfs.VFSSetup()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	rtTests := []*avfs.RndTreeParams{
		{
			MinName: 5, MaxName: 10,
			MinDirs: 10, MaxDirs: 30,
			MinFiles: 10, MaxFiles: 30,
			MinFileSize: 0, MaxFileSize: 2048,
			MinSymlinks: 10, MaxSymlinks: 30,
		},
		{
			MinName: 3, MaxName: 3,
			MinDirs: 3, MaxDirs: 3,
			MinFiles: 3, MaxFiles: 3,
			MinFileSize: 3, MaxFileSize: 3,
			MinSymlinks: 3, MaxSymlinks: 3,
		},
	}

	t.Run("RndTree", func(t *testing.T) {
		rand.Seed(randSeed)

		for i, rtTest := range rtTests {
			path := vfs.Join(testDir, "RndTree", strconv.Itoa(i))

			sfs.CreateDir(t, path, avfs.DefaultDirPerm)

			rt, err := avfs.NewRndTree(vfs, path, rtTest)
			CheckNoError(t, "NewRndTree", err)

			err = rt.CreateTree()
			if !CheckNoError(t, "CreateTree "+strconv.Itoa(i), err) {
				continue
			}

			err = rt.CreateDirs()
			CheckNoError(t, "CreateDirs", err)

			err = rt.CreateFiles()
			CheckNoError(t, "CreateFiles", err)

			err = rt.CreateSymlinks()
			CheckNoError(t, "CreateSymlinks", err)

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

			if vfs.HasFeature(avfs.FeatSymlink) {
				nbSymlinks := len(rt.SymLinks)
				if nbSymlinks < rtTest.MinSymlinks || nbSymlinks > rtTest.MaxSymlinks {
					t.Errorf("Dirs %d : want nb Dirs to be between %d and %d, got %d",
						i, rtTest.MinSymlinks, rtTest.MaxSymlinks, nbSymlinks)
				}
			}
		}
	})

	t.Run("RndTreeAlreadyCreated", func(t *testing.T) {
		rand.Seed(randSeed)

		for i, rtTest := range rtTests {
			path := vfs.Join(testDir, "RndTree", strconv.Itoa(i))

			rt, err := avfs.NewRndTree(vfs, path, rtTest)
			CheckNoError(t, "NewRndTree", err)

			err = rt.CreateDirs()
			CheckNoError(t, "CreateDirs", err)

			err = rt.CreateFiles()
			CheckNoError(t, "CreateFile", err)

			if vfs.HasFeature(avfs.FeatSymlink) {
				err = rt.CreateSymlinks()
				CheckNoError(t, "CreateSymlinks", err)
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

	if avfs.CurrentOSType() == avfs.OsWindows {
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

// TestSegmentUnixPath tests avfs.SegmentUnixPath function.
func (sfs *SuiteFS) TestSegmentUnixPath(t *testing.T, testDir string) {
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
			end, isLast = avfs.SegmentUnixPath(c.path, start)
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
