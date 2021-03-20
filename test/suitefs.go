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
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// SuiteFS is a test suite for virtual file systems.
type SuiteFS struct {
	// vfsSetup is the file system used to setup the tests (generally with read and write access).
	vfsSetup avfs.VFS

	// vfsTest is the file system used to run the tests.
	vfsTest avfs.VFS

	// rootDir is the root directory for tests, it can be generated automatically or specified with WithRootDir().
	rootDir string

	// maxRace is the maximum number of concurrent goroutines used in race tests.
	maxRace int

	// osType is the operating system of the filesystem te test.
	osType avfs.OSType

	// Groups contains the test groups created with the identity manager.
	Groups []avfs.GroupReader

	// Users contains the test users created with the identity manager.
	Users []avfs.UserReader

	// currentUser
	currentUser avfs.UserReader

	// canTestPerm indicates if permissions can be tested.
	canTestPerm bool
}

// Option defines the option function used for initializing SuiteFS.
type Option func(*SuiteFS)

// NewSuiteFS creates a new test suite for a file system.
func NewSuiteFS(tb testing.TB, vfsSetup avfs.VFS, opts ...Option) *SuiteFS {
	if vfsSetup == nil {
		tb.Fatal("New : want vfsSetup to be set, got nil")
	}

	currentUser := vfsSetup.CurrentUser()
	canTestPerm := vfsSetup.HasFeature(avfs.FeatBasicFs) &&
		vfsSetup.HasFeature(avfs.FeatIdentityMgr) &&
		currentUser.IsRoot()

	sfs := &SuiteFS{
		vfsSetup:    vfsSetup,
		vfsTest:     vfsSetup,
		rootDir:     "",
		maxRace:     1000,
		osType:      vfsutils.RunTimeOS(),
		currentUser: currentUser,
		canTestPerm: canTestPerm,
	}

	if canTestPerm {
		sfs.Groups = CreateGroups(tb, vfsSetup, "")
		sfs.Users = CreateUsers(tb, vfsSetup, "")

		_, err := vfsSetup.User(UsrTest)
		if err != nil {
			tb.Fatalf("User %s : want error to be nil, got %s", UsrTest, err)
		}
	}

	for _, opt := range opts {
		opt(sfs)
	}

	info := "Info vfs : type = " + sfs.vfsTest.Type()
	if sfs.vfsTest.Name() != "" {
		info += ", name = " + sfs.vfsTest.Name()
	}

	tb.Log(info)

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

// CreateAndOpenFile creates, open a file and returns an avfs.File.
func (sfs *SuiteFS) CreateAndOpenFile(t *testing.T, rootDir string) avfs.File {
	t.Helper()

	if !sfs.vfsTest.HasFeature(avfs.FeatBasicFs) {
		return sfs.OpenNonExistingFile(t)
	}

	vfs := sfs.vfsSetup

	f, err := vfs.TempFile(rootDir, avfs.Avfs)
	if err != nil {
		t.Fatalf("TempFile : want error to be nil, got %v", err)
	}

	path := f.Name()

	err = f.Close()
	if err != nil {
		t.Fatalf("Close %s : want error to be nil, got %v", path, err)
	}

	vfs = sfs.vfsTest

	f, err = vfs.Open(path)
	if err != nil {
		t.Fatalf("Open %s : want error to be nil, got %v", path, err)
	}

	return f
}

// ClosedFile returns a closed avfs.File.
func (sfs *SuiteFS) ClosedFile(t *testing.T, rootDir string) (f avfs.File, fileName string) {
	t.Helper()

	vfs := sfs.vfsSetup
	fileName = vfs.Join(rootDir, "closedFile")

	f, err := vfs.Create(fileName)
	if err != nil {
		t.Fatalf("Create %s : want error to be nil, got %v", fileName, err)
	}

	err = f.Close()
	if err != nil {
		t.Fatalf("Close %s : want error to be nil, got %v", fileName, err)
	}

	return f, fileName
}

// CreateDir creates a directory and returns the path to this directory.
func (sfs *SuiteFS) CreateDir(t *testing.T) string {
	t.Helper()

	vfs := sfs.vfsSetup

	path, err := vfs.TempDir(sfs.rootDir, avfs.Avfs)
	if err != nil {
		t.Fatalf("TempDir : want error to be nil, got %v", err)
	}

	return path
}

// CreateEmptyFile creates an empty file and returns the file name.
func (sfs *SuiteFS) CreateEmptyFile(t *testing.T) string {
	t.Helper()

	return sfs.CreateFile(t, nil)
}

// CreateFile creates a file and returns the file name.
func (sfs *SuiteFS) CreateFile(t *testing.T, content []byte) string {
	t.Helper()

	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return sfs.NonExistingFile(t)
	}

	f, err := vfs.TempFile(sfs.rootDir, avfs.Avfs)
	if err != nil {
		t.Fatalf("TempFile : want error to be nil, got %v", err)
	}

	_, err = f.Write(content)
	if err != nil {
		t.Fatalf("Write : want error to be nil, got %v", err)
	}

	err = f.Close()
	if err != nil {
		t.Fatalf("Close : want error to be nil, got %v", err)
	}

	return f.Name()
}

