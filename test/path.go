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
	"strings"
	"testing"

	"github.com/avfs/avfs"
)

type pathTest struct {
	path, result string
}

// SuiteAbs test Abs function.
func (cf *ConfigFs) SuiteAbs() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

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

		fs = cf.GetFsRead()

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
func (cf *ConfigFs) SuiteBase() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

	for _, test := range basetests {
		s := fs.Base(test.path)
		if s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

// SuiteClean tests Clean function.
func (cf *ConfigFs) SuiteClean() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

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
func (cf *ConfigFs) SuiteDir() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

	for _, test := range dirtests {
		s := fs.Dir(test.path)
		if s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

// SuiteIsAbsPath tests IsAbs function.
func (cf *ConfigFs) SuiteIsAbsPath() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsRead()

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
func (cf *ConfigFs) SuiteJoin() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

	for _, test := range jointests {
		p := fs.Join(test.elem...)
		if p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
		}
	}
}

// SuiteRel tests Rel function.
func (cf *ConfigFs) SuiteRel() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

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
func (cf *ConfigFs) SuiteSplit() {
	t, _, removeDir := cf.CreateRootDir(UsrTest)
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

	fs := cf.GetFsRead()

	for _, test := range unixsplittests {
		d, f := fs.Split(test.path)
		if d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}
