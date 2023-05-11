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
	Groups      []avfs.GroupReader // Groups contains the test groups created with the identity manager.
	Users       []avfs.UserReader  // Users contains the test users created with the identity manager.
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

	canTestPerm := vfs.OSType() != avfs.OsWindows &&
		vfs.HasFeature(avfs.FeatIdentityMgr) &&
		initUser.IsAdmin()

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
	sfs.Groups = CreateGroups(tb, idm, "")
	sfs.Users = CreateUsers(tb, idm, "")

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

// createRootDir creates tests root directory.
func (sfs *SuiteFS) createRootDir(tb testing.TB) {
	vfs := sfs.vfsSetup
	rootDir := ""

	if _, ok := tb.(*testing.B); ok {
		// run Benches on real disks, /tmp is usually an in memory file system.
		rootDir = vfs.Join(vfs.HomeDirUser(vfs.User()), "tmp")
		sfs.createDir(tb, rootDir, avfs.DefaultDirPerm)
	}

	rootDir, err := vfs.MkdirTemp(rootDir, "avfs")
	if err != nil {
		tb.Fatalf("MkdirTemp %s : want error to be nil, got %s", rootDir, err)
	}

	// Make rootDir accessible by anyone.
	err = vfs.Chmod(rootDir, avfs.DefaultDirPerm)
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
}

// setUser sets the test user to userName.
func (sfs *SuiteFS) setUser(tb testing.TB, userName string) avfs.UserReader {
	vfs := sfs.vfsSetup

	u := vfs.User()
	if !sfs.canTestPerm || u.Name() == userName {
		return u
	}

	u, err := vfs.SetUser(userName)
	if err != nil {
		tb.Fatalf("setUser %s : want error to be nil, got %s", userName, err)
	}

	return u
}

// VFSTest returns the file system used to run the tests.
func (sfs *SuiteFS) VFSTest() avfs.VFSBase {
	return sfs.vfsTest
}

// VFSSetup returns the file system used to set up the tests.
func (sfs *SuiteFS) VFSSetup() avfs.VFSBase {
	return sfs.vfsSetup
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

// createDir creates a directory for the tests.
func (sfs *SuiteFS) createDir(tb testing.TB, dirName string, mode fs.FileMode) {
	vfs := sfs.vfsSetup

	err := vfs.MkdirAll(dirName, mode)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %s", dirName, err)
	}

	err = vfs.Chmod(dirName, mode)
	if err != nil {
		tb.Fatalf("Chmod %s : want error to be nil, got %s", dirName, err)
	}
}

// createFile creates an empty file for the tests.
func (sfs *SuiteFS) createFile(tb testing.TB, fileName string, mode fs.FileMode) {
	vfs := sfs.vfsSetup

	err := vfs.WriteFile(fileName, nil, mode)
	if err != nil {
		tb.Fatalf("WriteFile %s : want error to be nil, got %s", fileName, err)
	}
}

// changeDir changes the current directory for the tests.
func (sfs *SuiteFS) changeDir(tb testing.TB, dir string) {
	vfs := sfs.vfsTest

	err := vfs.Chdir(dir)
	if err != nil {
		tb.Fatalf("Chdir %s : want error to be nil, got %s", dir, err)
	}
}

// removeDir removes all files under testDir.
func (sfs *SuiteFS) removeDir(tb testing.TB, testDir string) {
	vfs := sfs.vfsSetup

	err := vfs.Chdir(sfs.rootDir)
	if err != nil {
		tb.Fatalf("Chdir %s : want error to be nil, got %v", sfs.rootDir, err)
	}

	// RemoveAll() should be executed as the user who started the tests, generally root,
	// to clean up files with different permissions.
	sfs.setUser(tb, sfs.initUser.Name())

	err = vfs.RemoveAll(testDir)
	if err != nil && avfs.CurrentOSType() != avfs.OsWindows {
		tb.Fatalf("RemoveAll %s : want error to be nil, got %v", testDir, err)
	}
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
		sfs.TestSetUser,
		sfs.TestVolume,
		sfs.TestWriteOnReadOnlyFS,
	)
}

