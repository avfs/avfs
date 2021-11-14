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
	vfsSetup    avfs.VFS           // vfsSetup is the file system used to setup the tests (generally with read/write access).
	vfsTest     avfs.VFS           // vfsTest is the file system used to run the tests.
	initDir     string             // initDir is the initial directory of the tests.
	initUser    avfs.UserReader    // initUser is the initial user running the test suite.
	rootDir     string             // rootDir is the root directory for tests.
	Groups      []avfs.GroupReader // Groups contains the test groups created with the identity manager.
	Users       []avfs.UserReader  // Users contains the test users created with the identity manager.
	maxRace     int                // maxRace is the maximum number of concurrent goroutines used in race tests.
	canTestPerm bool               // canTestPerm indicates if permissions can be tested.
}

// Option defines the option function used for initializing SuiteFS.
type Option func(*SuiteFS)

// NewSuiteFS creates a new test suite for a file system.
func NewSuiteFS(tb testing.TB, vfsSetup avfs.VFS, opts ...Option) *SuiteFS {
	if vfsSetup == nil {
		tb.Fatal("New : want vfsSetup to be set, got nil")
	}

	vfs := vfsSetup

	osType := avfs.Cfg.OSType()
	if osType != vfs.OSType() {
		tb.Skipf("New : Current OSType = %s is different from %s OSType = %s, skipping tests",
			osType, vfs.Type(), vfs.OSType())
	}

	initUser := vfs.User()
	_, file, _, _ := runtime.Caller(0)
	initDir := filepath.Dir(file)

	canTestPerm := vfs.OSType() != avfs.OsWindows &&
		vfs.HasFeature(avfs.FeatBasicFs) &&
		vfs.HasFeature(avfs.FeatIdentityMgr) &&
		initUser.IsAdmin()

	sfs := &SuiteFS{
		vfsSetup:    vfs,
		vfsTest:     vfs,
		initDir:     initDir,
		initUser:    initUser,
		rootDir:     vfs.TempDir(),
		maxRace:     100,
		canTestPerm: canTestPerm,
	}

	defer func() {
		vfs = sfs.vfsTest
		tb.Logf("Info vfs : type=%s, OSType=%s, Features=%s", vfs.Type(), vfs.OSType(), vfs.Features())
	}()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return sfs
	}

	for _, opt := range opts {
		opt(sfs)
	}

	rootDir, err := vfs.MkdirTemp("", "avfs")
	if err != nil {
		tb.Fatalf("MkdirTemp %s : want error to be nil, got %s", rootDir, err)
	}

	// Default permissions on rootDir are 0o700 (generated by MkdirTemp()).
	// Make rootDir accessible by anyone.
	err = vfs.Chmod(rootDir, 0o777)
	if err != nil {
		tb.Fatalf("Chmod %s : want error to be nil, got %s", rootDir, err)
	}

	// Ensure rootDir does not include symbolic links.
	if vfs.HasFeature(avfs.FeatSymlink) {
		rootDir, err = vfs.EvalSymlinks(rootDir)
		if err != nil {
			tb.Fatalf("EvalSymlinks %s : want error to be nil, got %s", rootDir, err)
		}
	}

	sfs.rootDir = rootDir

	if canTestPerm {
		idm := vfsSetup.Idm()

		sfs.Groups = CreateGroups(tb, idm, "")
		sfs.Users = CreateUsers(tb, idm, "")
	}

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

// SetUser sets the test user to userName.
func (sfs *SuiteFS) SetUser(tb testing.TB, userName string) avfs.UserReader {
	vfs := sfs.vfsSetup

	u := vfs.User()
	if !sfs.canTestPerm || u.Name() == userName {
		return u
	}

	u, err := vfs.SetUser(userName)
	if err != nil {
		tb.Fatalf("SetUser %s : want error to be nil, got %s", userName, err)
	}

	return u
}

// VFSTest returns the file system used to run the tests.
func (sfs *SuiteFS) VFSTest() avfs.VFS {
	return sfs.vfsTest
}

// VFSSetup returns the file system used to setup the tests.
func (sfs *SuiteFS) VFSSetup() avfs.VFS {
	return sfs.vfsSetup
}

// SuiteTestFunc is a test function to be used by RunTest.
type SuiteTestFunc func(t *testing.T, testDir string)

