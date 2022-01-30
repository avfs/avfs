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
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
)

const (
	defaultUmask = fs.FileMode(0o22) // defaultUmask is the default umask.
)

type pathTest struct {
	path, result string
}

// TestAbs test Abs function.
func (sfs *SuiteFS) TestAbs(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Abs(testDir)
		CheckNoError(t, "Abs "+testDir, err)

		return
	}

	t.Run("Abs", func(t *testing.T) {
		// Test directories relative to temporary directory.
		// The tests are run in absTestDirs[0].
		absTestDirs := []string{
			"a",
			"a/b",
			"a/b/c",
		}

		// Test paths relative to temporary directory. $ expands to the directory.
		// The tests are run in absTestDirs[0].
		// We create absTestDirs first.
		absTests := []string{
			".",
			"b",
			"b/",
			"../a",
			"../a/b",
			"../a/b/./c/../../.././a",
			"../a/b/./c/../../.././a/",
			"$",
			"$/.",
			"$/a/../a/b",
			"$/a/b/c/../../.././a",
			"$/a/b/c/../../.././a/",
		}

		wd, err := vfs.Getwd()
		if err != nil {
			t.Fatal("getwd failed: ", err)
		}

		err = vfs.Chdir(testDir)
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		defer vfs.Chdir(wd) //nolint:errcheck // Ignore errors.

		for _, dir := range absTestDirs {
			err = vfs.Mkdir(dir, 0o777)
			if err != nil {
				t.Fatal("Mkdir failed: ", err)
			}
		}

		err = vfs.Chdir(absTestDirs[0])
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		vfs = sfs.vfsTest

		for _, path := range absTests {
			path = strings.ReplaceAll(path, "$", testDir)

			info, err := vfs.Stat(path)
			if err != nil {
				t.Errorf("%s: %s", path, err)

				continue
			}

			abspath, err := vfs.Abs(path)
			if err != nil {
				t.Errorf("Abs(%q) error: %v", path, err)

				continue
			}

			absinfo, err := vfs.Stat(abspath)
			if err != nil || !vfs.SameFile(absinfo, info) {
				t.Errorf("Abs(%q)=%q, not the same file", path, abspath)
			}

			if !vfs.IsAbs(abspath) {
				t.Errorf("Abs(%q)=%q, not an absolute path", path, abspath)
			}

			if vfs.IsAbs(abspath) && abspath != vfs.Clean(abspath) {
				t.Errorf("Abs(%q)=%q, isn't clean", path, abspath)
			}
		}
	})

	// AbsEmptyString tests Abs functions with an empty string input.
	// Empty path needs to be special-cased on Windows. See golang.org/issue/24441.
	// We test it separately from all other absTests because the empty string is not
	// a valid path, so it can't be used with os.Stat.
	t.Run("AbsEmptyString", func(t *testing.T) {
		wd, err := vfs.Getwd()
		if err != nil {
			t.Fatal("getwd failed: ", err)
		}

		err = vfs.Chdir(testDir)
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		defer vfs.Chdir(wd) //nolint:errcheck // Ignore errors.

		info, err := vfs.Stat(testDir)
		if err != nil {
			t.Fatalf("%s: %s", testDir, err)
		}

		abspath, err := vfs.Abs("")
		if err != nil {
			t.Fatalf(`Abs("") error: %v`, err)
		}

		absinfo, err := vfs.Stat(abspath)
		if err != nil || !vfs.SameFile(absinfo, info) {
			t.Errorf(`Abs("")=%q, not the same file`, abspath)
		}

		if !vfs.IsAbs(abspath) {
			t.Errorf(`Abs("")=%q, not an absolute path`, abspath)
		}

		if vfs.IsAbs(abspath) && abspath != vfs.Clean(abspath) {
			t.Errorf(`Abs("")=%q, isn't clean`, abspath)
		}
	})
}