// BenchAll runs all benchmarks.
func (sfs *SuiteFS) BenchAll(b *testing.B) {
	sfs.RunBenchmarks(b, UsrTest,
		sfs.BenchCreate,
		sfs.BenchFileReadWrite,
		sfs.BenchMkdir,
		sfs.BenchOpenFile,
		sfs.BenchRemove,
	)
}

// closedFile returns a closed avfs.File.
func (sfs *SuiteFS) closedFile(tb testing.TB, testDir string) (f avfs.File, fileName string) {
	fileName = sfs.emptyFile(tb, testDir)

	vfs := sfs.vfsTest

	f, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		tb.Fatalf("Create %s : want error to be nil, got %v", fileName, err)
	}

	err = f.Close()
	if err != nil {
		tb.Fatalf("Close %s : want error to be nil, got %v", fileName, err)
	}

	return f, fileName
}

// emptyFile returns an empty file name.
func (sfs *SuiteFS) emptyFile(tb testing.TB, testDir string) string {
	const emptyFile = "emptyFile"

	vfs := sfs.vfsSetup
	fileName := vfs.Join(testDir, emptyFile)

	_, err := vfs.Stat(fileName)
	if errors.Is(err, fs.ErrNotExist) {
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

// existingDir returns an existing directory.
func (sfs *SuiteFS) existingDir(tb testing.TB, testDir string) string {
	vfs := sfs.vfsSetup

	dirName, err := vfs.MkdirTemp(testDir, "existingDir")
	if err != nil {
		tb.Fatalf("MkdirTemp %s : want error to be nil, got %v", dirName, err)
	}

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
type Dir struct {
	Path      string
	Mode      fs.FileMode
	WantModes []fs.FileMode
}

// sampleDirs returns the sample directories used by Mkdir function.
func (sfs *SuiteFS) sampleDirs(testDir string) []*Dir {
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

	for i, dir := range dirs {
		dirs[i].Path = sfs.vfsTest.Join(testDir, dir.Path)
	}

	return dirs
}

// sampleDirsAll returns the sample directories used by MkdirAll function.
func (sfs *SuiteFS) sampleDirsAll(testDir string) []*Dir {
	dirs := []*Dir{
		{Path: "/H/6", Mode: 0o750, WantModes: []fs.FileMode{0o750, 0o750}},
		{Path: "/H/6/I/7", Mode: 0o755, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755}},
		{Path: "/H/6/I/7/J/8", Mode: 0o777, WantModes: []fs.FileMode{0o750, 0o750, 0o755, 0o755, 0o777, 0o777}},
	}

	for i, dir := range dirs {
		dirs[i].Path = sfs.vfsTest.Join(testDir, dir.Path)
	}

	return dirs
}

// createSampleDirs creates and returns sample directories and testDir directory if necessary.
func (sfs *SuiteFS) createSampleDirs(tb testing.TB, testDir string) []*Dir {
	vfs := sfs.vfsSetup

	err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
	if err != nil {
		tb.Fatalf("MkdirAll %s : want error to be nil, got %v", testDir, err)
	}

	dirs := sfs.sampleDirs(testDir)
	for _, dir := range dirs {
		err = vfs.Mkdir(dir.Path, dir.Mode)
		if err != nil {
			tb.Fatalf("Mkdir %s : want error to be nil, got %v", dir.Path, err)
		}
	}

	return dirs
}

// File contains the sample files.
type File struct {
	Path    string
	Mode    fs.FileMode
	Content []byte
}

// sampleFiles returns the sample files.
func (sfs *SuiteFS) sampleFiles(testDir string) []*File {
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

	for i, file := range files {
		files[i].Path = sfs.vfsTest.Join(testDir, file.Path)
	}

	return files
}

// createSampleFiles creates and returns the sample files.
func (sfs *SuiteFS) createSampleFiles(tb testing.TB, testDir string) []*File {
	vfs := sfs.vfsSetup

	files := sfs.sampleFiles(testDir)
	for _, file := range files {
		err := vfs.WriteFile(file.Path, file.Content, file.Mode)
		if err != nil {
			tb.Fatalf("WriteFile %s : want error to be nil, got %v", file.Path, err)
		}
	}

	return files
}

// Symlink contains sample symbolic links.
type Symlink struct {
	NewPath string
	OldPath string
}

// sampleSymlinks returns the sample symbolic links.
func (sfs *SuiteFS) sampleSymlinks(testDir string) []*Symlink {
	vfs := sfs.vfsTest
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sls := []*Symlink{
		{NewPath: "/A/lroot", OldPath: "/"},
		{NewPath: "/lC", OldPath: "/C"},
		{NewPath: "/B/1/lafile2.txt", OldPath: "/A/afile2.txt"},
		{NewPath: "/B/2/lf", OldPath: "/B/2/F"},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/lf"},
		{NewPath: "/C/lllf", OldPath: "/B/2/F/3/llf"},
		{NewPath: "/A/l3file2.txt", OldPath: "/C/lllf/3/3file2.txt"},
		{NewPath: "/C/lNonExist", OldPath: "/A/path/to/a/non/existing/file"},
	}

	for i, sl := range sls {
		sls[i].NewPath = vfs.Join(testDir, sl.NewPath)
		sls[i].OldPath = vfs.Join(testDir, sl.OldPath)
	}

	return sls
}

// SymlinkEval contains the data to evaluate the sample symbolic links.
type SymlinkEval struct {
	NewPath   string      // Name of the symbolic link.
	OldPath   string      // Value of the symbolic link.
	IsSymlink bool        //
	Mode      fs.FileMode //
}

// sampleSymlinksEval returns the sample symbolic links to evaluate.
func (sfs *SuiteFS) sampleSymlinksEval(testDir string) []*SymlinkEval {
	vfs := sfs.vfsTest
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	sles := []*SymlinkEval{
		{NewPath: "/A/lroot/", OldPath: "/", IsSymlink: true, Mode: fs.ModeDir | 0o777},
		{NewPath: "/A/lroot/B", OldPath: "/B", IsSymlink: false, Mode: fs.ModeDir | 0o755},
		{NewPath: "/", OldPath: "/", IsSymlink: false, Mode: fs.ModeDir | 0o777},
		{NewPath: "/lC", OldPath: "/C", IsSymlink: true, Mode: fs.ModeDir | 0o750},
		{NewPath: "/B/1/lafile2.txt", OldPath: "/A/afile2.txt", IsSymlink: true, Mode: 0o644},
		{NewPath: "/B/2/lf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/C/lllf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/B/2/F/3/llf", OldPath: "/B/2/F", IsSymlink: true, Mode: fs.ModeDir | 0o755},
		{NewPath: "/C/lllf/3/3file1.txt", OldPath: "/B/2/F/3/3file1.txt", IsSymlink: false, Mode: 0o640},
	}

	for i, sle := range sles {
		sles[i].NewPath = vfs.Join(testDir, sle.NewPath)
		sles[i].OldPath = vfs.Join(testDir, sle.OldPath)
	}

	return sles
}

// CreateSampleSymlinks creates the sample symbolic links.
func (sfs *SuiteFS) CreateSampleSymlinks(tb testing.TB, testDir string) []*Symlink {
	vfs := sfs.vfsSetup

	symlinks := sfs.sampleSymlinks(testDir)
	for _, sl := range symlinks {
		err := vfs.Symlink(sl.OldPath, sl.NewPath)
		if err != nil {
			tb.Fatalf("TestSymlink %s %s : want error to be nil, got %v", sl.OldPath, sl.NewPath, err)
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
	tb   testing.TB
	err  error
	op   string
	path string
}

// CheckPathError checks if err is a fs.PathError.
func CheckPathError(tb testing.TB, err error) *checkPathError {
	tb.Helper()

	if err == nil {
		tb.Error("want error to be not nil, got nil")

		return &checkPathError{tb: tb, op: "", path: "", err: nil}
	}

	e, ok := err.(*fs.PathError)
	if !ok {
		tb.Errorf("want error type to be *fs.PathError, got %v : %v", reflect.TypeOf(err), err)

		return &checkPathError{tb: tb, op: "", path: "", err: nil}
	}

	return &checkPathError{tb: tb, op: e.Op, path: e.Path, err: e.Err}
}

// Op checks if op is equal to the current fs.PathError Op for one of the osTypes.
func (cp *checkPathError) Op(op string, osTypes ...avfs.OSType) *checkPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true
		}
	}

	if !canTest {
		return cp
	}

	if cp.op != op {
		tb.Errorf("want Op to be %s, got %s", op, cp.op)
	}

	return cp
}

// OpStat checks if the current fs.PathError Op is a Stat Op.
func (cp *checkPathError) OpStat() *checkPathError {
	tb := cp.tb
	tb.Helper()

	return cp.
		Op("stat", avfs.OsLinux).
		Op("createFile", avfs.OsWindows)
}

// OpLstat checks if the current fs.PathError Op is a Lstat Op.
func (cp *checkPathError) OpLstat() *checkPathError {
	tb := cp.tb
	tb.Helper()

	return cp.
		Op("lstat", avfs.OsLinux).
		Op("createFile", avfs.OsWindows)
}

// Path checks the path of the current fs.PathError.
func (cp *checkPathError) Path(path string) *checkPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	if cp.path != path {
		tb.Errorf("want Path to be %s, got %s", path, cp.path)
	}

	return cp
}

