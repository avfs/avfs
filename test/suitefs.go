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
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// SuiteFS is a test suite for virtual file systems.
type SuiteFS struct {
	// vfsRoot is the file system as a root user.
	vfsRoot avfs.VFS

	// vfsWrite is the file system as test user with read and write permissions.
	vfsWrite avfs.VFS

	// vfsRead is the file system as test user with read only permissions.
	vfsRead avfs.VFS

	// rootDir is the root directory for tests, it can be generated automatically or specified with WithRootDir().
	rootDir string

	// maxRace is the maximum number of concurrent goroutines used in race tests.
	maxRace int

	// canTestPerm indicates if permissions can be tested.
	canTestPerm bool

	// osType is the operating system of the filesystem te test.
	osType avfs.OSType

	// Groups contains the test groups created with the identity manager.
	Groups []avfs.GroupReader

	// Users contains the test users created with the identity manager.
	Users []avfs.UserReader
}

// Option defines the option function used for initializing SuiteFS.
type Option func(*SuiteFS)

// NewSuiteFS creates a new test suite for a file system.
func NewSuiteFS(tb testing.TB, vfsRoot avfs.VFS, opts ...Option) *SuiteFS {
	if vfsRoot == nil {
		tb.Fatal("New : want FsRoot to be set, got nil")
	}

	currentUser := vfsRoot.CurrentUser()
	canTestPerm := vfsRoot.HasFeature(avfs.FeatIdentityMgr) && currentUser.IsRoot()

	info := "Info vfs : type = " + vfsRoot.Type()
	if vfsRoot.Name() != "" {
		info += ", name = " + vfsRoot.Name()
	}

	tb.Log(info)

	sfs := &SuiteFS{
		vfsRoot:     vfsRoot,
		vfsWrite:    vfsRoot,
		vfsRead:     vfsRoot,
		rootDir:     "",
		maxRace:     1000,
		canTestPerm: canTestPerm,
		osType:      vfsutils.RunTimeOS(),
	}

	if canTestPerm {
		sfs.Groups = CreateGroups(tb, vfsRoot, "")
		sfs.Users = CreateUsers(tb, vfsRoot, "")

		_, err := vfsRoot.User(UsrTest)
		if err != nil {
			tb.Fatalf("User %s : want error to be nil, got %s", UsrTest, err)
		}
	}

	for _, opt := range opts {
		opt(sfs)
	}

	return sfs
}

// Options

// WithVFSRead returns an option function which sets the VFS for read access.
func WithVFSRead(vfsRead avfs.VFS) Option {
	return func(sfs *SuiteFS) {
		sfs.vfsRead = vfsRead
	}
}

// WithVFSWrite returns an option function which sets the VFS for write access.
func WithVFSWrite(vfsWrite avfs.VFS) Option {
	return func(sfs *SuiteFS) {
		sfs.vfsWrite = vfsWrite
	}
}

// WithOs returns an option function which sets the operating system for the tests.
func WithOs(osType avfs.OSType) Option {
	return func(sfs *SuiteFS) {
		sfs.osType = osType
	}
}

