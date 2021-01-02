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

// Path tests all path related functions.
func (sfs *SuiteFS) Path() {
	sfs.Abs()
	sfs.Base()
	sfs.Clean()
	sfs.Dir()
	sfs.FromToSlash()
	sfs.Glob()
	sfs.IsAbs()
	sfs.Join()
	sfs.Rel()
	sfs.Split()
	sfs.Walk()
}

type pathTest struct {
	path, result string
}

// Abs test Abs function.
func (sfs *SuiteFS) Abs() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

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

		err = vfs.Chdir(rootDir)
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

		vfs = sfs.GetFsRead()

		for _, path := range absTests {
			path = strings.ReplaceAll(path, "$", rootDir)

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

		err = vfs.Chdir(rootDir)
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		defer vfs.Chdir(wd) //nolint:errcheck // Ignore errors.

		info, err := vfs.Stat(rootDir)
		if err != nil {
			t.Fatalf("%s: %s", rootDir, err)
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

// Base tests Base function.
func (sfs *SuiteFS) Base() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

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

// Clean tests Clean function.
func (sfs *SuiteFS) Clean() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

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

// Dir tests Dir function.
func (sfs *SuiteFS) Dir() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	dirtests := []*pathTest{
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

	vfs := sfs.GetFsRead()

	for _, test := range dirtests {
		s := vfs.Dir(test.path)
		if s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

func (sfs *SuiteFS) FromToSlash() {
	t := sfs.t

	// TODO : Add test cases for windows.
	pathtests := []*struct {
		path, from, to string
	}{
		{"/a/b/c", "", ""},
		{"C:\\A\\b/c", "", ""},
	}

	vfs := sfs.GetFsRead()
	ost := vfs.OSType()

	for _, pt := range pathtests {
		var want string

		if ost == avfs.OsWindows {
			want = pt.from
		} else {
			want = pt.path
		}

		got := vfs.FromSlash(pt.path)
		if got != want {
			t.Errorf("FromSlash %s, want %s, got %s", pt.path, want, got)
		}

		if ost == avfs.OsWindows {
			want = pt.to
		} else {
			want = pt.path
		}

		got = vfs.ToSlash(pt.path)
		if got != want {
			t.Errorf("FromSlash %s, want %s, got %s", pt.path, want, got)
		}
	}
}

// IsAbs tests IsAbs function.
func (sfs *SuiteFS) IsAbs() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

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
			{avfs.PathSeparator, true},
			{'\\', false},
			{'.', false},
			{'a', false},
		}

		for _, test := range isPathSepTests {
			r := vfs.IsPathSeparator(test.sep)
			if r != test.isAbs {
				t.Errorf("IsPathSeparator(%q) = %v, want %v", test.sep, r, test.isAbs)
			}
		}
	})
}