// TestBase tests Base function.
func (sfs *SuiteFS) TestBase(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	var baseTests []*pathTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		baseTests = []*pathTest{
			{`c:\`, `\`},
			{`c:.`, `.`},
			{`c:\a\b`, `b`},
			{`c:a\b`, `b`},
			{`c:a\b\c`, `c`},
			{`\\host\share\`, `\`},
			{`\\host\share\a`, `a`},
			{`\\host\share\a\b`, `b`},
		}
	default:
		baseTests = []*pathTest{
			{"", "."},
			{".", "."},
			{"/.", "."},
			{"/", "/"},
			{"////", "/"},
			{"x/", "x"},
			{"abc", "abc"},
			{"abc/def", "def"},
			{"a/b/.x", ".x"},
			{"a/b/c.", "c."},
			{"a/b/c.x", "c.x"},
		}
	}

	for _, test := range baseTests {
		s := vfs.Base(test.path)
		if s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

// TestClean tests Clean function.
func (sfs *SuiteFS) TestClean(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	var cleanTests []*pathTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		cleanTests = []*pathTest{
			{`c:`, `c:.`},
			{`c:\`, `c:\`},
			{`c:\abc`, `c:\abc`},
			{`c:abc\..\..\.\.\..\def`, `c:..\..\def`},
			{`c:\abc\def\..\..`, `c:\`},
			{`c:\..\abc`, `c:\abc`},
			{`c:..\abc`, `c:..\abc`},
			{`\`, `\`},
			{`/`, `\`},
			{`\\i\..\c$`, `\c$`},
			{`\\i\..\i\c$`, `\i\c$`},
			{`\\i\..\I\c$`, `\I\c$`},
			{`\\host\share\foo\..\bar`, `\\host\share\bar`},
			{`//host/share/foo/../baz`, `\\host\share\baz`},
			{`\\a\b\..\c`, `\\a\b\c`},
			{`\\a\b`, `\\a\b`},
		}
	default:
		cleanTests = []*pathTest{
			// Already clean
			{"abc", "abc"},
			{"abc/def", "abc/def"},
			{"a/b/c", "a/b/c"},
			{".", "."},
			{"..", ".."},
			{"../..", "../.."},
			{"../../abc", "../../abc"},
			{"/abc", "/abc"},
			{"/", "/"},

			// Empty is current dir
			{"", "."},

			// Remove trailing slash
			{"abc/", "abc"},
			{"abc/def/", "abc/def"},
			{"a/b/c/", "a/b/c"},
			{"./", "."},
			{"../", ".."},
			{"../../", "../.."},
			{"/abc/", "/abc"},

			// Remove doubled slash
			{"abc//def//ghi", "abc/def/ghi"},
			{"//abc", "/abc"},
			{"///abc", "/abc"},
			{"//abc//", "/abc"},
			{"abc//", "abc"},

			// Remove . elements
			{"abc/./def", "abc/def"},
			{"/./abc/def", "/abc/def"},
			{"abc/.", "abc"},

			// Remove .. elements
			{"abc/def/ghi/../jkl", "abc/def/jkl"},
			{"abc/def/../ghi/../jkl", "abc/jkl"},
			{"abc/def/..", "abc"},
			{"abc/def/../..", "."},
			{"/abc/def/../..", "/"},
			{"abc/def/../../..", ".."},
			{"/abc/def/../../..", "/"},
			{"abc/def/../../../ghi/jkl/../../../mno", "../../mno"},
			{"/../abc", "/abc"},

			// Combinations
			{"abc/./../def", "def"},
			{"abc//./../def", "def"},
			{"abc/../../././../def", "../../def"},
		}
	}

	for _, test := range cleanTests {
		s := vfs.Clean(test.path)
		if s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}

		s = vfs.Clean(test.result)
		if s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

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

			wantSum, err := avfs.CopyFileHash(dstFs, srcFs, dstPath, srcFile.Name, h)
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

			wantSum, err := avfs.CopyFileHash(dstFs, srcFs, dstPath, srcFile.Name, nil)
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

// TestCreateBaseDirs tests CreateBaseDirs function.
func (sfs *SuiteFS) TestCreateBaseDirs(t *testing.T, testDir string) {
	vfs := sfs.VFSSetup()
	ut := vfs.Utils()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := ut.CreateBaseDirs(vfs, testDir)
	if !CheckNoError(t, "CreateBaseDirs", err) {
		return
	}

	for _, dir := range ut.BaseDirs(testDir) {
		info, err := vfs.Stat(dir.Path)
		if !CheckNoError(t, "Stat "+dir.Path, err) {
			continue
		}

		gotMode := info.Mode() & fs.ModePerm
		if gotMode != dir.Perm {
			t.Errorf("CreateBaseDirs %s :  want mode to be %o, got %o", dir.Path, dir.Perm, gotMode)
		}
	}
}

// TestCreateHomeDir tests that the user home directory exists and has the correct permissions.
func (sfs *SuiteFS) TestCreateHomeDir(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return
	}

	ut := vfs.Utils()

	for _, ui := range UserInfos() {
		u, err := vfs.Idm().LookupUser(ui.Name)
		if err != nil {
			t.Fatalf("")
		}

		homeDir, err := ut.CreateHomeDir(vfs, u)
		if !CheckNoError(t, "CreateHomeDir "+ui.Name, err) {
			continue
		}

		fst, err := vfs.Stat(homeDir)
		if !CheckNoError(t, "Stat "+homeDir, err) {
			continue
		}

		if vfs.OSType() == avfs.OsWindows {
			return
		}

		wantMode := fs.ModeDir | ut.HomeDirPerm()&^vfs.UMask()
		if fst.Mode() != wantMode {
			t.Errorf("Stat %s : want mode to be %o, got %o", homeDir, wantMode, fst.Mode())
		}

		sst := vfs.ToSysStat(fst)

		uid, gid := sst.Uid(), sst.Gid()
		if uid != u.Uid() || gid != u.Gid() {
			t.Errorf("Stat %s : want uid=%d, gid=%d, got uid=%d, gid=%d", homeDir, u.Uid(), u.Gid(), uid, gid)
		}
	}
}

