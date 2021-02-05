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

// +build !datarace

package vfsutils_test

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfsutils"
)

func InitTest(t *testing.T) avfs.VFS {
	vfsWrite, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfsWrite)
	vfs := sfs.VFSRead()

	return vfs
}

func TestIOUtil(t *testing.T) {
	vfsWrite, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfsWrite)
	sfs.TestReadDir(t)
	sfs.TestReadFile(t)
	sfs.TestWriteFile(t)
}

func TestReadOnlyWriteFile(t *testing.T) {
	vfs := InitTest(t)

	// We don't want to use TempFile directly, since that opens a file for us as 0600.
	tempDir, err := vfs.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("TempDir %s: %v", t.Name(), err)
	}

	defer vfs.RemoveAll(tempDir) //nolint:errcheck // Ignore errors.

	filename := vfsutils.Join(tempDir, "blurp.txt")

	shmorp := []byte("shmorp")
	florp := []byte("florp")

	err = vfs.WriteFile(filename, shmorp, 0o444)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", filename, err)
	}

	err = vfs.WriteFile(filename, florp, 0o444)
	if err == nil {
		t.Fatalf("Expected an error when writing to read-only file %s", filename)
	}

	got, err := vfs.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", filename, err)
	}

	if !bytes.Equal(got, shmorp) {
		t.Fatalf("want %s, got %s", shmorp, got)
	}
}

func TestTempFile(t *testing.T) {
	vfs := InitTest(t)

	dir, err := vfs.TempDir("", "TestTempFile_BadDir")
	if err != nil {
		t.Fatal(err)
	}

	defer vfs.RemoveAll(dir) //nolint:errcheck // Ignore errors.

	nonexistentDir := vfsutils.Join(dir, "_not_exists_")

	f, err := vfs.TempFile(nonexistentDir, "foo")
	if f != nil || err == nil {
		t.Errorf("TempFile(%q, `foo`) = %v, %v", nonexistentDir, f, err)
	}
}

func TestTempFile_pattern(t *testing.T) {
	tests := []struct{ pattern, prefix, suffix string }{
		{"ioutil_test", "ioutil_test", ""},
		{"ioutil_test*", "ioutil_test", ""},
		{"ioutil_test*xyz", "ioutil_test", "xyz"},
	}

	vfs := InitTest(t)

	for _, tt := range tests {
		f, err := vfs.TempFile("", tt.pattern)
		if err != nil {
			t.Errorf("TempFile(..., %q) error: %v", tt.pattern, err)

			continue
		}

		defer vfs.Remove(f.Name()) //nolint:errcheck // Ignore errors.

		base := vfsutils.Base(f.Name())
		_ = f.Close()

		if !(strings.HasPrefix(base, tt.prefix) && strings.HasSuffix(base, tt.suffix)) {
			t.Errorf("TempFile pattern %q created bad name %q; want prefix %q & suffix %q",
				tt.pattern, base, tt.prefix, tt.suffix)
		}
	}
}

func TestTempDir(t *testing.T) {
	vfs := InitTest(t)

	name, err := vfs.TempDir("/_not_exists_", "foo")
	if name != "" || err == nil {
		t.Errorf("TempDir(`/_not_exists_`, `foo`) = %v, %v", name, err)
	}

	tests := []struct {
		pattern                string
		wantPrefix, wantSuffix string
	}{
		{"ioutil_test", "ioutil_test", ""},
		{"ioutil_test*", "ioutil_test", ""},
		{"ioutil_test*xyz", "ioutil_test", "xyz"},
	}

	dir := vfs.GetTempDir()

	runTestTempDir := func(t *testing.T, pattern, wantRePat string) {
		name, err := vfs.TempDir(dir, pattern)
		if name == "" || err != nil {
			t.Fatalf("TempDir(dir, `ioutil_test`) = %v, %v", name, err)
		}

		defer vfs.Remove(name) //nolint:errcheck // Ignore errors.

		re := regexp.MustCompile(wantRePat)
		if !re.MatchString(name) {
			t.Errorf("TempDir(%q, %q) created bad name\n\t%q\ndid not match pattern\n\t%q",
				dir, pattern, name, wantRePat)
		}
	}

	for _, tt := range tests {
		prefix, suffix, pattern := tt.wantPrefix, tt.wantSuffix, tt.pattern

		t.Run(tt.pattern, func(t *testing.T) {
			wantRePat := "^" + regexp.QuoteMeta(vfsutils.Join(dir, prefix)) + "[0-9]+" + regexp.QuoteMeta(suffix) + "$"
			runTestTempDir(t, pattern, wantRePat)
		})
	}

	// Separately testing "*xyz" (which has no prefix). That is when constructing the
	// pattern to assert on, as in the previous loop, using filepath.Join for an empty
	// prefix filepath.Join(dir, ""), produces the pattern:
	//     ^<DIR>[0-9]+xyz$
	// yet we just want to match
	//     "^<DIR>/[0-9]+xyz"
	t.Run("*xyz", func(t *testing.T) {
		wantRePat := "^" + regexp.QuoteMeta(vfsutils.Join(dir)) +
			regexp.QuoteMeta(string(avfs.PathSeparator)) + "[0-9]+xyz$"
		runTestTempDir(t, "*xyz", wantRePat)
	})
}

// TestTempDir_BadDir tests that we return a nice error message if the dir argument to TempDir doesn't
// exist (or that it's empty and os.TempDir doesn't exist).
func TestTempDir_BadDir(t *testing.T) {
	vfs := InitTest(t)

	dir, err := vfs.TempDir("", "TestTempDir_BadDir")
	if err != nil {
		t.Fatal(err)
	}

	defer vfs.RemoveAll(dir) //nolint:errcheck // Ignore errors.

	badDir := vfsutils.Join(dir, "not-exist")

	_, err = vfs.TempDir(badDir, "foo")
	if pe, ok := err.(*os.PathError); !ok || !vfs.IsNotExist(err) || pe.Path != badDir {
		t.Errorf("TempDir error = %#v; want PathError for path %q satisifying os.IsNotExist", err, badDir)
	}
}