// RunTests runs all test functions testFuncs specified as user userName.
func (sfs *SuiteFS) RunTests(t *testing.T, userName string, testFuncs ...SuiteTestFunc) {
	vfs := sfs.vfsSetup

	defer func() {
		sfs.SetUser(t, sfs.initUser.Name())

		err := os.Chdir(sfs.initDir)
		if err != nil {
			t.Fatalf("Chdir %s : want error to be nil, got %v", sfs.initDir, err)
		}
	}()

	for _, tf := range testFuncs {
		sfs.SetUser(t, userName)

		fn := funcName(tf)
		testDir := vfs.Join(sfs.rootDir, fn)

		sfs.CreateDir(t, testDir, 0o777)
		sfs.ChangeDir(t, testDir)

		t.Run(fn, func(t *testing.T) {
			tf(t, testDir)
		})

		sfs.RemoveTestDir(t, testDir)
	}
}

// SuiteBenchFunc is a bench function to be used by RunBenches.
type SuiteBenchFunc func(b *testing.B, testDir string)

// RunBenches runs all benchmark functions benchFuncs specified as user userName.
func (sfs *SuiteFS) RunBenches(b *testing.B, userName string, benchFuncs ...SuiteBenchFunc) {
	vfs := sfs.vfsSetup

	defer func() {
		sfs.SetUser(b, sfs.initUser.Name())

		err := os.Chdir(sfs.initDir)
		if err != nil {
			b.Fatalf("Chdir %s : want error to be nil, got %v", sfs.initDir, err)
		}
	}()

	for _, bf := range benchFuncs {
		sfs.SetUser(b, userName)

		fn := funcName(bf)
		testDir := vfs.Join(sfs.rootDir, fn)

		sfs.CreateDir(b, testDir, 0o777)
		sfs.ChangeDir(b, testDir)

		b.Run(fn, func(b *testing.B) {
			bf(b, testDir)
		})

		sfs.RemoveTestDir(b, testDir)
	}
}

// funcName returns the name of a function or a method.
// It returns an empty string if not available.
func funcName(i interface{}) string {
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

// CreateDir creates a directory for the tests.
func (sfs *SuiteFS) CreateDir(tb testing.TB, dir string, mode fs.FileMode) {
	tb.Helper()

	vfs := sfs.vfsSetup
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := vfs.MkdirAll(dir, mode)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %s", dir, err)
	}

	err = vfs.Chmod(dir, mode)
	if err != nil {
		tb.Fatalf("Chmod %s : want error to be nil, got %s", dir, err)
	}
}

// ChangeDir changes the current directory for the tests.
func (sfs *SuiteFS) ChangeDir(tb testing.TB, dir string) {
	tb.Helper()

	vfs := sfs.vfsTest
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := vfs.Chdir(dir)
	if err != nil {
		tb.Fatalf("Chdir %s : want error to be nil, got %s", dir, err)
	}
}

// RemoveTestDir removes all files under testDir.
func (sfs *SuiteFS) RemoveTestDir(tb testing.TB, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := vfs.Chdir(sfs.rootDir)
	if err != nil {
		tb.Fatalf("Chdir %s : want error to be nil, got %v", sfs.rootDir, err)
	}

	err = vfs.RemoveAll(testDir)
	if err == nil || !sfs.canTestPerm {
		return
	}

	// Cleanup permissions for RemoveAll()
	// as the user who started the tests.
	sfs.SetUser(tb, sfs.initUser.Name())

	err = vfs.WalkDir(testDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		return vfs.Chmod(path, 0o777)
	})

	if err != nil {
		tb.Fatalf("Walk %s : want error to be nil, got %v", testDir, err)
	}

	err = vfs.RemoveAll(testDir)
	if err != nil {
		tb.Fatalf("RemoveAll %s : want error to be nil, got %v", testDir, err)
	}
}

// TestAll runs all tests.
func (sfs *SuiteFS) TestAll(t *testing.T) {
	sfs.TestVFS(t)
	sfs.TestVFSUtils(t)
}