// CreateRootDir creates the root directory for the tests.
// Each test have its own directory in /tmp/avfs.../
// this directory and its descendants are removed by removeDir() function.
func (sfs *SuiteFS) CreateRootDir(tb testing.TB, userName string) (rootDir string, removeDir func()) {
	vfs, _ := sfs.User(tb, userName)

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return avfs.TmpDir, func() {}
	}

	rootDir, err := vfs.TempDir("", avfs.Avfs)
	if err != nil {
		tb.Fatalf("TempDir : want error to be nil, got %s", err)
	}

	if vfsutils.RunTimeOS() == avfs.OsDarwin {
		rootDir, err = vfs.EvalSymlinks(rootDir)
		if err != nil {
			tb.Fatalf("EvalSymlinks : want error to be nil, got %s", err)
		}
	}

	_, err = vfs.Stat(rootDir)
	if err != nil {
		tb.Fatalf("Stat : want error to be nil, got %s", err)
	}

	sfs.rootDir = rootDir

	// default permissions on rootDir are 0o700 (generated by TempDir())
	// make rootDir accessible by anyone.
	err = vfs.Chmod(rootDir, 0o777)
	if err != nil {
		tb.Fatalf("Chmod : want error to be nil, got %s", err)
	}

	err = vfs.Chdir(rootDir)
	if err != nil {
		tb.Fatalf("Chdir : want error to be nil, got %s", err)
	}

	removeDir = func() {
		err = vfs.RemoveAll(rootDir)
		if err != nil {
			vfs, _ := sfs.User(tb, avfs.UsrRoot)

			// Cleanup permissions for RemoveAll()
			err = vfs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				return vfs.Chmod(path, 0o777)
			})

			err = vfs.RemoveAll(rootDir)
			if err != nil {
				tb.Fatalf("RemoveAll %s : want error to be nil, got %v", rootDir, err)
			}
		}
	}

	return rootDir, removeDir
}

// CreateRndDir creates one directory with random empty subdirectories, files and symbolic links.
func (sfs *SuiteFS) CreateRndDir(t *testing.T) *vfsutils.RndTree {
	t.Helper()

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

	vfs := sfs.vfsSetup

	rndTree, err := vfsutils.NewRndTree(vfs, RndParamsOneDir)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(sfs.rootDir)
	if err != nil {
		t.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	return rndTree
}

// NonExistingFile returns the name of a non existing file.
func (sfs *SuiteFS) NonExistingFile(t *testing.T) string {
	t.Helper()

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return "nonExistingFile"
	}

	name := fmt.Sprintf("%s-%x", avfs.Avfs, rand.Int63())
	path := vfs.Join(sfs.rootDir, name)

	_, err := vfs.Stat(path)
	if !vfs.IsNotExist(err) {
		t.Fatalf("Stat : want error to be %v, got %v", avfs.ErrNoSuchFileOrDir, err)
	}

	return path
}

