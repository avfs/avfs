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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
)

// NewSuiteFS creates a new test suite for a file system.
func NewSuiteFS(tb testing.TB, vfsSetup, vfsTest avfs.VFSBase) *Suite {
	if vfsSetup == nil {
		tb.Skip("NewSuiteFS : vfsSetup must not be nil, skipping tests")
	}

	ts := newSuite(tb, vfsSetup, vfsTest, nil)

	vfs := ts.VFSTest()
	tb.Logf("VFS: Type=%s OSType=%s UMask=%03o Idm=%s Features=%s",
		vfs.Type(), vfs.OSType(), vfs.UMask(), vfs.Idm().Type(), vfs.Features())

	return ts
}

// NewSuiteIdm creates a new test suite for an identity manager.
func NewSuiteIdm(tb testing.TB, idm avfs.IdentityMgr) *Suite {
	if idm == nil {
		tb.Skip("NewSuiteIdm : vfsSetup must not be nil, skipping tests")
	}

	ts := newSuite(tb, nil, nil, idm)

	tb.Logf("Idm: Type=%s OSType=%s Features=%s", idm.Type(), idm.OSType(), idm.Features())

	return ts
}

// newSuite creates a new test suite.
func newSuite(tb testing.TB, vfsSetup, vfsTest avfs.VFSBase, idm avfs.IdentityMgr) *Suite {
	if vfsSetup == nil {
		vfsSetup = osfs.New()
	}

	if vfsTest == nil {
		vfsTest = vfsSetup
	}

	if idm == nil {
		idm = vfsSetup.Idm()
	}

	vfs := vfsTest

	if vfs.OSType() != avfs.CurrentOSType() {
		tb.Skipf("NewSuite : Current OSType = %s is different from %s OSType = %s, skipping tests",
			avfs.CurrentOSType(), vfs.Type(), vfs.OSType())
	}

	initUser := vfs.User()
	canTestPerm := vfs.OSType() != avfs.OsWindows && initUser.IsAdmin() &&
		vfs.HasFeature(avfs.FeatIdentityMgr) && !vfs.HasFeature(avfs.FeatReadOnlyIdm)

	ts := &Suite{
		vfsSetup:    vfsSetup,
		vfsTest:     vfsTest,
		idm:         idm,
		initUser:    initUser,
		testDataDir: testDataDir(),
		maxRace:     100,
		canTestPerm: canTestPerm,
	}

	ts.groups = ts.CreateGroups(tb, "")
	ts.users = ts.CreateUsers(tb, "")

	return ts
}

// AssertInvalid asserts that the error is fs.ErrInvalid.
func AssertInvalid(tb testing.TB, err error, msgAndArgs ...any) bool {
	if err != fs.ErrInvalid {
		tb.Helper()
		tb.Errorf("error : want error to be %v, got %v\n%s", fs.ErrInvalid, err, formatArgs(msgAndArgs))

		return false
	}

	return true
}

// AssertNoError asserts that there is no error (err == nil).
func AssertNoError(tb testing.TB, err error, msgAndArgs ...any) bool {
	if err != nil {
		tb.Helper()
		tb.Errorf("error : want error to be nil, got %v\n%s", err, formatArgs(msgAndArgs))

		return false
	}

	return true
}

// AssertPanic checks that function f panics.
func AssertPanic(tb testing.TB, funcName string, f func()) {
	tb.Helper()

	defer func() {
		if r := recover(); r == nil {
			tb.Errorf("%s : want function to panic, not panicking", funcName)
		}
	}()

	f()
}

// changeDir changes the current directory for the tests.
func (ts *Suite) changeDir(tb testing.TB, dir string) {
	vfs := ts.vfsTest

	err := vfs.Chdir(dir)
	RequireNoError(tb, err, "Chdir %s", dir)
}

// closedFile returns a closed avfs.File.
func (ts *Suite) closedFile(tb testing.TB, testDir string) (f avfs.File, fileName string) {
	fileName = ts.emptyFile(tb, testDir)

	vfs := ts.vfsTest

	f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
	RequireNoError(tb, err, "OpenFile %s", fileName)

	err = f.Close()
	RequireNoError(tb, err, "Close %s", fileName)

	return f, fileName
}

// createDir creates a directory for the tests.
func (ts *Suite) createDir(tb testing.TB, dirName string, mode fs.FileMode) {
	vfs := ts.vfsSetup

	err := vfs.MkdirAll(dirName, mode)
	RequireNoError(tb, err, "MkdirAll %s", dirName)

	err = vfs.Chmod(dirName, mode)
	RequireNoError(tb, err, "Chmod %s", dirName)
}