// Err checks the error of current fs.PathError.
func (cp *checkPathError) Err(wantErr error, osTypes ...avfs.OSType) *checkPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cp
	}

	e := reflect.ValueOf(cp.err)
	we := reflect.ValueOf(wantErr)

	if e.CanUint() && we.CanUint() && e.Uint() == we.Uint() {
		return cp
	}

	if cp.err.Error() == wantErr.Error() {
		return cp
	}

	tb.Errorf("%s : want error to be %v, got %v", cp.path, wantErr, cp.err)

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
	tb  testing.TB
	err error
	op  string
	old string
	new string
}

// CheckLinkError checks if err is a os.LinkError.
func CheckLinkError(tb testing.TB, err error) *checkLinkError {
	tb.Helper()

	if err == nil {
		tb.Error("want error to be not nil")

		return &checkLinkError{tb: tb, op: "", old: "", new: "", err: nil}
	}

	e, ok := err.(*os.LinkError)
	if !ok {
		tb.Errorf("want error type to be *os.LinkError, got %v : %v", reflect.TypeOf(err), err)

		return &checkLinkError{tb: tb, op: "", old: "", new: "", err: nil}
	}

	return &checkLinkError{tb: tb, op: e.Op, old: e.Old, new: e.New, err: e.Err}
}

