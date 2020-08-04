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

// SuitePath tests all path related functions.
func (sfs *SuiteFs) SuitePath() {
	sfs.SuiteAbs()
	sfs.SuiteBase()
	sfs.SuiteClean()
	sfs.SuiteDir()
	sfs.SuiteFromToSlash()
	sfs.SuiteGlob()
	sfs.SuiteIsAbsPath()
	sfs.SuiteJoin()
	sfs.SuiteRel()
	sfs.SuiteSplit()
	sfs.SuiteWalk()
}

type pathTest struct {
	path, result string
}

// SuiteAbs test Abs function.
func (sfs *SuiteFs) SuiteAbs() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

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

		wd, err := fs.Getwd()
		if err != nil {
			t.Fatal("getwd failed: ", err)
		}

		err = fs.Chdir(rootDir)
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		defer fs.Chdir(wd) //nolint:errcheck

		for _, dir := range absTestDirs {
			err = fs.Mkdir(dir, 0o777)
			if err != nil {
				t.Fatal("Mkdir failed: ", err)
			}
		}

		err = fs.Chdir(absTestDirs[0])
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		fs = sfs.GetFsRead()

		for _, path := range absTests {
			path = strings.ReplaceAll(path, "$", rootDir)

			info, err := fs.Stat(path)
			if err != nil {
				t.Errorf("%s: %s", path, err)
				continue
			}

			abspath, err := fs.Abs(path)
			if err != nil {
				t.Errorf("Abs(%q) error: %v", path, err)
				continue
			}

			absinfo, err := fs.Stat(abspath)
			if err != nil || !fs.SameFile(absinfo, info) {
				t.Errorf("Abs(%q)=%q, not the same file", path, abspath)
			}

			if !fs.IsAbs(abspath) {
				t.Errorf("Abs(%q)=%q, not an absolute path", path, abspath)
			}

			if fs.IsAbs(abspath) && abspath != fs.Clean(abspath) {
				t.Errorf("Abs(%q)=%q, isn't clean", path, abspath)
			}
		}
	})

	// AbsEmptyString tests Abs functions with an empty string input.
	// Empty path needs to be special-cased on Windows. See golang.org/issue/24441.
	// We test it separately from all other absTests because the empty string is not
	// a valid path, so it can't be used with os.Stat.
	t.Run("AbsEmptyString", func(t *testing.T) {
		wd, err := fs.Getwd()
		if err != nil {
			t.Fatal("getwd failed: ", err)
		}

		err = fs.Chdir(rootDir)
		if err != nil {
			t.Fatal("chdir failed: ", err)
		}

		defer fs.Chdir(wd) //nolint:errcheck

		info, err := fs.Stat(rootDir)
		if err != nil {
			t.Fatalf("%s: %s", rootDir, err)
		}

		abspath, err := fs.Abs("")
		if err != nil {
			t.Fatalf(`Abs("") error: %v`, err)
		}

		absinfo, err := fs.Stat(abspath)
		if err != nil || !fs.SameFile(absinfo, info) {
			t.Errorf(`Abs("")=%q, not the same file`, abspath)
		}

		if !fs.IsAbs(abspath) {
			t.Errorf(`Abs("")=%q, not an absolute path`, abspath)
		}

		if fs.IsAbs(abspath) && abspath != fs.Clean(abspath) {
			t.Errorf(`Abs("")=%q, isn't clean`, abspath)
		}
	})
}