// CreateDir  creates a directory and returns the path to this directory.
func (sfs *SuiteFS) CreateDir(t *testing.T) string {
	t.Helper()

	vfs := sfs.vfsWrite

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

// CreateFile  creates a file and returns the file name.
func (sfs *SuiteFS) CreateFile(t *testing.T, content []byte) string {
	t.Helper()

	vfs := sfs.vfsWrite

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
	vfs, _ := sfs.VFSAsUser(tb, userName)

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
		vfs, _ := sfs.VFSAsUser(tb, avfs.UsrRoot)

		err = vfs.RemoveAll(rootDir)
		if err != nil && sfs.osType != avfs.OsWindows {
			tb.Logf("RemoveAll : want error to be nil, got %s", err)
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

	vfs := sfs.vfsWrite

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

// OpenNonExistingFile open a non existing file and returns an avfs.File.
func (sfs *SuiteFS) OpenNonExistingFile(t *testing.T) avfs.File {
	t.Helper()

	vfs := sfs.vfsRead
	name := vfs.Join(sfs.rootDir, "nonExistingFile")

	f, err := vfs.Open(name)
	if vfs.IsExist(err) {
		t.Fatalf("Open %s : want non existing file, got file exists", name)
	}

	return f
}

// OSType returns the operating system, real or simulated.
func (sfs *SuiteFS) OSType() avfs.OSType {
	return sfs.osType
}

// VFSAsUser sets the test user to userName.
func (sfs *SuiteFS) VFSAsUser(tb testing.TB, name string) (avfs.VFS, avfs.UserReader) {
	vfs := sfs.vfsRoot

	u := vfs.CurrentUser()
	if !sfs.canTestPerm || u.Name() == name {
		return vfs, u
	}

	u, err := vfs.User(name)
	if err != nil {
		tb.Fatalf("User %s : want error to be nil, got %s", name, err)
	}

	return vfs, u
}

// VFSRoot return the root file system.
func (sfs *SuiteFS) VFSRoot() avfs.VFS {
	return sfs.vfsRoot
}

// VFSRead returns the file system for read only functions.
func (sfs *SuiteFS) VFSRead() avfs.VFS {
	return sfs.vfsRead
}

// VFSWrite returns the file system for read and write functions.
func (sfs *SuiteFS) VFSWrite() avfs.VFS {
	return sfs.vfsWrite
}

// All runs all file systems tests.
func (sfs *SuiteFS) All(t *testing.T) {
	sfs.Read(t)
	sfs.Write(t)
	sfs.Path(t)
}

// Write runs all file systems tests with write access.
func (sfs *SuiteFS) Write(t *testing.T) {
	sfs.Create(t)
	sfs.Chtimes(t)
	sfs.Link(t)
	sfs.Mkdir(t)
	sfs.MkdirAll(t)
	sfs.OpenFileWrite(t)
	sfs.Remove(t)
	sfs.RemoveAll(t)
	sfs.Rename(t)
	sfs.SameFile(t)
	sfs.Symlink(t)
	sfs.TempDir(t)
	sfs.TempFile(t)
	sfs.Truncate(t)
	sfs.WriteFile(t)
	sfs.WriteString(t)
	sfs.Umask(t)
	sfs.FileChmod(t)
	sfs.FileChown(t)
	sfs.FileSync(t)
	sfs.FileTruncate(t)
	sfs.FileWrite(t)
	sfs.FileWriteString(t)
	sfs.FileWriteTime(t)
	sfs.FileCloseWrite(t)
}

// Read runs all file systems tests with read access.
func (sfs *SuiteFS) Read(t *testing.T) {
	sfs.Chdir(t)
	sfs.Clone(t)
	sfs.EvalSymlink(t)
	sfs.GetTempDir(t)
	sfs.Glob(t)
	sfs.Lstat(t)
	sfs.Open(t)
	sfs.ReadDir(t)
	sfs.ReadFile(t)
	sfs.Readlink(t)
	sfs.Stat(t)
	sfs.FileChdir(t)
	sfs.FileCloseRead(t)
	sfs.FileFd(t)
	sfs.FileFuncOnClosedFile(t)
	sfs.FileName(t)
	sfs.FileRead(t)
	sfs.FileReadDir(t)
	sfs.FileReaddirnames(t)
	sfs.FileSeek(t)
	sfs.FileStat(t)
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
func CreateDirs(t *testing.T, vfs avfs.VFS, baseDir string) []*Dir {
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
func CreateFiles(t *testing.T, vfs avfs.VFS, baseDir string) []*File {
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
func CreateSymlinks(t *testing.T, vfs avfs.VFS, baseDir string) []*Symlink {
	symlinks := GetSymlinks(vfs)
	for _, sl := range symlinks {
		oldPath := vfs.Join(baseDir, sl.OldName)
		newPath := vfs.Join(baseDir, sl.NewName)

		err := vfs.Symlink(oldPath, newPath)
		if err != nil {
			t.Fatalf("Symlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
		}
	}

	return symlinks
}