// TestVFS runs all file system tests.
func (sfs *SuiteFS) TestVFS(t *testing.T) {
	adminUser := sfs.vfsSetup.Idm().AdminUser()

	sfs.RunTests(t, UsrTest,
		// VFS tests
		sfs.TestClone,
		sfs.TestChdir,
		sfs.TestChtimes,
		sfs.TestCreate,
		sfs.TestEvalSymlink,
		sfs.TestLink,
		sfs.TestLstat,
		sfs.TestMkdir,
		sfs.TestMkdirAll,
		sfs.TestName,
		sfs.TestOpen,
		sfs.TestOpenFileWrite,
		sfs.TestReadlink,
		sfs.TestRemove,
		sfs.TestRemoveAll,
		sfs.TestRename,
		sfs.TestSameFile,
		sfs.TestStat,
		sfs.TestSymlink,
		sfs.TestTempDir,
		sfs.TestToSysStat,
		sfs.TestTruncate,
		sfs.TestUser,
		sfs.TestWriteString,

		// File tests
		sfs.TestFileChdir,
		sfs.TestFileCloseWrite,
		sfs.TestFileCloseRead,
		sfs.TestFileFd,
		sfs.TestFileName,
		sfs.TestFileRead,
		sfs.TestFileReadDir,
		sfs.TestFileReaddirnames,
		sfs.TestFileSeek,
		sfs.TestFileStat,
		sfs.TestFileSync,
		sfs.TestFileTruncate,
		sfs.TestFileWrite,
		sfs.TestFileWriteString,
		sfs.TestFileWriteTime)

	// Tests to be run as root
	sfs.RunTests(t, adminUser.Name(),
		sfs.TestChmod,
		sfs.TestChown,
		sfs.TestChroot,
		sfs.TestLchown,
		sfs.TestFileChmod,
		sfs.TestFileChown,
		sfs.TestSetUser)
}

// TestVFSUtils runs vfsutils package tests.
func (sfs *SuiteFS) TestVFSUtils(t *testing.T) {
	sfs.RunTests(t, UsrTest,
		// Path tests
		sfs.TestAbs,
		sfs.TestBase,
		sfs.TestClean,
		sfs.TestDir,
		sfs.TestFromToSlash,
		sfs.TestGlob,
		sfs.TestIsAbs,
		sfs.TestJoin,
		sfs.TestMatch,
		sfs.TestRel,
		sfs.TestSplit,
		sfs.TestWalkDir,

		// other functions
		sfs.TestCopyFile,
		sfs.TestCreateBaseDirs,
		sfs.TestCreateTemp,
		sfs.TestDirExists,
		sfs.TestExists,
		sfs.TestIsDir,
		sfs.TestIsEmpty,
		sfs.TestHashFile,
		sfs.TestMkdirTemp,
		sfs.TestPathIterator,
		sfs.TestReadDir,
		sfs.TestReadFile,
		sfs.TestRndTree,
		sfs.TestUmask,
		sfs.TestWriteFile,
	)
}

// ClosedFile returns a closed avfs.File.
func (sfs *SuiteFS) ClosedFile(tb testing.TB, testDir string) (f avfs.File, fileName string) {
	tb.Helper()

	fileName = sfs.EmptyFile(tb, testDir)

	vfs := sfs.vfsTest

	f, err := vfs.Open(fileName)
	if err != nil {
		tb.Fatalf("Create %s : want error to be nil, got %v", fileName, err)
	}

	err = f.Close()
	if err != nil {
		tb.Fatalf("Close %s : want error to be nil, got %v", fileName, err)
	}

	return f, fileName
}

// EmptyFile returns an empty file name.
func (sfs *SuiteFS) EmptyFile(tb testing.TB, testDir string) string {
	tb.Helper()

	const emptyFile = "emptyFile"

	vfs := sfs.vfsSetup
	fileName := vfs.Join(testDir, emptyFile)

	_, err := vfs.Stat(fileName)
	if vfs.IsNotExist(err) {
		f, err := vfs.Create(fileName)
		if err != nil {
			tb.Fatalf("Create %s : want error to be nil, got %v", fileName, err)
		}

		err = f.Close()
		if err != nil {
			tb.Fatalf("Close %s : want error to be nil, got %v", fileName, err)
		}
	}

	return fileName
}

// ExistingDir returns an existing directory.
func (sfs *SuiteFS) ExistingDir(tb testing.TB, testDir string) string {
	tb.Helper()

	vfs := sfs.vfsSetup
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return vfs.Join(testDir, defaultDir)
	}

	dirName, err := vfs.MkdirTemp(testDir, defaultDir)
	if err != nil {
		tb.Fatalf("MkdirTemp %s : want error to be nil, got %v", dirName, err)
	}

	_, err = vfs.Stat(dirName)
	if vfs.IsNotExist(err) {
		tb.Fatalf("Stat %s : want error to be nil, got %v", dirName, err)
	}

	return dirName
}