// OpenNonExistingFile open a non existing file and returns an avfs.File.
func (sfs *SuiteFS) OpenNonExistingFile(t *testing.T) avfs.File {
	t.Helper()

	vfs := sfs.vfsTest
	path := sfs.NonExistingFile(t)

	f, err := vfs.Open(path)
	if vfs.IsExist(err) {
		t.Fatalf("Open %s : want non existing file, got %v", path, err)
	}

	return f
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

// TestAll runs all file systems tests.
func (sfs *SuiteFS) TestAll(t *testing.T) {
	sfs.TestRead(t)
	sfs.TestWrite(t)
	sfs.TestPath(t)
}

// TestWrite runs all file systems tests with write access.
func (sfs *SuiteFS) TestWrite(t *testing.T) {
	sfs.TestChmod(t)
	sfs.TestChown(t)
	sfs.TestChroot(t)
	sfs.TestChtimes(t)
	sfs.TestCreate(t)
	sfs.TestLchown(t)
	sfs.TestLink(t)
	sfs.TestMkdir(t)
	sfs.TestMkdirAll(t)
	sfs.TestOpenFileWrite(t)
	sfs.TestRemove(t)
	sfs.TestRemoveAll(t)
	sfs.TestRename(t)
	sfs.TestSameFile(t)
	sfs.TestSymlink(t)
	sfs.TestTempDir(t)
	sfs.TestTempFile(t)
	sfs.TestTruncate(t)
	sfs.TestWriteFile(t)
	sfs.TestWriteString(t)
	sfs.TestUmask(t)
	sfs.TestFileChmod(t)
	sfs.TestFileChown(t)
	sfs.TestFileSync(t)
	sfs.TestFileTruncate(t)
	sfs.TestFileWrite(t)
	sfs.TestFileWriteString(t)
	sfs.TestFileWriteTime(t)
	sfs.TestFileCloseWrite(t)
}

// TestRead runs all file systems tests with read access.
func (sfs *SuiteFS) TestRead(t *testing.T) {
	sfs.TestChdir(t)
	sfs.TestClone(t)
	sfs.TestEvalSymlink(t)
	sfs.TestGetTempDir(t)
	sfs.TestLstat(t)
	sfs.TestOpen(t)
	sfs.TestReadDir(t)
	sfs.TestReadFile(t)
	sfs.TestReadlink(t)
	sfs.TestStat(t)
	sfs.TestFileChdir(t)
	sfs.TestFileCloseRead(t)
	sfs.TestFileFd(t)
	sfs.TestFileName(t)
	sfs.TestFileRead(t)
	sfs.TestFileReadDir(t)
	sfs.TestFileReaddirnames(t)
	sfs.TestFileSeek(t)
	sfs.TestFileStat(t)
	sfs.TestStatT(t)
}

// Dir contains the data to test directories.
type Dir struct {
	Path      string
	Mode      os.FileMode
	WantModes []os.FileMode
}

// GetDirs returns the test directories used by Mkdir function.
func GetDirs() []*Dir {
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

// GetDirsAll returns the test directories used by MkdirAll function.
func GetDirsAll() []*Dir {
	dirs := []*Dir{
		{Path: "/H/6", Mode: 0o750, WantModes: []os.FileMode{0o750, 0o750}},
		{Path: "/H/6/I/7", Mode: 0o755, WantModes: []os.FileMode{0o750, 0o750, 0o755, 0o755}},
		{Path: "/H/6/I/7/J/8", Mode: 0o777, WantModes: []os.FileMode{0o750, 0o750, 0o755, 0o755, 0o777, 0o777}},
	}

	return dirs
}

// CreateDirs create test directories and baseDir directory if necessary.
func (sfs *SuiteFS) CreateDirs(t *testing.T, baseDir string) []*Dir {
	t.Helper()

	vfs := sfs.vfsSetup

	err := vfs.MkdirAll(baseDir, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("MkdirAll %s : want error to be nil, got %v", baseDir, err)
	}

	dirs := GetDirs()
	for _, dir := range dirs {
		path := vfs.Join(baseDir, dir.Path)

		err = vfs.Mkdir(path, dir.Mode)
		if err != nil {
			t.Fatalf("Mkdir %s : want error to be nil, got %v", path, err)
		}
	}

	return dirs
}

// File contains the data to test files.
type File struct {
	Path    string
	Mode    os.FileMode
	Content []byte
}

// GetFiles returns the test files.
func GetFiles() []*File {
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

// CreateFiles create test files.
func (sfs *SuiteFS) CreateFiles(t *testing.T, baseDir string) []*File {
	t.Helper()

	vfs := sfs.vfsSetup

	files := GetFiles()
	for _, file := range files {
		path := vfs.Join(baseDir, file.Path)

		err := vfs.WriteFile(path, file.Content, file.Mode)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
		}
	}

	return files
}

// Symlink contains the data to create symbolic links.
type Symlink struct {
	NewName string
	OldName string
}

// GetSymlinks returns the symbolic links to create.
func GetSymlinks(vfs avfs.Featurer) []*Symlink {
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

// SymlinkEval contains the data to evaluate symbolic links.
type SymlinkEval struct {
	NewName   string
	OldName   string
	WantErr   error
	IsSymlink bool
	Mode      os.FileMode
}

// GetSymlinksEval return the symbolic links to evaluate.
func GetSymlinksEval(vfs avfs.Featurer) []*SymlinkEval {
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

// CreateSymlinks creates test symbolic links.
func (sfs *SuiteFS) CreateSymlinks(t *testing.T, baseDir string) []*Symlink {
	t.Helper()

	vfs := sfs.vfsSetup

	symlinks := GetSymlinks(vfs)
	for _, sl := range symlinks {
		oldPath := vfs.Join(baseDir, sl.OldName)
		newPath := vfs.Join(baseDir, sl.NewName)

		err := vfs.Symlink(oldPath, newPath)
		if err != nil {
			t.Fatalf("TestSymlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
		}
	}

	return symlinks
}

// CheckPathError checks errors of type os.PathError.
func CheckPathError(t *testing.T, testName, wantOp, wantPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.PathError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.PathError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if wantOp != err.Op || wantPath != err.Path || (wantErr != err.Err && wantErr.Error() != err.Err.Error()) {
		wantPathErr := &os.PathError{Op: wantOp, Path: wantPath, Err: wantErr}
		t.Errorf("%s %s: want error to be %v, got %v", testName, wantPath, wantPathErr, gotErr)
	}
}

// CheckSyscallError checks errors of type os.SyscallError.
func CheckSyscallError(t *testing.T, testName, wantOp, wantPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.SyscallError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.SyscallError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if err.Syscall != wantOp || err.Err != wantErr {
		t.Errorf("%s %s : want error to be %v, got %v", testName, wantPath, wantErr, err)
	}
}

// CheckLinkError checks errors of type os.LinkError.
func CheckLinkError(t *testing.T, testName, wantOp, oldPath, newPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, newPath, wantErr)
	}

	err, ok := gotErr.(*os.LinkError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.LinkError,\n got %v", testName, newPath, reflect.TypeOf(gotErr))
	}

	if err.Op != wantOp || err.Err != wantErr {
		t.Errorf("%s %s %s : want error to be %v,\n got %v", testName, oldPath, newPath, wantErr, err)
	}
}

// CheckInvalid checks if the error in os.ErrInvalid.
func CheckInvalid(t *testing.T, funcName string, err error) {
	t.Helper()

	if err != os.ErrInvalid {
		t.Errorf("%s : want error to be %v, got %v", funcName, os.ErrInvalid, err)
	}
}

// CheckPanic checks that function f panics.
func CheckPanic(t *testing.T, funcName string, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s : want function to panic, not panicing", funcName)
		}
	}()

	f()
}
