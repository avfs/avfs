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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
)

type pathTest struct {
	path, result string
}

func (ts *Suite) TestUtils(t *testing.T) {
	ts.RunTests(t, UsrTest,
		ts.TestCopyFile,
		ts.TestDirExists,
		ts.TestExists,
		ts.TestFromUnixPath,
		ts.TestHashFile,
		ts.TestIsDir,
		ts.TestIsEmpty,
		ts.TestIsPathSeparator,
		ts.TestRndTree,
		ts.TestUMask)
}

// TestAbs test Abs function.
func (ts *Suite) TestAbs(t *testing.T, testDir string) {
	vfs := ts.vfsSetup

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

		vfs = ts.vfsTest

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
func (ts *Suite) TestBase(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
func (ts *Suite) TestClean(t *testing.T, _ string) {
	vfs := ts.vfsTest

	cleanTests := []*pathTest{
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

	nonWinCleanTests := []*pathTest{
		// Remove leading doubled slash
		{"//abc", "/abc"},
		{"///abc", "/abc"},
		{"//abc//", "/abc"},
	}

	winCleanTests := []*pathTest{
		{`c:`, `c:.`},
		{`c:\`, `c:\`},
		{`c:\abc`, `c:\abc`},
		{`c:abc\..\..\.\.\..\def`, `c:..\..\def`},
		{`c:\abc\def\..\..`, `c:\`},
		{`c:\..\abc`, `c:\abc`},
		{`c:..\abc`, `c:..\abc`},
		{`\`, `\`},
		{`/`, `\`},
		{`\\i\..\c$`, `\\i\..\c$`},
		{`\\i\..\i\c$`, `\\i\..\i\c$`},
		{`\\i\..\I\c$`, `\\i\..\I\c$`},
		{`\\host\share\foo\..\bar`, `\\host\share\bar`},
		{`//host/share/foo/../baz`, `\\host\share\baz`},
		{`\\host\share\foo\..\..\..\..\bar`, `\\host\share\bar`},
		{`\\.\C:\a\..\..\..\..\bar`, `\\.\C:\bar`},
		{`\\.\C:\\\\a`, `\\.\C:\a`},
		{`\\a\b\..\c`, `\\a\b\c`},
		{`\\a\b`, `\\a\b`},
		{`.\c:`, `.\c:`},
		{`.\c:\foo`, `.\c:\foo`},
		{`.\c:foo`, `.\c:foo`},
		{`//abc`, `\\abc`},
		{`///abc`, `\\\abc`},
		{`//abc//`, `\\abc\\`},

		// Don't allow cleaning to move an element with a colon to the start of the path.
		{`a/../c:`, `.\c:`},
		{`a\..\c:`, `.\c:`},
		{`a/../c:/a`, `.\c:\a`},
		{`a/../../c:`, `..\c:`},
		{`foo:bar`, `foo:bar`},
	}

	tests := cleanTests
	if vfs.OSType() == avfs.OsWindows {
		for i := range tests {
			tests[i].result = filepath.FromSlash(tests[i].result)
		}

		tests = append(tests, winCleanTests...)
	} else {
		tests = append(tests, nonWinCleanTests...)
	}

	for _, test := range tests {
		if s := filepath.Clean(test.path); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}

		if s := filepath.Clean(test.result); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

// TestCopyFile tests avfs.CopyFile function.
func (ts *Suite) TestCopyFile(t *testing.T, testDir string) {
	const copyFile = "CopyFile"

	srcFS := ts.vfsSetup
	dstFS := memfs.New()

	rt := avfs.NewRndTree(srcFS, &avfs.RndTreeOpts{NbFiles: 32, MaxFileSize: 100 * 1024})

	err := rt.CreateTree(testDir)
	RequireNoError(t, err, "CreateTree %s")

	h := sha512.New()

	t.Run("CopyFile_WithHashSum", func(t *testing.T) {
		dstDir, err := dstFS.MkdirTemp("", copyFile)
		RequireNoError(t, err, "MkdirTemp")

		defer dstFS.RemoveAll(dstDir) //nolint:errcheck // Ignore errors.

		for _, srcFile := range rt.Files() {
			srcPath := srcFS.Join(testDir, srcFile.Name)
			fileName := srcFS.Base(srcFile.Name)
			dstPath := dstFS.Join(dstDir, fileName)

			wantSum, err := avfs.CopyFileHash(dstFS, srcFS, dstPath, srcPath, h)
			RequireNoError(t, err, "CopyFile (%s)%s, (%s)%s",
				dstFS.Type(), dstPath, srcFS.Type(), srcFile.Name)

			gotSum, err := avfs.HashFile(dstFS, dstPath, h)
			RequireNoError(t, err, "HashFile (%s)%s", dstFS.Type(), dstPath)

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		dstDir, err := dstFS.MkdirTemp("", copyFile)
		RequireNoError(t, err, "MkdirTemp %s", copyFile)

		for _, srcFile := range rt.Files() {
			srcPath := srcFS.Join(testDir, srcFile.Name)
			fileName := srcFS.Base(srcFile.Name)
			dstPath := dstFS.Join(dstDir, fileName)

			err = avfs.CopyFile(dstFS, srcFS, dstPath, srcPath)
			RequireNoError(t, err, "CopyFile (%s)%s, (%s)%s",
				dstFS.Type(), dstPath, srcFS.Type(), srcFile.Name)

			wantSum, err := avfs.HashFile(srcFS, srcPath, h)
			RequireNoError(t, err, "HashFile (%s)%s", srcFS.Type(), srcFile.Name)

			gotSum, err := avfs.HashFile(dstFS, dstPath, h)
			RequireNoError(t, err, "HashFile (%s)%s", dstFS.Type(), dstPath)

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})
}

// TestMkSystemDirs tests CreateSystemDirs function.
func (ts *Suite) TestMkSystemDirs(t *testing.T, testDir string) {
	vfs := ts.vfsSetup
	dirs := avfs.SystemDirs(vfs, testDir)

	err := avfs.MkSystemDirs(vfs, dirs)
	RequireNoError(t, err, "MkSystemDirs %s", testDir)

	for _, dir := range avfs.SystemDirs(vfs, testDir) {
		info, err := vfs.Stat(dir.Path)
		if !AssertNoError(t, err, "Stat %s", dir.Path) {
			continue
		}

		gotMode := info.Mode() & fs.ModePerm
		if gotMode != dir.Perm {
			t.Errorf("MkSystemDirs %s :  want mode to be %o, got %o", dir.Path, dir.Perm, gotMode)
		}
	}
}

// TestCreateHomeDir tests that the user home directory exists and has the correct permissions.
func (ts *Suite) TestCreateHomeDir(t *testing.T, _ string) {
	if !ts.canTestPerm {
		return
	}

	vfs := ts.vfsSetup

	for _, ui := range UserInfos() {
		u, err := vfs.Idm().LookupUser(ui.Name)
		RequireNoError(t, err, "Can't find user %s", ui.Name)

		homeDir, err := avfs.MkHomeDir(vfs, "", u)
		if !AssertNoError(t, err, "CreateHomeDir %s", ui.Name) {
			continue
		}

		fst, err := vfs.Stat(homeDir)
		if !AssertNoError(t, err, "Stat %s", homeDir) {
			continue
		}

		err = vfs.Remove(homeDir)
		if !AssertNoError(t, err, "Remove %s", homeDir) {
			continue
		}

		if vfs.OSType() == avfs.OsWindows {
			return
		}

		wantMode := fs.ModeDir | avfs.HomeDirPerm()&^vfs.UMask()
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
func (ts *Suite) TestDir(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
func (ts *Suite) TestDirExists(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	t.Run("DirExistsDir", func(t *testing.T) {
		ok, err := avfs.DirExists(vfs, testDir)
		RequireNoError(t, err, "DirExists %s", testDir)

		if !ok {
			t.Error("DirExists : want DirExists to be true, got false")
		}
	})

	t.Run("DirExistsFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		ok, err := avfs.DirExists(vfs, existingFile)
		RequireNoError(t, err, "DirExists %s", testDir)

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})

	t.Run("DirExistsNotExisting", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		ok, err := avfs.DirExists(vfs, nonExistingFile)
		RequireNoError(t, err, "DirExists %s", nonExistingFile)

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})
}

// TestExists tests avfs.Exists function.
func (ts *Suite) TestExists(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	t.Run("ExistsDir", func(t *testing.T) {
		ok, err := avfs.Exists(vfs, testDir)
		RequireNoError(t, err, "Exists %s", testDir)

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		ok, err := avfs.Exists(vfs, existingFile)
		RequireNoError(t, err, "DirExists %s", existingFile)

		if !ok {
			t.Error("Exists : want DirExists to be true, got false")
		}
	})

	t.Run("ExistsNotExisting", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		ok, err := avfs.Exists(vfs, nonExistingFile)
		RequireNoError(t, err, "Exists %s", nonExistingFile)

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})

	t.Run("ExistsInvalidPath", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)
		invalidPath := vfs.Join(existingFile, defaultFile)

		ok, err := avfs.Exists(vfs, invalidPath)

		AssertPathError(t, err).
			OSType(avfs.OsLinux).Op("stat").Path(invalidPath).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).NoError().Test()

		if ok {
			t.Error("Exists : want Exists to be false, got true")
		}
	})
}

// TestFromToSlash tests FromSlash and ToSlash functions.
func (ts *Suite) TestFromToSlash(t *testing.T, _ string) {
	vfs := ts.vfsTest

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

func (ts *Suite) TestFromUnixPath(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	windowsRoot := avfs.DefaultVolume + "\\"

	upTests := []struct{ path, resultUnix, resultWindows string }{
		{path: "", resultUnix: "/", resultWindows: windowsRoot},
		{path: "/", resultUnix: "/", resultWindows: windowsRoot},
		{path: "/tmp", resultUnix: "/tmp", resultWindows: vfs.Join(windowsRoot, "tmp")},
		{path: "/tmp/subdir", resultUnix: "/tmp/subdir", resultWindows: vfs.Join(windowsRoot, "tmp/subdir")},
		{path: "../tmp/subdir", resultUnix: "../tmp/subdir", resultWindows: `..\tmp\subdir`},
	}

	for _, upTest := range upTests {
		want := upTest.resultUnix
		if vfs.OSType() == avfs.OsWindows {
			want = upTest.resultWindows
		}

		got := avfs.FromUnixPath(vfs, upTest.path)
		if got != want {
			t.Errorf("FromUnixPath(%q) = %q, want %q", upTest.path, got, want)
		}
	}
}

// TestGlob tests Glob function.
func (ts *Suite) TestGlob(t *testing.T, testDir string) {
	_ = ts.createSampleDirs(t, testDir)
	_ = ts.createSampleFiles(t, testDir)
	sl := len(ts.createSampleSymlinks(t, testDir))

	vfs := ts.vfsTest

	t.Run("GlobNormal", func(t *testing.T) {
		pattern := testDir + "/*/*/[A-Z0-9]"

		dirNames, err := vfs.Glob(pattern)
		RequireNoError(t, err, "Glob %s", pattern)

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
		RequireNoError(t, err, "Glob %s", pattern)

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

// TestHashFile tests avfs.HashFile function.
func (ts *Suite) TestHashFile(t *testing.T, testDir string) {
	vfs := ts.vfsSetup
	rt := avfs.NewRndTree(vfs, &avfs.RndTreeOpts{NbFiles: 100, MaxFileSize: 100 * 1024})

	err := rt.CreateTree(testDir)
	RequireNoError(t, err, "CreateTree %s", testDir)

	defer vfs.RemoveAll(testDir) //nolint:errcheck // Ignore errors.

	h := sha512.New()

	for _, file := range rt.Files() {
		path := vfs.Join(testDir, file.Name)

		content, err := vfs.ReadFile(path)
		if !AssertNoError(t, err, "ReadFile %s", path) {
			continue
		}

		h.Reset()

		_, err = h.Write(content)
		RequireNoError(t, err, "Write %s", path)

		wantSum := h.Sum(nil)

		gotSum, err := avfs.HashFile(vfs, path, h)
		RequireNoError(t, err, "HashFile %s", path)

		if !bytes.Equal(wantSum, gotSum) {
			t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", file.Name, wantSum, gotSum)
		}
	}
}

// TestIsAbs tests IsAbs function.
func (ts *Suite) TestIsAbs(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
}

// TestIsDir tests avfs.IsDir function.
func (ts *Suite) TestIsDir(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	t.Run("IsDir", func(t *testing.T) {
		existingDir := ts.existingDir(t, testDir)

		ok, err := avfs.IsDir(vfs, existingDir)
		RequireNoError(t, err, "IsDir %s", existingDir)

		if !ok {
			t.Error("IsDir : want IsDir to be true, got false")
		}
	})

	t.Run("IsDirFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		ok, err := avfs.IsDir(vfs, existingFile)
		RequireNoError(t, err, "IsDir %s", existingFile)

		if ok {
			t.Error("IsDirFile : want DirExists to be false, got true")
		}
	})

	t.Run("IsDirNonExisting", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		ok, err := avfs.IsDir(vfs, nonExistingFile)
		AssertPathError(t, err).Path(nonExistingFile).
			OSType(avfs.OsLinux).Op("stat").Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Op(avfs.OpWinCreateFile).Err(avfs.ErrWinFileNotFound).Test()

		if ok {
			t.Error("IsDirNonExisting : want DirExists to be false, got true")
		}
	})
}

// TestIsEmpty tests avfs.IsEmpty function.
func (ts *Suite) TestIsEmpty(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	t.Run("IsEmptyFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		ok, err := avfs.IsEmpty(vfs, existingFile)
		RequireNoError(t, err, "IsEmpty %s", existingFile)

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDirEmpty", func(t *testing.T) {
		emptyDir := ts.existingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, emptyDir)
		RequireNoError(t, err, "IsEmpty %s", emptyDir)

		if !ok {
			t.Error("IsEmpty : want IsEmpty to be true, got false")
		}
	})

	t.Run("IsEmptyDir", func(t *testing.T) {
		ts.existingDir(t, testDir)

		ok, err := avfs.IsEmpty(vfs, testDir)
		RequireNoError(t, err, "IsEmpty %s", testDir)

		if ok {
			t.Error("IsEmpty : want IsEmpty to be false, got true")
		}
	})

	t.Run("IsEmptyNonExisting", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

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

// TestIsPathSeparator tests IsPathSeparator function.
func (ts *Suite) TestIsPathSeparator(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
}

// TestJoin tests Join function.
func (ts *Suite) TestJoin(t *testing.T, _ string) {
	vfs := ts.vfsTest

	type joinTest struct {
		elem []string
		path string
	}

	joinTests := []*joinTest{
		// zero parameters
		{[]string{}, ""},

		// one parameter
		{[]string{""}, ""},
		{[]string{"/"}, "/"},
		{[]string{"a"}, "a"},

		// two parameters
		{[]string{"a", "b"}, "a/b"},
		{[]string{"a", ""}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"/", "a"}, "/a"},
		{[]string{"/", "a/b"}, "/a/b"},
		{[]string{"/", ""}, "/"},
		{[]string{"/a", "b"}, "/a/b"},
		{[]string{"a", "/b"}, "a/b"},
		{[]string{"/a", "/b"}, "/a/b"},
		{[]string{"a/", "b"}, "a/b"},
		{[]string{"a/", ""}, "a"},
		{[]string{"", ""}, ""},

		// three parameters
		{[]string{"/", "a", "b"}, "/a/b"},
	}

	nonWinJoinTests := []*joinTest{
		{[]string{"//", "a"}, "/a"},
	}

	winJoinTests := []*joinTest{
		{[]string{`directory`, `file`}, `directory\file`},
		{[]string{`C:\Windows\`, `System32`}, `C:\Windows\System32`},
		{[]string{`C:\Windows\`, ``}, `C:\Windows`},
		{[]string{`C:\`, `Windows`}, `C:\Windows`},
		{[]string{`C:`, `a`}, `C:a`},
		{[]string{`C:`, `a\b`}, `C:a\b`},
		{[]string{`C:`, `a`, `b`}, `C:a\b`},
		{[]string{`C:`, ``, `b`}, `C:b`},
		{[]string{`C:`, ``, ``, `b`}, `C:b`},
		{[]string{`C:`, ``}, `C:.`},
		{[]string{`C:`, ``, ``}, `C:.`},
		{[]string{`C:`, `\a`}, `C:\a`},
		{[]string{`C:`, ``, `\a`}, `C:\a`},
		{[]string{`C:.`, `a`}, `C:a`},
		{[]string{`C:a`, `b`}, `C:a\b`},
		{[]string{`C:a`, `b`, `d`}, `C:a\b\d`},
		{[]string{`\\host\share`, `foo`}, `\\host\share\foo`},
		{[]string{`\\host\share\foo`}, `\\host\share\foo`},
		{[]string{`//host/share`, `foo/bar`}, `\\host\share\foo\bar`},
		{[]string{`\`}, `\`},
		{[]string{`\`, ``}, `\`},
		{[]string{`\`, `a`}, `\a`},
		{[]string{`\\`, `a`}, `\\a`},
		{[]string{`\`, `a`, `b`}, `\a\b`},
		{[]string{`\\`, `a`, `b`}, `\\a\b`},
		{[]string{`\`, `\\a\b`, `c`}, `\a\b\c`},
		{[]string{`\\a`, `b`, `c`}, `\\a\b\c`},
		{[]string{`\\a\`, `b`, `c`}, `\\a\b\c`},
		{[]string{`//`, `a`}, `\\a`},
	}

	if vfs.OSType() == avfs.OsWindows {
		joinTests = append(joinTests, winJoinTests...)
	} else {
		joinTests = append(joinTests, nonWinJoinTests...)
	}

	for _, test := range joinTests {
		expected := filepath.FromSlash(test.path)
		if p := filepath.Join(test.elem...); p != expected {
			t.Errorf("join(%q) = %q, want %q", test.elem, p, expected)
		}
	}
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}

	return e.Error()
}

func (ts *Suite) TestMatch(t *testing.T, _ string) {
	vfs := ts.vfsTest

	matchTests := []struct {
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

// TestRel tests Rel function.
func (ts *Suite) TestRel(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
func (ts *Suite) TestRndTree(t *testing.T, testDir string) {
	vfs := ts.vfsSetup

	rtOpts := []*avfs.RndTreeOpts{
		{NbDirs: -1, NbFiles: -1, NbSymlinks: -1, MaxFileSize: -1, MaxDepth: -1},
		{NbDirs: 0, NbFiles: 0, NbSymlinks: 0, MaxFileSize: 0, MaxDepth: 0},
		{NbDirs: 0, NbFiles: 3, NbSymlinks: 3, MaxFileSize: 3, MaxDepth: 0},
		{NbDirs: 3, NbFiles: 0, NbSymlinks: 3, MaxFileSize: 0, MaxDepth: 0},
		{NbDirs: 3, NbFiles: 3, NbSymlinks: 0, MaxFileSize: 3, MaxDepth: 0},
		{NbDirs: 3, NbFiles: 3, NbSymlinks: 3, MaxFileSize: 0, MaxDepth: 0},
		{NbDirs: 3, NbFiles: 11, NbSymlinks: 4, MaxFileSize: 0, MaxDepth: 0},
		{NbDirs: 20, NbFiles: 30, NbSymlinks: 10, MaxFileSize: 2048, MaxDepth: 3},
	}

	t.Run("RndTreeGenerate", func(t *testing.T) {
		for i, rtOpt := range rtOpts {
			rt := avfs.NewRndTree(vfs, rtOpt)

			rt.GenTree()

			nbDirs := len(rt.Dirs())
			if nbDirs != rtOpt.NbDirs {
				t.Errorf("Dirs %d : want nb Dirs to be %d, got %d", i, rtOpt.NbDirs, nbDirs)
			}

			maxDepth := 0

			for _, file := range rt.Files() {
				depth := strings.Count(file.Name, "/") - 1
				if depth > maxDepth {
					maxDepth = depth
				}
			}

			if maxDepth > rt.MaxDepth {
				t.Errorf("Dirs MaxDepth %d : want MaxDepth to be <= %d, got %d", i, rt.MaxDepth, maxDepth)
			}

			nbFiles := len(rt.Files())
			if nbFiles != rtOpt.NbFiles {
				t.Errorf("Files %d : want NbFiles to be %d, got %d", i, rtOpt.NbFiles, nbFiles)
			}

			maxDepth = 0

			for _, file := range rt.Files() {
				depth := strings.Count(file.Name, "/") - 1
				if depth > maxDepth {
					maxDepth = depth
				}
			}

			if maxDepth > rt.MaxDepth {
				t.Errorf("Files MaxDepth %d : want MaxDepth to be <= %d, got %d", i, rt.MaxDepth, maxDepth)
			}

			maxSize := 0

			for _, file := range rt.Files() {
				size := file.Size
				if size > maxSize {
					maxSize = size
				}
			}

			if maxSize > rt.MaxFileSize {
				t.Errorf("MaxFileSize %d : want MaxFileSize to be <= %d, got %d", i, rt.MaxFileSize, maxSize)
			}

			nbSymlinks := len(rt.SymLinks())
			wantSymLinks := rtOpt.NbSymlinks

			if rt.NbFiles == 0 || !vfs.HasFeature(avfs.FeatSymlink) {
				wantSymLinks = 0
			}

			if nbSymlinks != wantSymLinks {
				t.Errorf("Symlinks %d : want NbSymlinks to be %d, got %d", i, wantSymLinks, nbSymlinks)
			}

			maxDepth = 0

			for _, symlink := range rt.SymLinks() {
				depth := strings.Count(symlink.NewName, "/") - 1
				if depth > maxDepth {
					maxDepth = depth
				}
			}

			if maxDepth > rt.MaxDepth {
				t.Errorf("Symlinks MaxDepth %d : want MaxDepth to be <= %d, got %d", i, rt.MaxDepth, maxDepth)
			}
		}
	})

	t.Run("RndTreeCreate", func(t *testing.T) {
		for i, rtOpt := range rtOpts {
			path := vfs.Join(testDir, strconv.Itoa(i))

			ts.createDir(t, path, avfs.DefaultDirPerm)

			rt := avfs.NewRndTree(vfs, rtOpt)

			err := rt.CreateDirs(testDir)
			RequireNoError(t, err, "CreateDirs %s", path)

			err = rt.CreateFiles(testDir)
			RequireNoError(t, err, "CreateFiles %s", path)

			if !vfs.HasFeature(avfs.FeatSymlink) {
				continue
			}

			err = rt.CreateSymlinks(testDir)
			RequireNoError(t, err, "CreateSymlinks %s", path)
		}
	})
}

// TestSplit tests Split function.
func (ts *Suite) TestSplit(t *testing.T, _ string) {
	vfs := ts.vfsTest

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

// TestSplitAbs tests SplitAbs function.
func (ts *Suite) TestSplitAbs(t *testing.T, _ string) {
	vfs := ts.vfsTest

	cases := []struct {
		path   string
		dir    string
		file   string
		osType avfs.OSType
	}{
		{osType: avfs.OsLinux, path: "/", dir: "", file: ""},
		{osType: avfs.OsLinux, path: "/home", dir: "", file: "home"},
		{osType: avfs.OsLinux, path: "/home/user", dir: "/home", file: "user"},
		{osType: avfs.OsLinux, path: "/usr/lib/xorg", dir: "/usr/lib", file: "xorg"},
		{osType: avfs.OsWindows, path: `C:\`, dir: `C:`, file: ""},
		{osType: avfs.OsWindows, path: `C:\Users`, dir: `C:`, file: "Users"},
		{osType: avfs.OsWindows, path: `C:\Users\Default`, dir: `C:\Users`, file: "Default"},
	}

	for _, c := range cases {
		if c.osType != vfs.OSType() {
			continue
		}

		dir, file := avfs.SplitAbs(vfs, c.path)
		if c.dir != dir {
			t.Errorf("splitPath %s : want dir to be %s, got %s", c.path, c.dir, dir)
		}

		if c.file != file {
			t.Errorf("splitPath %s : want file to be %s, got %s", c.path, c.file, file)
		}
	}
}

// TestWalkDir tests WalkDir function.
func (ts *Suite) TestWalkDir(t *testing.T, testDir string) {
	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	symlinks := ts.createSampleSymlinks(t, testDir)

	vfs := ts.vfsTest
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
		RequireNoError(t, err, "WalkDir %s", testDir)

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
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.WalkDir(nonExistingFile, func(path string, info fs.DirEntry, err error) error {
			return nil
		})

		RequireNoError(t, err, "WalkDir %s", nonExistingFile)
	})
}