// ExistingFile returns an existing file name with the given content.
func (sfs *SuiteFS) ExistingFile(tb testing.TB, testDir string, content []byte) string {
	tb.Helper()

	vfs := sfs.vfsSetup
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return vfs.Join(testDir, defaultFile)
	}

	f, err := vfs.CreateTemp(testDir, defaultFile)
	if err != nil {
		tb.Fatalf("CreateTemp : want error to be nil, got %v", err)
	}

	fileName := f.Name()

	_, err = f.Write(content)
	if err != nil {
		tb.Fatalf("Write %s : want error to be nil, got %v", fileName, err)
	}

	err = f.Close()
	if err != nil {
		tb.Fatalf("Close %s : want error to be nil, got %v", fileName, err)
	}

	return fileName
}

// NonExistingFile returns the name of a non existing file.
func (sfs *SuiteFS) NonExistingFile(tb testing.TB, testDir string) string {
	tb.Helper()

	vfs := sfs.vfsSetup

	fileName := vfs.Join(testDir, defaultNonExisting)
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return fileName
	}

	_, err := vfs.Stat(fileName)
	if !vfs.IsNotExist(err) {
		tb.Fatalf("Stat : want error to be %v, got %v", avfs.ErrNoSuchFileOrDir, err)
	}

	return fileName
}

// OpenedEmptyFile returns an opened empty avfs.File and its file name.
func (sfs *SuiteFS) OpenedEmptyFile(tb testing.TB, testDir string) (fd avfs.File, fileName string) {
	tb.Helper()

	fileName = sfs.EmptyFile(tb, testDir)

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f, _ := vfs.Open(fileName)

		return f, fileName
	}

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, err := vfs.Open(fileName)
		if err != nil {
			tb.Fatalf("Open %s : want error to be nil, got %v", fileName, err)
		}

		return f, fileName
	}

	f, err := vfs.OpenFile(fileName, os.O_CREATE|os.O_RDWR, avfs.DefaultFilePerm)
	if err != nil {
		tb.Fatalf("OpenFile %s : want error to be nil, got %v", fileName, err)
	}

	return f, fileName
}

// OpenedNonExistingFile returns a non existing avfs.File and its file name.
func (sfs *SuiteFS) OpenedNonExistingFile(tb testing.TB, testDir string) (f avfs.File) {
	tb.Helper()

	fileName := sfs.NonExistingFile(tb, testDir)

	vfs := sfs.vfsTest

	f, err := vfs.Open(fileName)
	if vfs.HasFeature(avfs.FeatBasicFs) && (err == nil || vfs.IsExist(err)) {
		tb.Fatalf("Open %s : want non existing file, got %v", fileName, err)
	}

	return f
}

// RandomDir returns one directory with random empty subdirectories, files and symbolic links.
func (sfs *SuiteFS) RandomDir(tb testing.TB, testDir string) *avfs.RndTree {
	tb.Helper()

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
	if err != nil {
		tb.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rt.CreateTree()
	if err != nil {
		tb.Fatalf("rt.Create : want error to be nil, got %v", err)
	}

	return rt
}

// Dir contains the sample directories.
type Dir struct { //nolint:govet // no fieldalignment for test structs.
	Path      string
	Mode      fs.FileMode
	WantModes []fs.FileMode
}

// GetSampleDirs returns the sample directories used by Mkdir function.
func GetSampleDirs() []*Dir {
	dirs := []*Dir{
		{Path: "/A", Mode: 0o777, WantModes: []fs.FileMode{0o777}},
		{Path: "/B", Mode: 0o755, WantModes: []fs.FileMode{0o755}},
		{Path: "/B/1", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o755}},
		{Path: "/B/1/D", Mode: 0o700, WantModes: []fs.FileMode{0o755, 0o755, 0o700}},
		{Path: "/B/1/E", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o755, 0o755}},
		{Path: "/B/2", Mode: 0o750, WantModes: []fs.FileMode{0o755, 0o750}},
		{Path: "/B/2/F", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o750, 0o755}},
		{Path: "/B/2/F/3", Mode: 0o755, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755}},
		{Path: "/B/2/F/3/G", Mode: 0o777, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755, 0o777}},
		{Path: "/B/2/F/3/G/4", Mode: 0o777, WantModes: []fs.FileMode{0o755, 0o750, 0o755, 0o755, 0o777, 0o777}},
		{Path: "/C", Mode: 0o750, WantModes: []fs.FileMode{0o750}},
		{Path: "/C/5", Mode: 0o750, WantModes: []fs.FileMode{0o750, 0o750}},
	}

	return dirs
}

