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
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// SuiteFS is a test suite for virtual file systems.
type SuiteFS struct {
	vfsSetup    avfs.VFS           // vfsSetup is the file system used to setup the tests (generally with read/write access).
	vfsTest     avfs.VFS           // vfsTest is the file system used to run the tests.
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
	canTestPerm := vfs.HasFeature(avfs.FeatBasicFs) &&
		vfs.HasFeature(avfs.FeatIdentityMgr) &&
		initUser.IsRoot()

	sfs := &SuiteFS{
		vfsSetup:    vfs,
		vfsTest:     vfs,
		initUser:    initUser,
		rootDir:     vfs.GetTempDir(),
		maxRace:     1000,
		osType:      vfsutils.RunTimeOS(),
		canTestPerm: canTestPerm,
	}

	defer func() {
		info := "Info vfs : type = " + sfs.vfsTest.Type()
		if sfs.vfsTest.Name() != "" {
			info += ", name = " + sfs.vfsTest.Name()
		}

		tb.Log(info)
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
func (sfs *SuiteFS) User(tb testing.TB, userName string) (avfs.VFS, avfs.UserReader) {
	vfs := sfs.vfsSetup

	u := vfs.CurrentUser()
	if !sfs.canTestPerm || u.Name() == userName {
		return vfs, u
	}

	u, err := vfs.User(userName)
	if err != nil {
		tb.Fatalf("User %s : want error to be nil, got %s", userName, err)
	}

	return vfs, u
}

// VFSTest returns the file system used to run the tests.
func (sfs *SuiteFS) VFSTest() avfs.VFS {
	return sfs.vfsTest
}

// VFSSetup returns the file system used to setup the tests.
func (sfs *SuiteFS) VFSSetup() avfs.VFS {
	return sfs.vfsSetup
}

// SuiteTestFunc defines the parameters of all tests functions.
type SuiteTestFunc func(t *testing.T, testDir string)

// RunTests runs all test functions testFuncs specified as user userName.
func (sfs *SuiteFS) RunTests(t *testing.T, userName string, stFuncs ...SuiteTestFunc) {
	vfs := sfs.vfsSetup

	_, _ = sfs.User(t, userName)
	defer sfs.User(t, sfs.initUser.Name())

	for _, stFunc := range stFuncs {
		funcName := runtime.FuncForPC(reflect.ValueOf(stFunc).Pointer()).Name()
		funcName = funcName[strings.LastIndex(funcName, ".")+1 : strings.LastIndex(funcName, "-")]
		testDir := vfs.Join(sfs.rootDir, funcName)

		sfs.CreateTestDir(t, testDir)

		t.Run(funcName, func(t *testing.T) {
			stFunc(t, testDir)
		})

		sfs.RemoveTestDir(t, testDir)
	}
}

// BenchFunc
type BenchFunc func(b *testing.B, testDir string)

// RunBenchs
func (sfs *SuiteFS) RunBenchs(b *testing.B, userName string, benchFuncs ...BenchFunc) {
	vfs := sfs.vfsSetup

	_, _ = sfs.User(b, userName)
	defer sfs.User(b, sfs.initUser.Name())

	for _, bf := range benchFuncs {
		funcName := runtime.FuncForPC(reflect.ValueOf(bf).Pointer()).Name()
		funcName = funcName[strings.LastIndex(funcName, ".")+1 : strings.LastIndex(funcName, "-")]
		testDir := vfs.Join(sfs.rootDir, funcName)

		sfs.CreateTestDir(b, testDir)

		b.Run(funcName, func(b *testing.B) {
			bf(b, testDir)
		})

		sfs.RemoveTestDir(b, testDir)
	}
}

// CreateTestDir creates the base directory for the tests.
func (sfs *SuiteFS) CreateTestDir(tb testing.TB, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %s", testDir, err)
	}

	_, err = vfs.Stat(testDir)
	if err != nil {
		tb.Fatalf("Stat %s : want error to be nil, got %s", testDir, err)
	}

	err = vfs.Chmod(testDir, 0o777)
	if err != nil {
		tb.Fatalf("Chmod %s : want error to be nil, got %s", testDir, err)
	}

	err = vfs.Chdir(testDir)
	if err != nil {
		tb.Fatalf("Chdir %s : want error to be nil, got %s", testDir, err)
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

	// Some dirs were not removed as
	vfs, _ = sfs.User(tb, sfs.initUser.Name())

	// Cleanup permissions for RemoveAll()
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

// TestAll runs all file systems tests.
func (sfs *SuiteFS) TestAll(t *testing.T) {
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
		sfs.TestReadDir,
		sfs.TestReadFile,
		sfs.TestReadlink,
		sfs.TestRemove,
		sfs.TestRemoveAll,
		sfs.TestRename,
		sfs.TestSameFile,
		sfs.TestStat,
		sfs.TestSymlink,
		sfs.TestTempDir,
		sfs.TestTempFile,
		sfs.TestTruncate,
		sfs.TestWriteFile,
		sfs.TestWriteString,
		sfs.TestToSysStat,
		sfs.TestUmask,

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
		sfs.TestFileWriteTime,

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
		sfs.TestWalk)

	// Tests to be run as root
	sfs.RunTests(t, avfs.UsrRoot,
		sfs.TestChmod,
		sfs.TestChown,
		sfs.TestChroot,
		sfs.TestLchown,
		sfs.TestFileChmod,
		sfs.TestFileChown)
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

	const existingDir = "existingDir"

	vfs := sfs.vfsSetup
	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return vfs.Join(testDir, existingDir)
	}

	dirName, err := vfs.TempDir(testDir, existingDir)
	if err != nil {
		tb.Fatalf("Mkdir %s : want error to be nil, got %v", dirName, err)
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

	const existingFile = "existingFile"

	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return vfs.Join(testDir, existingFile)
	}

	f, err := vfs.TempFile(testDir, existingFile)
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

	const nonExistingFile = "nonExistingFile"

	vfs := sfs.vfsSetup

	fileName := vfs.Join(testDir, nonExistingFile)
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
		MinDepth:    1,
		MaxDepth:    1,
		MinName:     4,
		MaxName:     32,
		MinDirs:     10,
		MaxDirs:     20,
		MinFiles:    50,
		MaxFiles:    100,
		MinFileLen:  0,
		MaxFileLen:  0,
		MinSymlinks: 5,
		MaxSymlinks: 10,
	}

	rndTree, err := vfsutils.NewRndTree(vfs, &RndParamsOneDir)
	if err != nil {
		tb.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(testDir)
	if err != nil {
		tb.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	return rndTree
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

// CheckPathError checks errors of type os.PathError.
func CheckPathError(tb testing.TB, testName, wantOp, wantPath string, wantErr, gotErr error) {
	tb.Helper()

	if gotErr == nil {
		tb.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.PathError)
	if !ok {
		tb.Fatalf("%s %s : want error type *os.PathError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if wantOp != err.Op || wantPath != err.Path || (wantErr != err.Err && wantErr.Error() != err.Err.Error()) {
		wantPathErr := &os.PathError{Op: wantOp, Path: wantPath, Err: wantErr}
		tb.Errorf("%s %s: want error to be %v, got %v", testName, wantPath, wantPathErr, gotErr)
	}
}

// CheckSyscallError checks errors of type os.SyscallError.
func CheckSyscallError(tb testing.TB, testName, wantOp, wantPath string, wantErr, gotErr error) {
	tb.Helper()

	if gotErr == nil {
		tb.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.SyscallError)
	if !ok {
		tb.Fatalf("%s %s : want error type *os.SyscallError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if err.Syscall != wantOp || err.Err != wantErr {
		tb.Errorf("%s %s : want error to be %v, got %v", testName, wantPath, wantErr, err)
	}
}

// CheckLinkError checks errors of type os.LinkError.
func CheckLinkError(tb testing.TB, testName, wantOp, oldPath, newPath string, wantErr, gotErr error) {
	tb.Helper()

	if gotErr == nil {
		tb.Fatalf("%s %s : want error to be %v, got nil", testName, newPath, wantErr)
	}

	err, ok := gotErr.(*os.LinkError)
	if !ok {
		tb.Fatalf("%s %s : want error type *os.LinkError,\n got %v", testName, newPath, reflect.TypeOf(gotErr))
	}

	if err.Op != wantOp || err.Err != wantErr {
		tb.Errorf("%s %s %s : want error to be %v,\n got %v", testName, oldPath, newPath, wantErr, err)
	}
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
