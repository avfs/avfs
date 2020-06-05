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

package fsutil_test

import (
	"strings"
	"testing"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fsutil"
)

type PathTest struct {
	path, result string
}

func TestAbs(t *testing.T) {
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

	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	root, err := fs.TempDir("", "TestAbs")
	if err != nil {
		t.Fatal("TempDir failed: ", err)
	}

	defer fs.RemoveAll(root) //nolint:errcheck

	wd, err := fs.Getwd()
	if err != nil {
		t.Fatal("getwd failed: ", err)
	}

	err = fs.Chdir(root)
	if err != nil {
		t.Fatal("chdir failed: ", err)
	}

	defer fs.Chdir(wd) //nolint:errcheck

	for _, dir := range absTestDirs {
		err = fs.Mkdir(dir, 0777)
		if err != nil {
			t.Fatal("Mkdir failed: ", err)
		}
	}

	err = fs.Chdir(absTestDirs[0])
	if err != nil {
		t.Fatal("chdir failed: ", err)
	}

	for _, path := range absTests {
		path = strings.ReplaceAll(path, "$", root)

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

		if !fsutil.IsAbs(abspath) {
			t.Errorf("Abs(%q)=%q, not an absolute path", path, abspath)
		}

		if fsutil.IsAbs(abspath) && abspath != fsutil.Clean(abspath) {
			t.Errorf("Abs(%q)=%q, isn't clean", path, abspath)
		}
	}
}

// Empty path needs to be special-cased on Windows. See golang.org/issue/24441.
// We test it separately from all other absTests because the empty string is not
// a valid path, so it can't be used with os.Stat.
func TestAbsEmptyString(t *testing.T) {
	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	root, err := fs.TempDir("", "TestAbsEmptyString")
	if err != nil {
		t.Fatal("TempDir failed: ", err)
	}

	defer fs.RemoveAll(root) //nolint:errcheck

	wd, err := fs.Getwd()
	if err != nil {
		t.Fatal("getwd failed: ", err)
	}

	err = fs.Chdir(root)
	if err != nil {
		t.Fatal("chdir failed: ", err)
	}

	defer fs.Chdir(wd) //nolint:errcheck

	info, err := fs.Stat(root)
	if err != nil {
		t.Fatalf("%s: %s", root, err)
	}

	abspath, err := fs.Abs("")
	if err != nil {
		t.Fatalf(`Abs("") error: %v`, err)
	}

	absinfo, err := fs.Stat(abspath)
	if err != nil || !fs.SameFile(absinfo, info) {
		t.Errorf(`Abs("")=%q, not the same file`, abspath)
	}

	if !fsutil.IsAbs(abspath) {
		t.Errorf(`Abs("")=%q, not an absolute path`, abspath)
	}

	if fsutil.IsAbs(abspath) && abspath != fsutil.Clean(abspath) {
		t.Errorf(`Abs("")=%q, isn't clean`, abspath)
	}
}

func TestBase(t *testing.T) {
	basetests := []PathTest{
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

	for _, test := range basetests {
		if s := fsutil.Base(test.path); s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

func TestClean(t *testing.T) {
	cleantests := []PathTest{
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

	for _, test := range cleantests {
		if s := fsutil.Clean(test.path); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}

		if s := fsutil.Clean(test.result); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

func TestDir(t *testing.T) {
	dirtests := []PathTest{
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

	for _, test := range dirtests {
		if s := fsutil.Dir(test.path); s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

// TestIsAbs
func TestIsAbs(t *testing.T) {
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
		if r := fsutil.IsAbs(test.path); r != test.isAbs {
			t.Errorf("IsAbs(%q) = %v, want %v", test.path, r, test.isAbs)
		}
	}
}

// TestJoin
func TestJoin(t *testing.T) {
	jointests := []struct {
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

	for _, test := range jointests {
		if p := fsutil.Join(test.elem...); p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
		}
	}
}

// TestRel
func TestRel(t *testing.T) {
	reltests := []struct {
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

	for _, test := range reltests {
		got, err := fsutil.Rel(test.root, test.path)
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

// TestSplit
func TestSplit(t *testing.T) {
	unixsplittests := []struct {
		path, dir, file string
	}{
		{"a/b", "a/", "b"},
		{"a/b/", "a/b/", ""},
		{"a/", "a/", ""},
		{"a", "", "a"},
		{"/", "/", ""},
	}

	for _, test := range unixsplittests {
		if d, f := fsutil.Split(test.path); d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}