// SuiteBase tests Base function.
func (sfs *SuiteFs) SuiteBase() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	basetests := []*pathTest{
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

	fs := sfs.GetFsRead()

	for _, test := range basetests {
		s := fs.Base(test.path)
		if s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

// SuiteClean tests Clean function.
func (sfs *SuiteFs) SuiteClean() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	cleantests := []*pathTest{
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

	fs := sfs.GetFsRead()

	for _, test := range cleantests {
		s := fs.Clean(test.path)
		if s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}

		s = fs.Clean(test.result)
		if s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

// SuiteDir tests Dir function.
func (sfs *SuiteFs) SuiteDir() {
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

	fs := sfs.GetFsRead()

	for _, test := range dirtests {
		s := fs.Dir(test.path)
		if s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

func (sfs *SuiteFs) SuiteFromToSlash() {
	t := sfs.t

	// TODO : Add test cases for windows.
	pathtests := []*struct {
		path, from, to string
	}{
		{"/a/b/c", "", ""},
		{"C:\\A\\b/c", "", ""},
	}

	fs := sfs.GetFsRead()
	ost := fs.OSType()

	for _, pt := range pathtests {
		var want string

		if ost == avfs.OsWindows {
			want = pt.from
		} else {
			want = pt.path
		}

		got := fs.FromSlash(pt.path)
		if got != want {
			t.Errorf("FromSlash %s, want %s, got %s", pt.path, want, got)
		}

		if ost == avfs.OsWindows {
			want = pt.to
		} else {
			want = pt.path
		}

		got = fs.ToSlash(pt.path)
		if got != want {
			t.Errorf("FromSlash %s, want %s, got %s", pt.path, want, got)
		}
	}
}

// SuiteIsAbsPath tests IsAbs function.
func (sfs *SuiteFs) SuiteIsAbsPath() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsRead()

	t.Run("IsAbs", func(t *testing.T) {
		isabstests := []struct {
			path  string
			isAbs bool
		}{
			{"", false},
			{"/", true},
			{"/usr/bin/gcc", true},
			{"..", false},
			{"/a/../bb", true},
			{".", false},
			{"./", false},
			{"lala", false},
		}

		for _, test := range isabstests {
			r := fs.IsAbs(test.path)
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
			r := fs.IsPathSeparator(test.sep)
			if r != test.isAbs {
				t.Errorf("IsPathSeparator(%q) = %v, want %v", test.sep, r, test.isAbs)
			}
		}
	})
}

// SuiteJoin tests Join function.
func (sfs *SuiteFs) SuiteJoin() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	jointests := []*struct {
		elem []string
		path string
	}{
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

	fs := sfs.GetFsRead()

	for _, test := range jointests {
		p := fs.Join(test.elem...)
		if p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
		}
	}
}

// SuiteRel tests Rel function.
func (sfs *SuiteFs) SuiteRel() {
	t, _, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	reltests := []*struct {
		root, path, want string
	}{
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

	fs := sfs.GetFsRead()

	for _, test := range reltests {
		got, err := fs.Rel(test.root, test.path)
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

// SuiteSplit tests Split function.
func (sfs *SuiteFs) SuiteSplit() {
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

	fs := sfs.GetFsRead()

	for _, test := range unixsplittests {
		d, f := fs.Split(test.path)
		if d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}

// SuiteGlob tests Glob function.
func (sfs *SuiteFs) SuiteGlob() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	_ = CreateDirs(t, fs, rootDir)
	_ = CreateFiles(t, fs, rootDir)
	sl := len(CreateSymlinks(t, fs, rootDir))

	fs = sfs.GetFsRead()

	t.Run("GlobNormal", func(t *testing.T) {
		pattern := rootDir + "/*/*/[A-Z0-9]"
		dirNames, err := fs.Glob(pattern)
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
		dirNames, err := fs.Glob(pattern)
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
		dirNames, err := fs.Glob(pattern)
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
			_, err := fs.Glob(pattern)
			if err != filepath.ErrBadPattern {
				t.Errorf("Glob %s : want error to be %v, got %v", pattern, filepath.ErrBadPattern, err)
			}
		}
	})
}

// SuiteWalk tests Walk function.
func (sfs *SuiteFs) SuiteWalk() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)

	fs = sfs.GetFsRead()
	lnames := len(dirs) + len(files) + len(symlinks)
	wantNames := make([]string, 0, lnames)

	wantNames = append(wantNames, rootDir)
	for _, dir := range dirs {
		wantNames = append(wantNames, fs.Join(rootDir, dir.Path))
	}

	for _, file := range files {
		wantNames = append(wantNames, fs.Join(rootDir, file.Path))
	}

	if fs.HasFeature(avfs.FeatSymlink) {
		for _, sl := range symlinks {
			wantNames = append(wantNames, fs.Join(rootDir, sl.NewName))
		}
	}

	sort.Strings(wantNames)

	t.Run("Walk", func(t *testing.T) {
		gotNames := make(map[string]int)
		err := fs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
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
}