// TestDir tests Dir function.
func (sfs *SuiteFS) TestDir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	var dirTests []*pathTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		dirTests = []*pathTest{
			{`c:\`, `c:\`},
			{`c:.`, `c:.`},
			{`c:\a\b`, `c:\a`},
			{`c:a\b`, `c:a`},
			{`c:a\b\c`, `c:a\b`},
			{`\\host\share`, `\\host\share`},
			{`\\host\share\`, `\\host\share\`},
			{`\\host\share\a`, `\\host\share\`},
			{`\\host\share\a\b`, `\\host\share\a`},
		}
	default:
		dirTests = []*pathTest{
			{"", "."},
			{".", "."},
			{"/.", "/"},
			{"/", "/"},
			{"////", "/"},
			{"/foo", "/"},
			{"x/", "x"},
			{"abc", "."},
			{"abc/def", "abc"},
			{"a/b/.x", "a/b"},
			{"a/b/c.", "a/b"},
			{"a/b/c.x", "a/b"},
		}
	}

	for _, test := range dirTests {
		s := vfs.Dir(test.path)
		if s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
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

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckNoError(t, "Stat "+invalidPath, err)
		default:
			CheckPathError(t, err).OpStat().Path(invalidPath).
				Err(avfs.ErrNotADirectory, avfs.OsLinux)
		}

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})
}

// TestFromToSlash tests FromSlash and ToSlash functions.
func (sfs *SuiteFS) TestFromToSlash(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	sep := byte('/')
	if vfs.OSType() == avfs.OsWindows {
		sep = '\\'
	}

	slashTests := []*pathTest{
		{"", ""},
		{"/", string(sep)},
		{"/a/b", string([]byte{sep, 'a', sep, 'b'})},
		{"a//b", string([]byte{'a', sep, sep, 'b'})},
	}

	for _, test := range slashTests {
		if s := vfs.FromSlash(test.path); s != test.result {
			t.Errorf("FromSlash(%q) = %q, want %q", test.path, s, test.result)
		}

		if s := vfs.ToSlash(test.result); s != test.path {
			t.Errorf("ToSlash(%q) = %q, want %q", test.result, s, test.path)
		}
	}
}

// TestGlob tests Glob function.
func (sfs *SuiteFS) TestGlob(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Glob("")
		CheckNoError(t, "Glob", err)

		return
	}

	_ = sfs.SampleDirs(t, testDir)
	_ = sfs.SampleFiles(t, testDir)
	sl := len(sfs.SampleSymlinks(t, testDir))

	vfs = sfs.vfsTest

	t.Run("GlobNormal", func(t *testing.T) {
		pattern := testDir + "/*/*/[A-Z0-9]"
		dirNames, err := vfs.Glob(pattern)
		if !CheckNoError(t, "Glob "+pattern, err) {
			return
		}

		wantDirs := 3
		if sl > 0 {
			wantDirs += 5
		}

		if len(dirNames) != wantDirs {
			t.Errorf("Glob %s : want dirs to be %d, got %d", pattern, wantDirs, len(dirNames))
			for _, dirName := range dirNames {
				t.Log(dirName)
			}
		}
	})

	t.Run("GlobWithoutMeta", func(t *testing.T) {
		pattern := testDir
		dirNames, err := vfs.Glob(pattern)
		if !CheckNoError(t, "Glob "+pattern, err) {
			return
		}

		if len(dirNames) != 1 {
			t.Errorf("Glob %s : want dirs to be %d, got %d", pattern, 1, len(dirNames))
			for _, dirName := range dirNames {
				t.Log(dirName)
			}
		}
	})

	t.Run("GlobWithoutMetaNonExisting", func(t *testing.T) {
		pattern := vfs.Join(testDir, "/NonExisting")
		dirNames, err := vfs.Glob(pattern)
		if dirNames != nil || err != nil {
			t.Errorf("Glob %s : want error and result to be nil, got %s, %v", pattern, dirNames, err)
		}
	})

	t.Run("GlobError", func(t *testing.T) {
		patterns := []string{
			"[]",
			testDir + "/[A-Z",
		}

		for _, pattern := range patterns {
			_, err := vfs.Glob(pattern)
			if err != filepath.ErrBadPattern {
				t.Errorf("Glob %s : want error to be %v, got %v", pattern, filepath.ErrBadPattern, err)
			}
		}
	})
}

// TestIsAbs tests IsAbs function.
func (sfs *SuiteFS) TestIsAbs(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	t.Run("IsAbs", func(t *testing.T) {
		type IsAbsTest struct {
			path  string
			isAbs bool
		}

		var isAbsTests []IsAbsTest

		switch vfs.OSType() {
		case avfs.OsWindows:
			isAbsTests = []IsAbsTest{
				{`C:\`, true},
				{`c\`, false},
				{`c::`, false},
				{`c:`, false},
				{`/`, false},
				{`\`, false},
				{`\Windows`, false},
				{`c:a\b`, false},
				{`c:\a\b`, true},
				{`c:/a/b`, true},
				{`\\host\share\foo`, true},
				{`//host/share/foo/bar`, true},
			}

		default:
			isAbsTests = []IsAbsTest{
				{"", false},
				{"/", true},
				{"/usr/bin/gcc", true},
				{"..", false},
				{"/a/../bb", true},
				{".", false},
				{"./", false},
				{"lala", false},
			}
		}

		for _, test := range isAbsTests {
			r := vfs.IsAbs(test.path)
			if r != test.isAbs {
				t.Errorf("IsAbs(%q) = %v, want %v", test.path, r, test.isAbs)
			}
		}
	})

	t.Run("IsPathSeparator", func(t *testing.T) {
		isPathSepTests := []struct {
			sep   uint8
			isAbs bool
		}{
			{sep: '/', isAbs: true},
			{sep: '\\', isAbs: vfs.OSType() == avfs.OsWindows},
			{sep: '.', isAbs: false},
			{sep: 'a', isAbs: false},
		}

		for _, test := range isPathSepTests {
			r := vfs.IsPathSeparator(test.sep)
			if r != test.isAbs {
				t.Errorf("IsPathSeparator(%q) = %v, want %v", test.sep, r, test.isAbs)
			}
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
		CheckPathError(t, err).OpStat().Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

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

// TestJoin tests Join function.
func (sfs *SuiteFS) TestJoin(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	type joinTest struct { //nolint:govet // no fieldalignment for test structs
		elem []string
		path string
	}

	var joinTests []*joinTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		joinTests = []*joinTest{
			{elem: []string{`directory`, `file`}, path: `directory\file`},
			{elem: []string{`C:\Windows\`, `System32`}, path: `C:\Windows\System32`},
			{elem: []string{`C:\Windows\`, ``}, path: `C:\Windows`},
			{elem: []string{`C:\`, `Windows`}, path: `C:\Windows`},
			{elem: []string{`C:`, `a`}, path: `C:a`},
			{elem: []string{`C:`, `a\b`}, path: `C:a\b`},
			{elem: []string{`C:`, `a`, `b`}, path: `C:a\b`},
			{elem: []string{`C:`, ``, `b`}, path: `C:b`},
			{elem: []string{`C:`, ``, ``, `b`}, path: `C:b`},
			{elem: []string{`C:`, ``}, path: `C:.`},
			{elem: []string{`C:`, ``, ``}, path: `C:.`},
			{elem: []string{`C:.`, `a`}, path: `C:a`},
			{elem: []string{`C:a`, `b`}, path: `C:a\b`},
			{elem: []string{`C:a`, `b`, `d`}, path: `C:a\b\d`},
			{elem: []string{`\\host\share`, `foo`}, path: `\\host\share\foo`},
			{elem: []string{`\\host\share\foo`}, path: `\\host\share\foo`},
			{elem: []string{`//host/share`, `foo/bar`}, path: `\\host\share\foo\bar`},
			{elem: []string{`\`}, path: `\`},
			{elem: []string{`\`, ``}, path: `\`},
			{elem: []string{`\`, `a`}, path: `\a`},
			{elem: []string{`\\`, `a`}, path: `\a`},
			{elem: []string{`\`, `a`, `b`}, path: `\a\b`},
			{elem: []string{`\\`, `a`, `b`}, path: `\a\b`},
			{elem: []string{`\`, `\\a\b`, `c`}, path: `\a\b\c`},
			{elem: []string{`\\a`, `b`, `c`}, path: `\a\b\c`},
			{elem: []string{`\\a\`, `b`, `c`}, path: `\a\b\c`},
		}
	default:
		joinTests = []*joinTest{
			// zero parameters
			{elem: []string{}},

			// one parameter
			{elem: []string{""}},
			{elem: []string{"a"}, path: "a"},

			// two parameters
			{elem: []string{"a", "b"}, path: "a/b"},
			{elem: []string{"a", ""}, path: "a"},
			{elem: []string{"", "b"}, path: "b"},
			{elem: []string{"/", "a"}, path: "/a"},
			{elem: []string{"/", ""}, path: "/"},
			{elem: []string{"a/", "b"}, path: "a/b"},
			{elem: []string{"a/", ""}, path: "a"},
			{elem: []string{"", ""}},
		}
	}

	for _, test := range joinTests {
		p := vfs.Join(test.elem...)
		if p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
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
	vfs := sfs.vfsTest

	matchTests := []struct { //nolint:govet // no fieldalignment for test structs
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

		if vfs.OSType() == avfs.OsWindows {
			if strings.Contains(pattern, "\\") {
				// no escape allowed on Windows.
				continue
			}

			pattern = vfs.Clean(pattern)
			s = vfs.Clean(s)
		}

		ok, err := vfs.Match(pattern, s)
		if ok != tt.match || err != tt.err {
			t.Errorf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}

// TestPathIterator tests PathIterator methods.
func (sfs *SuiteFS) TestPathIterator(t *testing.T, testDir string) {
	utLinux := avfs.NewUtils(avfs.OsLinux)
	utWindows := avfs.NewUtils(avfs.OsWindows)

	t.Run("PathIterator", func(t *testing.T) {
		cases := []struct {
			path   string
			parts  []string
			osType avfs.OSType
		}{
			{path: `C:\`, parts: nil, osType: avfs.OsWindows},
			{path: `C:\Users`, parts: []string{"Users"}, osType: avfs.OsWindows},
			{path: `c:\नमस्ते\दुनिया`, parts: []string{"नमस्ते", "दुनिया"}, osType: avfs.OsWindows},

			{path: "/", parts: nil, osType: avfs.OsLinux},
			{path: "/a", parts: []string{"a"}, osType: avfs.OsLinux},
			{path: "/b/c/d", parts: []string{"b", "c", "d"}, osType: avfs.OsLinux},
			{path: "/नमस्ते/दुनिया", parts: []string{"नमस्ते", "दुनिया"}, osType: avfs.OsLinux},
		}

		for _, c := range cases {
			var ut avfs.Utils

			switch c.osType {
			case avfs.OsLinux:
				ut = utLinux
			case avfs.OsWindows:
				ut = utWindows
			}

			pi := ut.NewPathIterator(c.path)
			i := 0
			gotPath := pi.VolumeName() + string(ut.PathSeparator())

			for ; pi.Next(); i++ {
				if pi.Part() != c.parts[i] {
					t.Errorf("%s : want part %d to be %s, got %s", c.path, i, c.parts[i], pi.Part())
				}

				wantLeft := pi.Path()[:pi.Start()]
				if pi.Left() != wantLeft {
					t.Errorf("%s : want left %d to be %s, got %s", c.path, i, wantLeft, pi.Left())
				}

				wantLeftPart := pi.Path()[:pi.End()]
				if pi.LeftPart() != wantLeftPart {
					t.Errorf("%s : want left %d to be %s, got %s", c.path, i, wantLeftPart, pi.LeftPart())
				}

				wantRight := pi.Path()[pi.End():]
				if pi.Right() != wantRight {
					t.Errorf("%s : want right %d to be %s, got %s", c.path, i, wantRight, pi.Right())
				}

				wantRightPart := pi.Path()[pi.Start():]
				if pi.RightPart() != wantRightPart {
					t.Errorf("%s : want right %d to be %s, got %s", c.path, i, wantRightPart, pi.RightPart())
				}

				wantIsLast := i == (len(c.parts) - 1)
				if pi.IsLast() != wantIsLast {
					t.Errorf("%s : want IsLast %d to be %t, got %t", c.path, i, wantIsLast, pi.IsLast())
				}

				gotPath = ut.Join(gotPath, pi.Part())
			}

			if gotPath != pi.Path() {
				t.Errorf("%s : want path to be %s, got %s", c.path, c.path, gotPath)
			}
		}
	})

	t.Run("ReplacePart", func(t *testing.T) {
		cases := []struct {
			path    string
			part    string
			newPart string
			newPath string
			reset   bool
			osType  avfs.OSType
		}{
			{
				path: `c:\path`, part: `path`, newPart: `..\..\..`,
				newPath: `c:\`, reset: true, osType: avfs.OsWindows,
			},
			{
				path: `c:\an\absolute\path`, part: `absolute`, newPart: `c:\just\another`,
				newPath: `c:\just\another\path`, reset: true, osType: avfs.OsWindows,
			},
			{
				path: `c:\a\random\path`, part: `random`, newPart: `very\long`,
				newPath: `c:\a\very\long\path`, reset: false, osType: avfs.OsWindows,
			},
			{
				path: "/a/very/very/long/path", part: "long", newPart: "/a",
				newPath: "/a/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/path", part: "path", newPart: "../../..",
				newPath: "/", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/a/relative/path", part: "relative", newPart: "../../..",
				newPath: "/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/an/absolute/path", part: "random", newPart: "/just/another",
				newPath: "/just/another/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/a/relative/path", part: "relative", newPart: "very/long",
				newPath: "/a/very/long/path", reset: false, osType: avfs.OsLinux,
			},
		}

		for _, c := range cases {
			var ut avfs.Utils

			switch c.osType {
			case avfs.OsLinux:
				ut = utLinux
			case avfs.OsWindows:
				ut = utWindows
			}

			pi := ut.NewPathIterator(c.path)
			for pi.Next() {
				if pi.Part() == c.part {
					reset := pi.ReplacePart(c.newPart)
					if pi.Path() != c.newPath {
						t.Errorf("%s : want new path to be %s, got %s", c.path, c.newPath, pi.Path())
					}

					if reset != c.reset {
						t.Errorf("%s : want Reset to be %t, got %t", c.path, c.reset, reset)
					}

					break
				}
			}
		}
	})
}

// TestRel tests Rel function.
func (sfs *SuiteFS) TestRel(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	type relTest struct {
		root, path, want string
	}

	relTests := []*relTest{
		{root: "a/b", path: "a/b", want: "."},
		{root: "a/b/.", path: "a/b", want: "."},
		{root: "a/b", path: "a/b/.", want: "."},
		{root: "./a/b", path: "a/b", want: "."},
		{root: "a/b", path: "./a/b", want: "."},
		{root: "ab/cd", path: "ab/cde", want: "../cde"},
		{root: "ab/cd", path: "ab/c", want: "../c"},
		{root: "a/b", path: "a/b/c/d", want: "c/d"},
		{root: "a/b", path: "a/b/../c", want: "../c"},
		{root: "a/b/../c", path: "a/b", want: "../b"},
		{root: "a/b/c", path: "a/c/d", want: "../../c/d"},
		{root: "a/b", path: "c/d", want: "../../c/d"},
		{root: "a/b/c/d", path: "a/b", want: "../.."},
		{root: "a/b/c/d", path: "a/b/", want: "../.."},
		{root: "a/b/c/d/", path: "a/b", want: "../.."},
		{root: "a/b/c/d/", path: "a/b/", want: "../.."},
		{root: "../../a/b", path: "../../a/b/c/d", want: "c/d"},
		{root: "/a/b", path: "/a/b", want: "."},
		{root: "/a/b/.", path: "/a/b", want: "."},
		{root: "/a/b", path: "/a/b/.", want: "."},
		{root: "/ab/cd", path: "/ab/cde", want: "../cde"},
		{root: "/ab/cd", path: "/ab/c", want: "../c"},
		{root: "/a/b", path: "/a/b/c/d", want: "c/d"},
		{root: "/a/b", path: "/a/b/../c", want: "../c"},
		{root: "/a/b/../c", path: "/a/b", want: "../b"},
		{root: "/a/b/c", path: "/a/c/d", want: "../../c/d"},
		{root: "/a/b", path: "/c/d", want: "../../c/d"},
		{root: "/a/b/c/d", path: "/a/b", want: "../.."},
		{root: "/a/b/c/d", path: "/a/b/", want: "../.."},
		{root: "/a/b/c/d/", path: "/a/b", want: "../.."},
		{root: "/a/b/c/d/", path: "/a/b/", want: "../.."},
		{root: "/../../a/b", path: "/../../a/b/c/d", want: "c/d"},
		{root: ".", path: "a/b", want: "a/b"},
		{root: ".", path: "..", want: ".."},

		// can't do purely lexically
		{root: "..", path: ".", want: "err"},
		{root: "..", path: "a", want: "err"},
		{root: "../..", path: "..", want: "err"},
		{root: "a", path: "/a", want: "err"},
		{root: "/a", path: "a", want: "err"},
	}

	relTestsWin := []*relTest{
		{root: `C:a\b\c`, path: `C:a/b/d`, want: `..\d`},
		{root: `C:\`, path: `D:\`, want: `err`},
		{root: `C:`, path: `D:`, want: `err`},
		{root: `C:\Projects`, path: `c:\projects\src`, want: `src`},
		{root: `C:\Projects`, path: `c:\projects`, want: `.`},
		{root: `C:\Projects\a\..`, path: `c:\projects`, want: `.`},
	}

	if vfs.OSType() == avfs.OsWindows {
		relTests = append(relTests, relTestsWin...)
		for i := range relTests {
			relTests[i].want = filepath.FromSlash(relTests[i].want)
		}
	}

	for _, test := range relTests {
		got, err := vfs.Rel(test.root, test.path)
		if test.want == "err" {
			if err == nil {
				t.Errorf("Rel(%q, %q)=%q, want error", test.root, test.path, got)
			}

			continue
		}

		if err != nil {
			t.Errorf("Rel(%q, %q): want %q, got error: %s", test.root, test.path, test.want, err)
		}

		if got != test.want {
			t.Errorf("Rel(%q, %q)=%q, want %q", test.root, test.path, got, test.want)
		}
	}
}

// TestRndTree tests RndTree methods.
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

// TestSplit tests Split function.
func (sfs *SuiteFS) TestSplit(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	type splitTest struct {
		path, dir, file string
	}

	var splitTests []*splitTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		splitTests = []*splitTest{
			{path: `c:`, dir: `c:`},
			{path: `c:/`, dir: `c:/`},
			{path: `c:/foo`, dir: `c:/`, file: `foo`},
			{path: `c:/foo/bar`, dir: `c:/foo/`, file: `bar`},
			{path: `//host/share`, dir: `//host/share`},
			{path: `//host/share/`, dir: `//host/share/`},
			{path: `//host/share/foo`, dir: `//host/share/`, file: `foo`},
			{path: `\\host\share`, dir: `\\host\share`},
			{path: `\\host\share\`, dir: `\\host\share\`},
			{path: `\\host\share\foo`, dir: `\\host\share\`, file: `foo`},
		}
	default:
		splitTests = []*splitTest{
			{path: "a/b", dir: "a/", file: "b"},
			{path: "a/b/", dir: "a/b/"},
			{path: "a/", dir: "a/"},
			{path: "a", file: "a"},
			{path: "/", dir: "/"},
		}
	}

	for _, test := range splitTests {
		d, f := vfs.Split(test.path)
		if d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}

// TestUMask tests Umask methods.
func (sfs *SuiteFS) TestUMask(t *testing.T, testDir string) {
	const umaskSet = fs.FileMode(0o77)

	umaskTest := umaskSet

	if avfs.Cfg.OSType() == avfs.OsWindows {
		umaskTest = defaultUmask
	}

	var um avfs.UMaskType

	umask := um.Get()
	if umask != defaultUmask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUmask, umask)
	}

	um.Set(umaskSet)

	umask = um.Get()
	if umask != umaskTest {
		t.Errorf("UMask : want test umask %o, got %o", umaskTest, umask)
	}

	um.Set(defaultUmask)

	umask = um.Get()
	if umask != defaultUmask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUmask, umask)
	}
}

// TestWalkDir tests WalkDir function.
func (sfs *SuiteFS) TestWalkDir(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		walkFunc := func(rootDir string, info fs.DirEntry, err error) error { return nil }

		err := vfs.WalkDir(testDir, walkFunc)
		CheckNoError(t, "WalkDir "+testDir, err)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest
	lNames := len(dirs) + len(files) + len(symlinks)
	wantNames := make([]string, 0, lNames)

	wantNames = append(wantNames, testDir)
	for _, dir := range dirs {
		wantNames = append(wantNames, dir.Path)
	}

	for _, file := range files {
		wantNames = append(wantNames, file.Path)
	}

	if vfs.HasFeature(avfs.FeatSymlink) {
		for _, sl := range symlinks {
			wantNames = append(wantNames, sl.NewPath)
		}
	}

	sort.Strings(wantNames)

	t.Run("WalkDir", func(t *testing.T) {
		gotNames := make(map[string]int)
		err := vfs.WalkDir(testDir, func(path string, info fs.DirEntry, err error) error {
			gotNames[path]++

			return nil
		})
		CheckNoError(t, "WalkDir"+testDir, err)

		if len(wantNames) != len(gotNames) {
			t.Errorf("Walk %s : want %d files or dirs, got %d", testDir, len(wantNames), len(gotNames))
		}

		for _, wantName := range wantNames {
			n, ok := gotNames[wantName]
			if !ok || n != 1 {
				t.Errorf("Walk %s : path %s not found", testDir, wantName)
			}
		}
	})

	t.Run("WalkNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.WalkDir(nonExistingFile, func(path string, info fs.DirEntry, err error) error {
			return nil
		})
		CheckNoError(t, "WalkDir "+nonExistingFile, err)
	})
}