// createFile creates an empty file for the tests.
func (ts *Suite) createFile(tb testing.TB, fileName string, mode fs.FileMode) {
	vfs := ts.vfsSetup

	err := vfs.WriteFile(fileName, nil, mode)
	RequireNoError(tb, err, "WriteFile %s", fileName)
}

// createRootDir creates tests root directory.
func (ts *Suite) createRootDir(tb testing.TB) {
	vfs := ts.vfsSetup
	rootDir := ""

	if _, ok := tb.(*testing.B); ok && vfs.HasFeature(avfs.FeatRealFS) {
		// run Benches on real disks, /tmp is usually an in memory file system.
		rootDir = vfs.Join(avfs.HomeDirUser(vfs, "", vfs.User()), "tmp")
		ts.createDir(tb, rootDir, avfs.DefaultDirPerm)
	}

	rootDir, err := vfs.MkdirTemp(rootDir, "avfs")
	RequireNoError(tb, err, "MkdirTemp %s", rootDir)

	// Make rootDir accessible by anyone.
	err = vfs.Chmod(rootDir, avfs.DefaultDirPerm)
	RequireNoError(tb, err, "Chmod %s", rootDir)

	// Ensure rootDir does not include symbolic links.
	if vfs.HasFeature(avfs.FeatSymlink) {
		rootDir, err = vfs.EvalSymlinks(rootDir)
		RequireNoError(tb, err, "EvalSymlinks %s", rootDir)
	}

	ts.rootDir = rootDir
}

// emptyFile returns an empty file name.
func (ts *Suite) emptyFile(tb testing.TB, testDir string) string {
	const emptyFile = "emptyFile"

	vfs := ts.vfsSetup
	fileName := vfs.Join(testDir, emptyFile)

	_, err := vfs.Stat(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		f, err := vfs.Create(fileName)
		RequireNoError(tb, err, "Create %s", fileName)

		err = f.Close()
		RequireNoError(tb, err, "Close %s", fileName)
	}

	return fileName
}

// existingDir returns an existing directory.
func (ts *Suite) existingDir(tb testing.TB, testDir string) string {
	vfs := ts.vfsSetup

	dirName, err := vfs.MkdirTemp(testDir, "existingDir")
	RequireNoError(tb, err, "MkdirTemp %s", testDir)

	_, err = vfs.Stat(dirName)
	if errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Stat %s : want error to be nil, got %v", dirName, err)
	}

	return dirName
}

// existingFile returns an existing file name with the given content.
func (ts *Suite) existingFile(tb testing.TB, testDir string, content []byte) string {
	vfs := ts.vfsSetup

	f, err := vfs.CreateTemp(testDir, defaultFile)
	RequireNoError(tb, err, "CreateTemp %s", testDir)

	fileName := f.Name()

	_, err = f.Write(content)
	RequireNoError(tb, err, "Write %s", fileName)

	err = f.Close()
	RequireNoError(tb, err, "Close %s", fileName)

	return fileName
}

// funcName returns the name of a function or a method.
// It returns an empty string if not available.
func funcName(i any) string {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Func {
		return ""
	}

	pc := v.Pointer()
	if pc == 0 {
		return ""
	}

	fpc := runtime.FuncForPC(pc)
	if fpc == nil {
		return ""
	}

	fn := fpc.Name()

	end := strings.LastIndex(fn, "-")
	if end == -1 {
		end = len(fn)
	}

	start := strings.LastIndex(fn[:end], ".")
	if start == -1 {
		return fn[:end]
	}

	return fn[start+1 : end]
}

// formatArgs formats a list of optional arguments to a string, the first argument being a format string.
func formatArgs(msgAndArgs []any) string {
	na := len(msgAndArgs)
	if na == 0 {
		return ""
	}

	format, ok := msgAndArgs[0].(string)
	if !ok {
		return ""
	}

	if na == 1 {
		return format
	}

	return fmt.Sprintf(format, msgAndArgs[1:]...)
}

// nonExistingFile returns the name of a non-existing file.
func (ts *Suite) nonExistingFile(tb testing.TB, testDir string) string {
	vfs := ts.vfsSetup
	fileName := vfs.Join(testDir, defaultNonExisting)

	_, err := vfs.Stat(fileName)
	if !errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Stat : want error to be %v, got %v", avfs.ErrNoSuchFileOrDir, err)
	}

	return fileName
}

// openedEmptyFile returns an opened empty avfs.File and its file name.
func (ts *Suite) openedEmptyFile(tb testing.TB, testDir string) (fd avfs.File, fileName string) {
	fileName = ts.emptyFile(tb, testDir)
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
		RequireNoError(tb, err, "OpenFile %s", fileName)

		return f, fileName
	}

	f, err := vfs.OpenFile(fileName, os.O_CREATE|os.O_RDWR, avfs.DefaultFilePerm)
	RequireNoError(tb, err, "OpenFile %s", fileName)

	return f, fileName
}