// GetSampleDirsAll returns the sample directories used by MkdirAll function.
func GetSampleDirsAll() []*Dir {
	dirs := []*Dir{
		{Path: "/H/6", Mode: 0o750, WantModes: []fs.FileMode{0o750, 0o750}},
		{Path: "/H/6/I/7", Mode: 0o755, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755}},
		{Path: "/H/6/I/7/J/8", Mode: 0o777, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755, 0o777, 0o777}},
	}

	return dirs
}

// SampleDirs create sample directories and testDir directory if necessary.
func (sfs *SuiteFS) SampleDirs(tb testing.TB, testDir string) []*Dir {
	tb.Helper()

	vfs := sfs.vfsSetup

	err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %v", testDir, err)
	}

	dirs := GetSampleDirs()
	for _, dir := range dirs {
		path := vfs.Join(testDir, dir.Path)

		err = vfs.Mkdir(path, dir.Mode)
		if err != nil {
			tb.Fatalf("Mkdir %s : want error to be nil, got %v", path, err)
		}
	}

	return dirs
}

// File contains the sample files.
type File struct { //nolint:govet // no fieldalignment for test structs.
	Path    string
	Mode    fs.FileMode
	Content []byte
}

// GetSampleFiles returns the sample files.
func GetSampleFiles() []*File {
	files := []*File{
		{Path: "/file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("file")},
		{Path: "/A/afile1.txt", Mode: 0o777, Content: []byte("afile1")},
		{Path: "/A/afile2.txt", Mode: avfs.DefaultFilePerm, Content: []byte("afile2")},
		{Path: "/A/afile3.txt", Mode: 0o600, Content: []byte("afile3")},
		{Path: "/B/1/1file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("1file")},
		{Path: "/B/1/E/efile.txt", Mode: avfs.DefaultFilePerm, Content: []byte("efile")},
		{Path: "/B/2/F/3/3file1.txt", Mode: 0o640, Content: []byte("3file1")},
		{Path: "/B/2/F/3/3file2.txt", Mode: avfs.DefaultFilePerm, Content: []byte("3file2")},
		{Path: "/B/2/F/3/G/4/4file.txt", Mode: avfs.DefaultFilePerm, Content: []byte("4file")},
		{Path: "/C/cfile.txt", Mode: avfs.DefaultFilePerm, Content: []byte("cfile")},
	}

	return files
}

// SampleFiles create the sample files.
func (sfs *SuiteFS) SampleFiles(tb testing.TB, testDir string) []*File {
	tb.Helper()

	vfs := sfs.vfsSetup

	files := GetSampleFiles()
	for _, file := range files {
		path := vfs.Join(testDir, file.Path)

		err := vfs.WriteFile(path, file.Content, file.Mode)
		if err != nil {
			tb.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
		}
	}

	return files
}

// Symlink contains sample symbolic links.
type Symlink struct {
	NewName string
	OldName string
}

// GetSampleSymlinks returns the sample symbolic links.
func GetSampleSymlinks(vfs avfs.Featurer) []*Symlink {
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	symlinks := []*Symlink{
		{NewName: "/A/lroot", OldName: "/"},
		{NewName: "/lC", OldName: "/C"},
		{NewName: "/B/1/lafile2.txt", OldName: "/A/afile2.txt"},
		{NewName: "/B/2/lf", OldName: "/B/2/F"},
		{NewName: "/B/2/F/3/llf", OldName: "/B/2/lf"},
		{NewName: "/C/lllf", OldName: "/B/2/F/3/llf"},
		{NewName: "/A/l3file2.txt", OldName: "/C/lllf/3/3file2.txt"},
		{NewName: "/C/lNonExist", OldName: "/A/path/to/a/non/existing/file"},
	}

	return symlinks
}

// SymlinkEval contains the data to evaluate the sample symbolic links.
type SymlinkEval struct {
	NewName   string      // Name of the symbolic link.
	OldName   string      // Value of the symbolic link.
	IsSymlink bool        //
	Mode      fs.FileMode //
}

// GetSampleSymlinksEval return the sample symbolic links to evaluate.
func GetSampleSymlinksEval(vfs avfs.Featurer) []*SymlinkEval {
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sl := []*SymlinkEval{
		{NewName: "/A/lroot/", OldName: "/", IsSymlink: true, Mode: fs.ModeDir | 0o777},
		{NewName: "/A/lroot/B", OldName: "/B", IsSymlink: false, Mode: fs.ModeDir | 0o755},
		{NewName: "/", OldName: "/", IsSymlink: false, Mode: fs.ModeDir | 0o777},
		{NewName: "/lC", OldName: "/C", IsSymlink: true, Mode: fs.ModeDir | 0o750},
		{NewName: "/B/1/lafile2.txt", OldName: "/A/afile2.txt", IsSymlink: true, Mode: 0o644},
		{NewName: "/B/2/lf", OldName: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewName: "/B/2/F/3/llf", OldName: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewName: "/C/lllf", OldName: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewName: "/B/2/F/3/llf", OldName: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewName: "/C/lllf/3/3file1.txt", OldName: "/B/2/F/3/3file1.txt", IsSymlink: false, Mode: 0o640},
	}

	return sl
}

// SampleSymlinks creates the sample symbolic links.
func (sfs *SuiteFS) SampleSymlinks(tb testing.TB, testDir string) []*Symlink {
	tb.Helper()

	vfs := sfs.vfsSetup

	symlinks := GetSampleSymlinks(vfs)
	for _, sl := range symlinks {
		oldPath := vfs.Join(testDir, sl.OldName)
		newPath := vfs.Join(testDir, sl.NewName)

		err := vfs.Symlink(oldPath, newPath)
		if err != nil {
			tb.Fatalf("TestSymlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
		}
	}

	return symlinks
}

// CheckNoError checks if there is no error.
func CheckNoError(tb testing.TB, name string, err error) bool {
	tb.Helper()

	if err != nil {
		tb.Errorf("%s : want error to be nil, got %v", name, err)

		return false
	}

	return true
}

// CheckInvalid checks if the error is fs.ErrInvalid.
func CheckInvalid(tb testing.TB, name string, err error) {
	tb.Helper()

	if err != fs.ErrInvalid {
		tb.Errorf("%s : want error to be %v, got %v", name, fs.ErrInvalid, err)
	}
}

// CheckPanic checks that function f panics.
func CheckPanic(tb testing.TB, funcName string, f func()) {
	tb.Helper()

	defer func() {
		if r := recover(); r == nil {
			tb.Errorf("%s : want function to panic, not panicking", funcName)
		}
	}()

	f()
}

// checkPathError stores the current fs.PathError test data.
type checkPathError struct {
	tb       testing.TB
	err      *fs.PathError
	foundErr bool
}

// CheckPathError checks if err is a fs.PathError.
func CheckPathError(tb testing.TB, err error) *checkPathError { //nolint:revive // No need to export checkPathError to use it.
	tb.Helper()

	foundErr := false

	if err == nil {
		foundErr = true

		tb.Error("want error to be not nil, got nil")
	}

	e, ok := err.(*fs.PathError)
	if !ok {
		foundErr = true

		tb.Errorf("want error type to be *fs.PathError, got %v : %v", reflect.TypeOf(err), err)
	}

	return &checkPathError{tb: tb, err: e, foundErr: foundErr}
}

// Op checks if op is equal to the current fs.PathError Op for one of the osTypes.
func (cp *checkPathError) Op(op string, osTypes ...avfs.OSType) *checkPathError {
	cp.tb.Helper()

	if cp.foundErr {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, ost := range osTypes {
		if ost == avfs.Cfg.OSType() {
			canTest = true
		}
	}

	if !canTest {
		return cp
	}

	err := cp.err
	if err.Op != op {
		cp.tb.Errorf("want Op to be %s, got %s", op, err.Op)
	}

	return cp
}

// OpStat checks if the current fs.PathError Op is a Stat Op.
func (cp *checkPathError) OpStat() *checkPathError {
	cp.tb.Helper()

	return cp.
		Op("stat", avfs.OsLinux).
		Op("CreateFile", avfs.OsWindows)
}

// OpLstat checks if the current fs.PathError Op is a Lstat Op.
func (cp *checkPathError) OpLstat() *checkPathError {
	cp.tb.Helper()

	return cp.
		Op("lstat", avfs.OsLinux).
		Op("CreateFile", avfs.OsWindows)
}

// Path checks the path of the current fs.PathError.
func (cp *checkPathError) Path(path string) *checkPathError {
	cp.tb.Helper()

	if cp.foundErr {
		return cp
	}

	err := cp.err
	if err.Path != path {
		cp.tb.Errorf("want Path to be %s, got %s", path, err.Path)
	}

	return cp
}

// Err checks the error of current fs.PathError.
func (cp *checkPathError) Err(wantErr error, osTypes ...avfs.OSType) *checkPathError {
	cp.tb.Helper()

	if cp.foundErr {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, ost := range osTypes {
		if ost == avfs.Cfg.OSType() {
			canTest = true
		}
	}

	if !canTest {
		return cp
	}

	err := cp.err
	if err.Err == wantErr || err.Err.Error() == wantErr.Error() {
		return cp
	}

	cp.tb.Errorf("%s : want error to be %v, got %v", err.Path, wantErr, err.Err)

	return cp
}

// ErrPermDenied checks if the current fs.PathError is a permission denied error.
func (cp *checkPathError) ErrPermDenied() *checkPathError {
	cp.tb.Helper()

	return cp.
		Err(avfs.ErrPermDenied, avfs.OsLinux).
		Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
}

// checkLinkError stores the current os.LinkError test data.
type checkLinkError struct {
	tb       testing.TB
	err      *os.LinkError
	foundErr bool
}

// CheckLinkError checks if err is a os.LinkError.
func CheckLinkError(tb testing.TB, err error) *checkLinkError { //nolint:revive // No need to export checkLinkError to use it.
	tb.Helper()

	foundErr := false

	if err == nil {
		foundErr = true

		tb.Error("want error to be not nil")
	}

	e, ok := err.(*os.LinkError)
	if !ok {
		foundErr = true

		tb.Errorf("want error type to be *os.LinkError, got %v : %v", reflect.TypeOf(err), err)
	}

	return &checkLinkError{tb: tb, err: e, foundErr: foundErr}
}

// Op checks if wantOp is equal to the current os.LinkError Op.
func (cl *checkLinkError) Op(wantOp string, osTypes ...avfs.OSType) *checkLinkError {
	cl.tb.Helper()

	if cl.foundErr {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, ost := range osTypes {
		if ost == avfs.Cfg.OSType() {
			canTest = true
		}
	}

	if !canTest {
		return cl
	}

	err := cl.err
	if err.Op != wantOp {
		cl.tb.Errorf("want Op to be %s, got %s", wantOp, err.Op)
	}

	return cl
}

// Old checks the old path of the current os.LinkError.
func (cl *checkLinkError) Old(wantOld string) *checkLinkError {
	cl.tb.Helper()

	if cl.foundErr {
		return cl
	}

	err := cl.err
	if err.Old != wantOld {
		cl.tb.Errorf("want old path to be %s, got %s", wantOld, err.Old)
	}

	return cl
}

// New checks the new path of the current os.LinkError.
func (cl *checkLinkError) New(wantNew string) *checkLinkError {
	cl.tb.Helper()

	if cl.foundErr {
		return cl
	}

	err := cl.err
	if err.New != wantNew {
		cl.tb.Errorf("want new path to be %s, got %s", wantNew, err.New)
	}

	return cl
}

// Err checks the error of current os.LinkError.
func (cl *checkLinkError) Err(wantErr error, osTypes ...avfs.OSType) *checkLinkError {
	cl.tb.Helper()

	if cl.foundErr {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, ost := range osTypes {
		if ost == avfs.Cfg.OSType() {
			canTest = true
		}
	}

	if !canTest {
		return cl
	}

	err := cl.err
	if err.Err == wantErr || err.Err.Error() == wantErr.Error() {
		return cl
	}

	cl.tb.Errorf("want error to be %v, got %v", wantErr, err.Err)

	return cl
}

// ErrPermDenied checks if the current os.LinkError is a permission denied error.
func (cl *checkLinkError) ErrPermDenied() *checkLinkError {
	cl.tb.Helper()

	return cl.
		Err(avfs.ErrPermDenied, avfs.OsLinux).
		Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
}
