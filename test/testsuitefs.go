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
)

const (
	defaultDir         = "defaultDir"
	defaultFile        = "defaultFile"
	defaultNonExisting = "defaultNonExisting"
)

// SuiteFS is a test suite for virtual file systems.
type SuiteFS struct {
	vfsSetup    avfs.VFSBase       // vfsSetup is the file system used to set up the tests (generally with read/write access).
	vfsTest     avfs.VFSBase       // vfsTest is the file system used to run the tests.
	initUser    avfs.UserReader    // initUser is the initial user running the test suite.
	groups      []avfs.GroupReader // groups contains the test groups created with the identity manager.
	users       []avfs.UserReader  // users contains the test users created with the identity manager.
	initDir     string             // initDir is the initial directory of the tests.
	rootDir     string             // rootDir is the root directory for tests and benchmarks.
	maxRace     int                // maxRace is the maximum number of concurrent goroutines used in race tests.
	canTestPerm bool               // canTestPerm indicates if permissions can be tested.
}

// Option defines the option function used for initializing SuiteFS.
type Option func(*SuiteFS)

// NewSuiteFS creates a new test suite for a file system.
func NewSuiteFS(tb testing.TB, vfsSetup avfs.VFSBase, opts ...Option) *SuiteFS {
	if vfsSetup == nil {
		tb.Fatal("New : want vfsSetup to be set, got nil")
	}

	vfs := vfsSetup

	if vfs.OSType() != avfs.CurrentOSType() {
		tb.Skipf("New : Current OSType = %s is different from %s OSType = %s, skipping tests",
			avfs.CurrentOSType(), vfs.Type(), vfs.OSType())
	}

	initUser := vfs.User()
	_, file, _, _ := runtime.Caller(0)
	initDir := filepath.Dir(file)

	canTestPerm := vfs.OSType() != avfs.OsWindows && initUser.IsAdmin() &&
		vfs.HasFeature(avfs.FeatIdentityMgr) && !vfs.HasFeature(avfs.FeatReadOnlyIdm)

	sfs := &SuiteFS{
		vfsSetup:    vfs,
		vfsTest:     vfs,
		initDir:     initDir,
		initUser:    initUser,
		maxRace:     100,
		canTestPerm: canTestPerm,
	}

	defer func() {
		vfs = sfs.vfsTest
		tb.Logf("VFS: Type=%s OSType=%s UMask=%03o Features=%s",
			vfs.Type(), vfs.OSType(), vfs.UMask(), vfs.Features())
	}()

	for _, opt := range opts {
		opt(sfs)
	}

	idm := vfs.Idm()
	sfs.groups = CreateGroups(tb, idm, "")
	sfs.users = CreateUsers(tb, idm, "")

	return sfs
}

// Options