// openedNonExistingFile returns a non-existing avfs.File and its file name.
func (ts *Suite) openedNonExistingFile(tb testing.TB, testDir string) (f avfs.File) {
	fileName := ts.nonExistingFile(tb, testDir)
	vfs := ts.vfsTest

	f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
	if !errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Open %s : want non existing file, got %v", fileName, err)
	}

	return f
}

// randomDir returns one directory with random empty subdirectories, files and symbolic links.
func (ts *Suite) randomDir(tb testing.TB, testDir string) *avfs.RndTree {
	vfs := ts.vfsSetup

	opts := &avfs.RndTreeOpts{NbDirs: 3, NbFiles: 11, NbSymlinks: 4, MaxFileSize: 0, MaxDepth: 0}
	rt := avfs.NewRndTree(vfs, opts)

	err := rt.CreateTree(testDir)
	RequireNoError(tb, err, "rt.Create %s", testDir)

	return rt
}

// removeDir removes all files under testDir.
func (ts *Suite) removeDir(tb testing.TB, testDir string) {
	vfs := ts.vfsSetup

	err := vfs.Chdir(ts.rootDir)
	RequireNoError(tb, err, "Chdir %s", ts.rootDir)

	// RemoveAll() should be executed as the user who started the tests, generally root,
	// to clean up files with different permissions.
	ts.setInitUser(tb)

	err = vfs.RemoveAll(testDir)
	if err != nil && avfs.CurrentOSType() != avfs.OsWindows {
		tb.Fatalf("RemoveAll %s : want error to be nil, got %v", testDir, err)
	}
}

// RequireNoError require that a function returned no error.
func RequireNoError(tb testing.TB, err error, msgAndArgs ...any) {
	tb.Helper()

	if !AssertNoError(tb, err, msgAndArgs...) {
		tb.FailNow()
	}
}

// RunBenchmarks runs all benchmark functions specified as user userName.
func (ts *Suite) RunBenchmarks(b *testing.B, userName string, BenchFuncs ...func(b *testing.B, testDir string)) {
	vfs := ts.vfsSetup

	ts.createRootDir(b)

	for _, bf := range BenchFuncs {
		ts.setUser(b, userName)

		fn := funcName(bf)
		testDir := vfs.Join(ts.rootDir, fn)

		ts.createDir(b, testDir, avfs.DefaultDirPerm)
		ts.changeDir(b, testDir)

		bf(b, testDir)

		ts.removeDir(b, testDir)
	}

	ts.removeDir(b, ts.rootDir)
}

// RunTests runs all test functions specified as user userName.
func (ts *Suite) RunTests(t *testing.T, userName string, testFuncs ...func(t *testing.T, testDir string)) {
	vfs := ts.vfsSetup

	ts.createRootDir(t)

	defer ts.setInitUser(t)

	for _, tf := range testFuncs {
		ts.setUser(t, userName)

		fn := funcName(tf)
		testDir := vfs.Join(ts.rootDir, fn)

		ts.createDir(t, testDir, avfs.DefaultDirPerm)
		ts.changeDir(t, testDir)

		t.Run(fn, func(t *testing.T) {
			tf(t, testDir)
		})

		ts.removeDir(t, testDir)
	}

	ts.removeDir(t, ts.rootDir)
}

// setUser sets the test user to userName.
func (ts *Suite) setUser(tb testing.TB, userName string) {
	vfs := ts.vfsTest

	u := vfs.User()
	if !ts.canTestPerm || u.Name() == userName {
		return
	}

	err := vfs.SetUserByName(userName)
	RequireNoError(tb, err, "SetUser %s", userName)
}

// setInitUser reset the user to the initial user.
func (ts *Suite) setInitUser(tb testing.TB) {
	ts.setUser(tb, ts.initUser.Name())
}

// testDataDir return the testdata directory of the test package.
func testDataDir() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	return filepath.Join(dir, "testdata")
}

// TestVFSAll runs all file system tests.
func (ts *Suite) TestVFSAll(t *testing.T) {
	ts.TestVFS(t)
	ts.TestFile(t)
	ts.TestUtils(t)
}

// VFSSetup returns the file system used to set up the tests.
func (ts *Suite) VFSSetup() avfs.VFSBase {
	return ts.vfsSetup
}

// VFSTest returns the file system used to run the tests.
func (ts *Suite) VFSTest() avfs.VFSBase {
	return ts.vfsTest
}