// Op checks if wantOp is equal to the current os.LinkError Op.
func (cl *checkLinkError) Op(wantOp string, osTypes ...avfs.OSType) *checkLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cl
	}

	if cl.op != wantOp {
		tb.Errorf("want Op to be %s, got %s", wantOp, cl.op)
	}

	return cl
}

// Old checks the old path of the current os.LinkError.
func (cl *checkLinkError) Old(wantOld string) *checkLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	if cl.old != wantOld {
		tb.Errorf("want old path to be %s, got %s", wantOld, cl.old)
	}

	return cl
}

// New checks the new path of the current os.LinkError.
func (cl *checkLinkError) New(wantNew string) *checkLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	if cl.new != wantNew {
		tb.Errorf("want new path to be %s, got %s", wantNew, cl.new)
	}

	return cl
}

// Err checks the error of current os.LinkError.
func (cl *checkLinkError) Err(wantErr error, osTypes ...avfs.OSType) *checkLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cl
	}

	e := reflect.ValueOf(cl.err)
	we := reflect.ValueOf(wantErr)

	if e.CanUint() && we.CanUint() && e.Uint() == we.Uint() {
		return cl
	}

	if cl.err.Error() == wantErr.Error() {
		return cl
	}

	tb.Errorf("%s %s : want error to be %v, got %v", cl.old, cl.new, wantErr, cl.err)

	return cl
}

// ErrPermDenied checks if the current os.LinkError is a permission denied error.
func (cl *checkLinkError) ErrPermDenied() *checkLinkError {
	cl.tb.Helper()

	return cl.
		Err(avfs.ErrPermDenied, avfs.OsLinux).
		Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
}