// WithVFSTest returns an option function which sets the VFS used for running tests.
func WithVFSTest(vfsTest avfs.VFS) Option {
	return func(sfs *SuiteFS) {
		sfs.vfsTest = vfsTest
		if vfsTest.HasFeature(avfs.FeatReadOnly) {
			sfs.canTestPerm = false
		}
	}
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

// BenchAll runs all benchmarks.
func (sfs *SuiteFS) BenchAll(b *testing.B) {
	sfs.RunBenchmarks(b, UsrTest,
		sfs.BenchCreate,
		sfs.BenchFileRead,
		sfs.BenchFileWrite,
		sfs.BenchMkdir,
		sfs.BenchOpenFile,
		sfs.BenchRemove,
	)
}

// changeDir changes the current directory for the tests.
func (sfs *SuiteFS) changeDir(tb testing.TB, dir string) {
	vfs := sfs.vfsTest

	err := vfs.Chdir(dir)
	RequireNoError(tb, err, "Chdir %s", dir)
}

// closedFile returns a closed avfs.File.
func (sfs *SuiteFS) closedFile(tb testing.TB, testDir string) (f avfs.File, fileName string) {
	fileName = sfs.emptyFile(tb, testDir)

	vfs := sfs.vfsTest

	f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
	RequireNoError(tb, err, "OpenFile %s", fileName)

	err = f.Close()
	RequireNoError(tb, err, "Close %s", fileName)

	return f, fileName
}

// createDir creates a directory for the tests.
func (sfs *SuiteFS) createDir(tb testing.TB, dirName string, mode fs.FileMode) {
	vfs := sfs.vfsSetup

	err := vfs.MkdirAll(dirName, mode)
	RequireNoError(tb, err, "MkdirAll %s", dirName)

	err = vfs.Chmod(dirName, mode)
	RequireNoError(tb, err, "Chmod %s", dirName)
}

// createFile creates an empty file for the tests.
func (sfs *SuiteFS) createFile(tb testing.TB, fileName string, mode fs.FileMode) {
	vfs := sfs.vfsSetup

	err := vfs.WriteFile(fileName, nil, mode)
	RequireNoError(tb, err, "WriteFile %s", fileName)
}

// createRootDir creates tests root directory.
func (sfs *SuiteFS) createRootDir(tb testing.TB) {
	vfs := sfs.vfsSetup
	rootDir := ""

	if _, ok := tb.(*testing.B); ok && vfs.HasFeature(avfs.FeatRealFS) {
		// run Benches on real disks, /tmp is usually an in memory file system.
		rootDir = vfs.Join(vfs.HomeDirUser(vfs.User()), "tmp")
		sfs.createDir(tb, rootDir, avfs.DefaultDirPerm)
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

	sfs.rootDir = rootDir
}

// emptyFile returns an empty file name.
func (sfs *SuiteFS) emptyFile(tb testing.TB, testDir string) string {
	const emptyFile = "emptyFile"

	vfs := sfs.vfsSetup
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
func (sfs *SuiteFS) existingDir(tb testing.TB, testDir string) string {
	vfs := sfs.vfsSetup

	dirName, err := vfs.MkdirTemp(testDir, "existingDir")
	RequireNoError(tb, err, "MkdirTemp %s", testDir)

	_, err = vfs.Stat(dirName)
	if errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Stat %s : want error to be nil, got %v", dirName, err)
	}

	return dirName
}

// existingFile returns an existing file name with the given content.
func (sfs *SuiteFS) existingFile(tb testing.TB, testDir string, content []byte) string {
	vfs := sfs.vfsSetup

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
func (sfs *SuiteFS) nonExistingFile(tb testing.TB, testDir string) string {
	vfs := sfs.vfsSetup
	fileName := vfs.Join(testDir, defaultNonExisting)

	_, err := vfs.Stat(fileName)
	if !errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Stat : want error to be %v, got %v", avfs.ErrNoSuchFileOrDir, err)
	}

	return fileName
}

// openedEmptyFile returns an opened empty avfs.File and its file name.
func (sfs *SuiteFS) openedEmptyFile(tb testing.TB, testDir string) (fd avfs.File, fileName string) {
	fileName = sfs.emptyFile(tb, testDir)
	vfs := sfs.vfsTest

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
func (sfs *SuiteFS) openedNonExistingFile(tb testing.TB, testDir string) (f avfs.File) {
	fileName := sfs.nonExistingFile(tb, testDir)
	vfs := sfs.vfsTest

	f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
	if !errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("Open %s : want non existing file, got %v", fileName, err)
	}

	return f
}

// randomDir returns one directory with random empty subdirectories, files and symbolic links.
func (sfs *SuiteFS) randomDir(tb testing.TB, testDir string) *avfs.RndTree {
	vfs := sfs.vfsSetup
	RndParamsOneDir := avfs.RndTreeParams{
		MinName:     4,
		MaxName:     32,
		MinDirs:     10,
		MaxDirs:     20,
		MinFiles:    50,
		MaxFiles:    100,
		MinFileSize: 0,
		MaxFileSize: 0,
		MinSymlinks: 5,
		MaxSymlinks: 10,
		OneLevel:    true,
	}

	rt, err := avfs.NewRndTree(vfs, testDir, &RndParamsOneDir)
	RequireNoError(tb, err, "NewRndTree %s", testDir)

	err = rt.CreateTree()
	RequireNoError(tb, err, "rt.Create %s", testDir)

	return rt
}

// removeDir removes all files under testDir.
func (sfs *SuiteFS) removeDir(tb testing.TB, testDir string) {
	vfs := sfs.vfsSetup

	err := vfs.Chdir(sfs.rootDir)
	RequireNoError(tb, err, "Chdir %s", sfs.rootDir)

	// RemoveAll() should be executed as the user who started the tests, generally root,
	// to clean up files with different permissions.
	sfs.setUser(tb, sfs.initUser.Name())

	err = vfs.RemoveAll(testDir)
	if err != nil && avfs.CurrentOSType() != avfs.OsWindows {
		tb.Fatalf("RemoveAll %s : want error to be nil, got %v", testDir, err)
	}
}

// RequireNoError require that a function returned no error.
func RequireNoError(tb testing.TB, err error, msgAndArgs ...any) {
	if !AssertNoError(tb, err, msgAndArgs...) {
		tb.FailNow()
	}
}

// RunBenchmarks runs all benchmark functions specified as user userName.
func (sfs *SuiteFS) RunBenchmarks(b *testing.B, userName string, BenchFuncs ...func(b *testing.B, testDir string)) {
	vfs := sfs.vfsSetup

	sfs.createRootDir(b)

	for _, bf := range BenchFuncs {
		sfs.setUser(b, userName)

		fn := funcName(bf)
		testDir := vfs.Join(sfs.rootDir, fn)

		sfs.createDir(b, testDir, avfs.DefaultDirPerm)
		sfs.changeDir(b, testDir)

		bf(b, testDir)

		sfs.removeDir(b, testDir)
	}

	sfs.removeDir(b, sfs.rootDir)
}

// RunTests runs all test functions specified as user userName.
func (sfs *SuiteFS) RunTests(t *testing.T, userName string, testFuncs ...func(t *testing.T, testDir string)) {
	vfs := sfs.vfsSetup

	sfs.createRootDir(t)

	for _, tf := range testFuncs {
		sfs.setUser(t, userName)

		fn := funcName(tf)
		testDir := vfs.Join(sfs.rootDir, fn)

		sfs.createDir(t, testDir, avfs.DefaultDirPerm)
		sfs.changeDir(t, testDir)

		t.Run(fn, func(t *testing.T) {
			tf(t, testDir)
		})

		sfs.removeDir(t, testDir)
	}

	sfs.removeDir(t, sfs.rootDir)
}

// setUser sets the test user to userName.
func (sfs *SuiteFS) setUser(tb testing.TB, userName string) {
	vfs := sfs.vfsSetup

	u := vfs.User()
	if !sfs.canTestPerm || u.Name() == userName {
		return
	}

	_, err := vfs.SetUser(userName)
	RequireNoError(tb, err, "SetUser %s", userName)
}

// TestAll runs all tests.
func (sfs *SuiteFS) TestAll(t *testing.T) {
	sfs.RunTests(t, UsrTest,
		// VFS tests
		sfs.TestAbs,
		sfs.TestBase,
		sfs.TestClean,
		sfs.TestDir,
		sfs.TestClone,
		sfs.TestChdir,
		sfs.TestChtimes,
		sfs.TestCreate,
		sfs.TestCreateTemp,
		sfs.TestEvalSymlink,
		sfs.TestFromToSlash,
		sfs.TestGlob,
		sfs.TestIsAbs,
		sfs.TestJoin,
		sfs.TestLink,
		sfs.TestLstat,
		sfs.TestMatch,
		sfs.TestMkdir,
		sfs.TestMkdirTemp,
		sfs.TestMkdirAll,
		sfs.TestName,
		sfs.TestOpen,
		sfs.TestOpenFileWrite,
		sfs.TestPathSeparator,
		sfs.TestReadDir,
		sfs.TestReadFile,
		sfs.TestReadlink,
		sfs.TestRel,
		sfs.TestRemove,
		sfs.TestRemoveAll,
		sfs.TestRename,
		sfs.TestSameFile,
		sfs.TestSplit,
		sfs.TestSplitAbs,
		sfs.TestStat,
		sfs.TestSymlink,
		sfs.TestTempDir,
		sfs.TestToSysStat,
		sfs.TestTruncate,
		sfs.TestUser,
		sfs.TestWalkDir,
		sfs.TestWriteFile,
		sfs.TestWriteString,

		// File tests
		sfs.TestFileChdir,
		sfs.TestFileCloseWrite,
		sfs.TestFileCloseRead,
		sfs.TestFileFd,
		sfs.TestFileName,
		sfs.TestFileRead,
		sfs.TestFileReadAt,
		sfs.TestFileReadDir,
		sfs.TestFileReaddirnames,
		sfs.TestFileSeek,
		sfs.TestFileStat,
		sfs.TestFileSync,
		sfs.TestFileTruncate,
		sfs.TestFileWrite,
		sfs.TestFileWriteAt,
		sfs.TestFileWriteString,
		sfs.TestFileWriteTime,

		// other functions
		sfs.TestCopyFile,
		sfs.TestDirExists,
		sfs.TestExists,
		sfs.TestHashFile,
		sfs.TestIsDir,
		sfs.TestIsEmpty,
		sfs.TestIsPathSeparator,
		sfs.TestRndTree,
		sfs.TestUMask,
	)

	// Tests to be run as root
	adminUser := sfs.vfsSetup.Idm().AdminUser()
	sfs.RunTests(t, adminUser.Name(),
		sfs.TestChmod,
		sfs.TestChown,
		sfs.TestChroot,
		sfs.TestCreateSystemDirs,
		sfs.TestCreateHomeDir,
		sfs.TestLchown,
		sfs.TestFileChmod,
		sfs.TestFileChown,
		sfs.TestVolume,
		sfs.TestWriteOnReadOnlyFS,
	)
}

// VFSSetup returns the file system used to set up the tests.
func (sfs *SuiteFS) VFSSetup() avfs.VFSBase {
	return sfs.vfsSetup
}

// VFSTest returns the file system used to run the tests.
func (sfs *SuiteFS) VFSTest() avfs.VFSBase {
	return sfs.vfsTest
}
