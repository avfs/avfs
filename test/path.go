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

package test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/avfs/avfs"
)

type pathTest struct {
	path, result string
}

// TestAbs test Abs function.
func (sfs *SuiteFS) TestAbs(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Abs(testDir)
		if err != nil {
			t.Errorf("Name : want error to be nil, got %v", err)
		}

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
			{sep: avfs.PathSeparator, isAbs: true},
			{sep: '\\'},
			{sep: '.'},
			{sep: 'a'},
		}

		for _, test := range isPathSepTests {
			r := vfs.IsPathSeparator(test.sep)
			if r != test.isAbs {
				t.Errorf("IsPathSeparator(%q) = %v, want %v", test.sep, r, test.isAbs)
			}
		}
	})
}

// TestJoin tests Join function.
func (sfs *SuiteFS) TestJoin(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	type joinTest struct { //nolint:govet // no fieldalignment for simple structs
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

// TestRel tests Rel function.
func (sfs *SuiteFS) TestRel(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	type relTest struct {
		root, path, want string
	}

	var relTests []*relTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		relTests = []*relTest{
			{root: `C:a\b\c`, path: `C:a/b/d`, want: `..\d`},
			{root: `C:\`, path: `D:\`, want: `err`},
			{root: `C:`, path: `D:`, want: `err`},
			{root: `C:\Projects`, path: `c:\projects\src`, want: `src`},
			{root: `C:\Projects`, path: `c:\projects`, want: `.`},
			{root: `C:\Projects\a\..`, path: `c:\projects`, want: `.`},
		}
	default:
		relTests = []*relTest{
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

// TestGlob tests Glob function.
func (sfs *SuiteFS) TestGlob(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Glob("")
		if err != nil {
			t.Errorf("Glob : want error to be nil, got %v", err)
		}

		return
	}

	_ = sfs.SampleDirs(t, testDir)
	_ = sfs.SampleFiles(t, testDir)
	sl := len(sfs.SampleSymlinks(t, testDir))

	vfs = sfs.vfsTest

	t.Run("GlobNormal", func(t *testing.T) {
		pattern := testDir + "/*/*/[A-Z0-9]"
		dirNames, err := vfs.Glob(pattern)
		if err != nil {
			t.Errorf("Glob %s : want error to be nil, got %v", pattern, err)
		} else {
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
		}
	})

	t.Run("GlobWithoutMeta", func(t *testing.T) {
		pattern := testDir
		dirNames, err := vfs.Glob(pattern)
		if err != nil {
			t.Errorf("Glob %s : want error to be nil, got %v", pattern, err)

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
		pattern := testDir + "/NonExisiting"
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

// TestWalk tests Walk function.
func (sfs *SuiteFS) TestWalk(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		walkFunc := func(rootDir string, info os.FileInfo, err error) error { return nil }

		err := vfs.Walk(testDir, walkFunc)
		if err != nil {
			t.Errorf("User : want error to be nil, got %v", err)
		}

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest
	lnames := len(dirs) + len(files) + len(symlinks)
	wantNames := make([]string, 0, lnames)

	wantNames = append(wantNames, testDir)
	for _, dir := range dirs {
		wantNames = append(wantNames, vfs.Join(testDir, dir.Path))
	}

	for _, file := range files {
		wantNames = append(wantNames, vfs.Join(testDir, file.Path))
	}

	if vfs.HasFeature(avfs.FeatSymlink) {
		for _, sl := range symlinks {
			wantNames = append(wantNames, vfs.Join(testDir, sl.NewName))
		}
	}

	sort.Strings(wantNames)

	t.Run("Walk", func(t *testing.T) {
		gotNames := make(map[string]int)
		err := vfs.Walk(testDir, func(path string, info os.FileInfo, err error) error {
			gotNames[path]++

			return nil
		})
		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", testDir, err)
		}

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

		err := vfs.Walk(nonExistingFile, func(path string, info os.FileInfo, err error) error {
			return nil
		})
		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", nonExistingFile, err)
		}
	})
}
