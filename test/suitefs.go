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
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
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
	osType      avfs.OSType        // osType is the operating system of the filesystem te test.
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
	initUser := vfs.CurrentUser()

	_, file, _, _ := runtime.Caller(0)
	initDir := filepath.Dir(file)

	canTestPerm := vfs.HasFeature(avfs.FeatBasicFs) &&
		vfs.HasFeature(avfs.FeatIdentityMgr) &&
		initUser.IsRoot()

	sfs := &SuiteFS{
		vfsSetup:    vfs,
		vfsTest:     vfs,
		initDir:     initDir,
		initUser:    initUser,
		rootDir:     vfs.GetTempDir(),
		maxRace:     1000,
		osType:      vfsutils.RunTimeOS(),
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

	rootDir, err := vfs.TempDir("", avfs.Avfs)
	if err != nil {
		tb.Fatalf("TempDir %s : want error to be nil, got %s", rootDir, err)
	}

	// Default permissions on rootDir are 0o700 (generated by TempDir()).
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
		sfs.Groups = CreateGroups(tb, vfsSetup, "")
		sfs.Users = CreateUsers(tb, vfsSetup, "")
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

// WithOS returns an option function which sets the operating system for the tests.
func WithOS(osType avfs.OSType) Option {
	return func(sfs *SuiteFS) {
		sfs.osType = osType
	}
}

// OSType returns the operating system, real or simulated.
func (sfs *SuiteFS) OSType() avfs.OSType {
	return sfs.osType
}

// User sets the test user to userName.
func (sfs *SuiteFS) User(tb testing.TB, userName string) avfs.UserReader {
	vfs := sfs.vfsSetup

	u := vfs.CurrentUser()
	if !sfs.canTestPerm || u.Name() == userName {
		return u
	}

	u, err := vfs.User(userName)
	if err != nil {
		tb.Fatalf("User %s : want error to be nil, got %s", userName, err)
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
		sfs.User(t, sfs.initUser.Name())

		err := os.Chdir(sfs.initDir)
		if err != nil {
			t.Fatalf("Chdir %s : want error to be nil, got %v", sfs.initDir, err)
		}
	}()

	for _, tf := range testFuncs {
		sfs.User(t, userName)

		funcName := functionName(tf)
		testDir := vfs.Join(sfs.rootDir, funcName)

		sfs.CreateDir(t, testDir, 0o777)
		sfs.ChangeDir(t, testDir)

		t.Run(funcName, func(t *testing.T) {
			tf(t, testDir)
		})

		sfs.RemoveTestDir(t, testDir)
	}
}

// SuiteBenchFunc is a bench function to be used by RunBenchs.
type SuiteBenchFunc func(b *testing.B, testDir string)

// RunBenchs runs all benchmark functions benchFuncs specified as user userName.
func (sfs *SuiteFS) RunBenchs(b *testing.B, userName string, benchFuncs ...SuiteBenchFunc) {
	vfs := sfs.vfsSetup

	defer func() {
		sfs.User(b, sfs.initUser.Name())

		err := os.Chdir(sfs.initDir)
		if err != nil {
			b.Fatalf("Chdir %s : want error to be nil, got %v", sfs.initDir, err)
		}
	}()

	for _, bf := range benchFuncs {
		sfs.User(b, userName)

		funcName := functionName(bf)
		testDir := vfs.Join(sfs.rootDir, funcName)

		sfs.CreateDir(b, testDir, 0o777)
		sfs.ChangeDir(b, testDir)

		b.Run(funcName, func(b *testing.B) {
			bf(b, testDir)
		})

		sfs.RemoveTestDir(b, testDir)
	}
}

// functionName return the name of a function or an empty string if not available.
func functionName(i interface{}) string {
	pc := reflect.ValueOf(i).Pointer()
	if pc == 0 {
		return ""
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}

	name := fn.Name()

	start := strings.LastIndex(name, ".") + 1
	if start == -1 {
		return name
	}

	end := strings.LastIndex(name, "-")
	if end == -1 {
		return name[start:]
	}

	return name[start:end]
}

// CreateDir creates a directory for the tests.
func (sfs *SuiteFS) CreateDir(tb testing.TB, dir string, mode os.FileMode) {
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

	err := vfs.RemoveAll(testDir)
	if err == nil || !sfs.canTestPerm {
		return
	}

	// Cleanup permissions for RemoveAll()
	// as the user who started the tests.
	sfs.User(tb, sfs.initUser.Name())

	err = vfs.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
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
	sfs.RunTests(t, UsrTest,
		// VFS tests
		sfs.TestClone,
		sfs.TestChdir,
		sfs.TestChtimes,
		sfs.TestCreate,
		sfs.TestEvalSymlink,
		sfs.TestGetTempDir,
		sfs.TestLink,
		sfs.TestLstat,
		sfs.TestMkdir,
		sfs.TestMkdirAll,
		sfs.TestOpen,
		sfs.TestOpenFileWrite,
		sfs.TestReadlink,
		sfs.TestRemove,
		sfs.TestRemoveAll,
		sfs.TestRename,
		sfs.TestSameFile,
		sfs.TestStat,
		sfs.TestSymlink,
		sfs.TestTruncate,
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
	sfs.RunTests(t, avfs.UsrRoot,
		sfs.TestChmod,
		sfs.TestChown,
		sfs.TestChroot,
		sfs.TestLchown,
		sfs.TestFileChmod,
		sfs.TestFileChown)
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
		sfs.TestRel,
		sfs.TestSplit,
		sfs.TestWalk,

		// ioutils functions
		sfs.TestReadDir,
		sfs.TestReadFile,
		sfs.TestTempDir,
		sfs.TestTempFile,
		sfs.TestWriteFile,

		// other functions
		sfs.TestCopyFile,
		sfs.TestCreateBaseDirs,
		sfs.TestDirExists,
		sfs.TestExists,
		sfs.TestHashFile,
		sfs.TestRndTree,
		sfs.TestSegmentPath,
		sfs.TestToSysStat,
		sfs.TestUmask,
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

	dirName, err := vfs.TempDir(testDir, defaultDir)
	if err != nil {
		tb.Fatalf("TempDir %s : want error to be nil, got %v", dirName, err)
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

	f, err := vfs.TempFile(testDir, defaultFile)
	if err != nil {
		tb.Fatalf("TempFile : want error to be nil, got %v", err)
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
func (sfs *SuiteFS) RandomDir(tb testing.TB, testDir string) *vfsutils.RndTree {
	tb.Helper()

	vfs := sfs.vfsSetup
	RndParamsOneDir := vfsutils.RndTreeParams{
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

	rt, err := vfsutils.NewRndTree(vfs, testDir, &RndParamsOneDir)
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
type Dir struct { //nolint:govet // no fieldalignment for simple structs
	Path      string
	Mode      os.FileMode
	WantModes []os.FileMode
}

// GetSampleDirs returns the sample directories used by Mkdir function.
func GetSampleDirs() []*Dir {
	dirs := []*Dir{
		{Path: "/A", Mode: 0o777, WantModes: []os.FileMode{0o777}},
		{Path: "/B", Mode: 0o755, WantModes: []os.FileMode{0o755}},
		{Path: "/B/1", Mode: 0o755, WantModes: []os.FileMode{0o755, 0o755}},
		{Path: "/B/1/D", Mode: 0o700, WantModes: []os.FileMode{0o755, 0o755, 0o700}},
		{Path: "/B/1/E", Mode: 0o755, WantModes: []os.FileMode{0o755, 0o755, 0o755}},
		{Path: "/B/2", Mode: 0o750, WantModes: []os.FileMode{0o755, 0o750}},
		{Path: "/B/2/F", Mode: 0o755, WantModes: []os.FileMode{0o755, 0o750, 0o755}},
		{Path: "/B/2/F/3", Mode: 0o755, WantModes: []os.FileMode{0o755, 0o750, 0o755, 0o755}},
		{Path: "/B/2/F/3/G", Mode: 0o777, WantModes: []os.FileMode{0o755, 0o750, 0o755, 0o755, 0o777}},
		{Path: "/B/2/F/3/G/4", Mode: 0o777, WantModes: []os.FileMode{0o755, 0o750, 0o755, 0o755, 0o777, 0o777}},
		{Path: "/C", Mode: 0o750, WantModes: []os.FileMode{0o750}},
		{Path: "/C/5", Mode: 0o750, WantModes: []os.FileMode{0o750, 0o750}},
	}

	return dirs
}

// GetSampleDirsAll returns the sample directories used by MkdirAll function.
func GetSampleDirsAll() []*Dir {
	dirs := []*Dir{
		{Path: "/H/6", Mode: 0o750, WantModes: []os.FileMode{0o750, 0o750}},
		{Path: "/H/6/I/7", Mode: 0o755, WantModes: []os.FileMode{0o750, 0o750, 0o755, 0o755}},
		{Path: "/H/6/I/7/J/8", Mode: 0o777, WantModes: []os.FileMode{0o750, 0o750, 0o755, 0o755, 0o777, 0o777}},
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
type File struct { //nolint:govet // no fieldalignment for simple structs
	Path    string
	Mode    os.FileMode
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
type SymlinkEval struct { //nolint:govet // no fieldalignment for simple structs
	NewName   string
	OldName   string
	WantErr   error
	IsSymlink bool
	Mode      os.FileMode
}

// GetSampleSymlinksEval return the sample symbolic links to evaluate.
func GetSampleSymlinksEval(vfs avfs.Featurer) []*SymlinkEval {
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sl := []*SymlinkEval{
		{NewName: "/A/lroot/", OldName: "/", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o777},
		{NewName: "/A/lroot/B", OldName: "/B", WantErr: nil, IsSymlink: false, Mode: os.ModeDir | 0o755},
		{NewName: "/C/lNonExist", OldName: "A/path", WantErr: avfs.ErrNoSuchFileOrDir, IsSymlink: true},
		{NewName: "/", OldName: "/", WantErr: nil, IsSymlink: false, Mode: os.ModeDir | 0o777},
		{NewName: "/lC", OldName: "/C", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o750},
		{NewName: "/B/1/lafile2.txt", OldName: "/A/afile2.txt", WantErr: nil, IsSymlink: true, Mode: 0o644},
		{NewName: "/B/2/lf", OldName: "/B/2/F", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o755},
		{NewName: "/B/2/F/3/llf", OldName: "/B/2/F", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o755},
		{NewName: "/C/lllf", OldName: "/B/2/F", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o755},
		{NewName: "/B/2/F/3/llf", OldName: "/B/2/F", WantErr: nil, IsSymlink: true, Mode: os.ModeDir | 0o755},
		{NewName: "/C/lllf/3/3file1.txt", OldName: "/B/2/F/3/3file1.txt", WantErr: nil, IsSymlink: false, Mode: 0o640},
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

// CheckInvalid checks if the error in os.ErrInvalid.
func CheckInvalid(tb testing.TB, funcName string, err error) {
	tb.Helper()

	if err != os.ErrInvalid {
		tb.Errorf("%s : want error to be %v, got %v", funcName, os.ErrInvalid, err)
	}
}

// CheckPanic checks that function f panics.
func CheckPanic(tb testing.TB, funcName string, f func()) {
	tb.Helper()

	defer func() {
		if r := recover(); r == nil {
			tb.Errorf("%s : want function to panic, not panicing", funcName)
		}
	}()

	f()
}

type CheckPath struct {
	tb   testing.TB
	err  *os.PathError
	halt bool
}

func CheckPathError(tb testing.TB, err error) *CheckPath {
	tb.Helper()

	halt := false

	if err == nil {
		halt = true

		tb.Error("want error to be not nil")
	}

	e, ok := err.(*os.PathError)
	if !ok {
		halt = true

		tb.Errorf("want error type to be *fs.PathError, got %v", reflect.TypeOf(err))
	}

	return &CheckPath{tb: tb, err: e, halt: halt}
}

func (cp *CheckPath) Op(wantOp string) *CheckPath {
	cp.tb.Helper()

	if cp.halt {
		return cp
	}

	err := cp.err
	if err.Op != wantOp {
		cp.tb.Errorf("want Op to be %s, got %s", wantOp, err.Op)
	}

	return cp
}

func (cp *CheckPath) OpLstat(vfs avfs.VFS) *CheckPath {
	op := "lstat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return cp.Op(op)
}

func (cp *CheckPath) OpStat(vfs avfs.VFS) *CheckPath {
	op := "stat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return cp.Op(op)
}

func (cp *CheckPath) Path(wantPath string) *CheckPath {
	cp.tb.Helper()

	if cp.halt {
		return cp
	}

	err := cp.err
	if err.Path != wantPath {
		cp.tb.Errorf("want Path to be %s, got %s", wantPath, err.Path)
	}

	return cp
}

func (cp *CheckPath) Err(wantErr error) *CheckPath {
	cp.tb.Helper()

	if cp.halt {
		return cp
	}

	err := cp.err
	if err.Err != wantErr {
		cp.tb.Errorf("want error to be %v, got %v", wantErr, err.Err)
	}

	return cp
}

func (cp *CheckPath) ErrAsString(wantErr error) *CheckPath {
	cp.tb.Helper()

	if cp.halt {
		return cp
	}

	err := cp.err
	if err.Err.Error() != wantErr.Error() {
		cp.tb.Errorf("want error to be %v, got %v", wantErr, err.Err)
	}

	return cp
}

type CheckLink struct {
	tb   testing.TB
	err  *os.LinkError
	halt bool
}

func CheckLinkError(tb testing.TB, err error) *CheckLink {
	tb.Helper()

	halt := false

	if err == nil {
		halt = true

		tb.Error("want error to be not nil")
	}

	e, ok := err.(*os.LinkError)
	if !ok {
		halt = true

		tb.Errorf("want error type to be *fs.LinkError, got %v", reflect.TypeOf(err))
	}

	return &CheckLink{tb: tb, err: e, halt: halt}
}

func (cl *CheckLink) Op(wantOp string) *CheckLink {
	cl.tb.Helper()

	if cl.halt {
		return cl
	}

	err := cl.err
	if err.Op != wantOp {
		cl.tb.Errorf("want Op to be %s, got %s", wantOp, err.Op)
	}

	return cl
}

func (cl *CheckLink) Old(wantOld string) *CheckLink {
	cl.tb.Helper()

	if cl.halt {
		return cl
	}

	err := cl.err
	if err.Old != wantOld {
		cl.tb.Errorf("want old path to be %s, got %s", wantOld, err.Old)
	}

	return cl
}

func (cl *CheckLink) New(wantNew string) *CheckLink {
	cl.tb.Helper()

	if cl.halt {
		return cl
	}

	err := cl.err
	if err.New != wantNew {
		cl.tb.Errorf("want new path to be %s, got %s", wantNew, err.New)
	}

	return cl
}

func (cl *CheckLink) Err(wantErr error) *CheckLink {
	cl.tb.Helper()

	if cl.halt {
		return cl
	}

	err := cl.err
	if err.Err != wantErr {
		cl.tb.Errorf("want error to be %v, got %v", wantErr, err.Err)
	}

	return cl
}