// Join tests Join function.
func (sfs *SuiteFS) Join() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

	type joinTest struct {
		elem []string
		path string
	}

	var joinTests []*joinTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		joinTests = []*joinTest{
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
			{[]string{`C:.`, `a`}, `C:a`},
			{[]string{`C:a`, `b`}, `C:a\b`},
			{[]string{`C:a`, `b`, `d`}, `C:a\b\d`},
			{[]string{`\\host\share`, `foo`}, `\\host\share\foo`},
			{[]string{`\\host\share\foo`}, `\\host\share\foo`},
			{[]string{`//host/share`, `foo/bar`}, `\\host\share\foo\bar`},
			{[]string{`\`}, `\`},
			{[]string{`\`, ``}, `\`},
			{[]string{`\`, `a`}, `\a`},
			{[]string{`\\`, `a`}, `\a`},
			{[]string{`\`, `a`, `b`}, `\a\b`},
			{[]string{`\\`, `a`, `b`}, `\a\b`},
			{[]string{`\`, `\\a\b`, `c`}, `\a\b\c`},
			{[]string{`\\a`, `b`, `c`}, `\a\b\c`},
			{[]string{`\\a\`, `b`, `c`}, `\a\b\c`},
		}
	default:
		joinTests = []*joinTest{
			// zero parameters
			{[]string{}, ""},

			// one parameter
			{[]string{""}, ""},
			{[]string{"a"}, "a"},

			// two parameters
			{[]string{"a", "b"}, "a/b"},
			{[]string{"a", ""}, "a"},
			{[]string{"", "b"}, "b"},
			{[]string{"/", "a"}, "/a"},
			{[]string{"/", ""}, "/"},
			{[]string{"a/", "b"}, "a/b"},
			{[]string{"a/", ""}, "a"},
			{[]string{"", ""}, ""},
		}
	}

	for _, test := range joinTests {
		p := vfs.Join(test.elem...)
		if p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
		}
	}
}

// Rel tests Rel function.
func (sfs *SuiteFS) Rel() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

	type relTest struct {
		root, path, want string
	}

	var relTests []*relTest

	switch vfs.OSType() {
	case avfs.OsWindows:
		relTests = []*relTest{
			{`C:a\b\c`, `C:a/b/d`, `..\d`},
			{`C:\`, `D:\`, `err`},
			{`C:`, `D:`, `err`},
			{`C:\Projects`, `c:\projects\src`, `src`},
			{`C:\Projects`, `c:\projects`, `.`},
			{`C:\Projects\a\..`, `c:\projects`, `.`},
		}
	default:
		relTests = []*relTest{
			{"a/b", "a/b", "."},
			{"a/b/.", "a/b", "."},
			{"a/b", "a/b/.", "."},
			{"./a/b", "a/b", "."},
			{"a/b", "./a/b", "."},
			{"ab/cd", "ab/cde", "../cde"},
			{"ab/cd", "ab/c", "../c"},
			{"a/b", "a/b/c/d", "c/d"},
			{"a/b", "a/b/../c", "../c"},
			{"a/b/../c", "a/b", "../b"},
			{"a/b/c", "a/c/d", "../../c/d"},
			{"a/b", "c/d", "../../c/d"},
			{"a/b/c/d", "a/b", "../.."},
			{"a/b/c/d", "a/b/", "../.."},
			{"a/b/c/d/", "a/b", "../.."},
			{"a/b/c/d/", "a/b/", "../.."},
			{"../../a/b", "../../a/b/c/d", "c/d"},
			{"/a/b", "/a/b", "."},
			{"/a/b/.", "/a/b", "."},
			{"/a/b", "/a/b/.", "."},
			{"/ab/cd", "/ab/cde", "../cde"},
			{"/ab/cd", "/ab/c", "../c"},
			{"/a/b", "/a/b/c/d", "c/d"},
			{"/a/b", "/a/b/../c", "../c"},
			{"/a/b/../c", "/a/b", "../b"},
			{"/a/b/c", "/a/c/d", "../../c/d"},
			{"/a/b", "/c/d", "../../c/d"},
			{"/a/b/c/d", "/a/b", "../.."},
			{"/a/b/c/d", "/a/b/", "../.."},
			{"/a/b/c/d/", "/a/b", "../.."},
			{"/a/b/c/d/", "/a/b/", "../.."},
			{"/../../a/b", "/../../a/b/c/d", "c/d"},
			{".", "a/b", "a/b"},
			{".", "..", ".."},

			// can't do purely lexically
			{"..", ".", "err"},
			{"..", "a", "err"},
			{"../..", "..", "err"},
			{"a", "/a", "err"},
			{"/a", "a", "err"},
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

// Split tests Split function.
func (sfs *SuiteFS) Split() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	unixsplittests := []*struct {
		path, dir, file string
	}{
		{"a/b", "a/", "b"},
		{"a/b/", "a/b/", ""},
		{"a/", "a/", ""},
		{"a", "", "a"},
		{"/", "/", ""},
	}

	vfs := sfs.GetFsRead()

	for _, test := range unixsplittests {
		d, f := vfs.Split(test.path)
		if d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}

// Glob tests Glob function.
func (sfs *SuiteFS) Glob() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	_ = CreateDirs(t, vfs, rootDir)
	_ = CreateFiles(t, vfs, rootDir)
	sl := len(CreateSymlinks(t, vfs, rootDir))

	vfs = sfs.GetFsRead()

	t.Run("GlobNormal", func(t *testing.T) {
		pattern := rootDir + "/*/*/[A-Z0-9]"
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
		pattern := rootDir
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
		pattern := rootDir + "/NonExisiting"
		dirNames, err := vfs.Glob(pattern)
		if dirNames != nil || err != nil {
			t.Errorf("Glob %s : want error and result to be nil, got %s, %v", pattern, dirNames, err)
		}
	})

	t.Run("GlobError", func(t *testing.T) {
		patterns := []string{
			"[]",
			rootDir + "/[A-Z",
		}

		for _, pattern := range patterns {
			_, err := vfs.Glob(pattern)
			if err != filepath.ErrBadPattern {
				t.Errorf("Glob %s : want error to be %v, got %v", pattern, filepath.ErrBadPattern, err)
			}
		}
	})
}

// Walk tests Walk function.
func (sfs *SuiteFS) Walk() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)

	vfs = sfs.GetFsRead()
	lnames := len(dirs) + len(files) + len(symlinks)
	wantNames := make([]string, 0, lnames)

	wantNames = append(wantNames, rootDir)
	for _, dir := range dirs {
		wantNames = append(wantNames, vfs.Join(rootDir, dir.Path))
	}

	for _, file := range files {
		wantNames = append(wantNames, vfs.Join(rootDir, file.Path))
	}

	if vfs.HasFeature(avfs.FeatSymlink) {
		for _, sl := range symlinks {
			wantNames = append(wantNames, vfs.Join(rootDir, sl.NewName))
		}
	}

	sort.Strings(wantNames)

	t.Run("Walk", func(t *testing.T) {
		gotNames := make(map[string]int)
		err := vfs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			gotNames[path]++

			return nil
		})
		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", rootDir, err)
		}

		if len(wantNames) != len(gotNames) {
			t.Errorf("Walk %s : want %d files or dirs, got %d", rootDir, len(wantNames), len(gotNames))
		}

		for _, wantName := range wantNames {
			n, ok := gotNames[wantName]
			if !ok || n != 1 {
				t.Errorf("Walk %s : path %s not found", rootDir, wantName)
			}
		}
	})

	t.Run("WalkNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		err := vfs.Walk(nonExistingFile, func(path string, info os.FileInfo, err error) error {
			return nil
		})
		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", nonExistingFile, err)
		}
	})
}
