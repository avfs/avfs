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
	"io"
	"math"
	"os"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
)

// DirFuncOnFile tests directory functions on files.
func (sfs *SuiteFS) DirFuncOnFile(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	existingFile := vfs.Join(rootDir, "existingFile")

	err := vfs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	vfs = sfs.GetFsRead()

	f, err := vfs.Open(existingFile)
	if err != nil {
		t.Fatalf("Create : want error to be nil, got %v", err)
	}

	defer f.Close()

	t.Run("DirFuncOnFileF", func(t *testing.T) {
		_, err = f.Readdirnames(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", f.Name(), avfs.ErrNotADirectory, err)
		default:
			if vfs.CurrentUser().IsRoot() {
				CheckSyscallError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			} else {
				CheckPathError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			}
		}

		err = f.Chdir()

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrNotADirectory, err)
		}
	})
}

// FileFuncOnDir test file functions on directories.
func (sfs *SuiteFS) FileFuncOnDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	pathDir := vfs.Join(rootDir, "existingDir")

	err := vfs.Mkdir(pathDir, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir : want error to be nil, got %v", err)
	}

	vfs = sfs.GetFsRead()

	t.Run("FileFuncOnDir", func(t *testing.T) {
		f, err := vfs.Open(pathDir)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Read(b)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Read", "read", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Read", "read", pathDir, avfs.ErrIsADirectory, err)
		}

		_, err = f.ReadAt(b, 0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadAt", "read", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "ReadAt", "read", pathDir, avfs.ErrIsADirectory, err)
		}

		_, err = f.Seek(0, io.SeekStart)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}
		}

		if vfs.HasFeature(avfs.FeatReadOnly) {
			return
		}

		err = f.Truncate(0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Truncate", "truncate", pathDir, os.ErrInvalid, err)
		}

		_, err = f.Write(b)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Write", "write", pathDir, avfs.ErrBadFileDesc, err)
		}

		_, err = f.WriteAt(b, 0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "WriteAt", "write", pathDir, avfs.ErrBadFileDesc, err)
		}
	})
}

// FileFuncOnClosed tests functions on closed files.
func (sfs *SuiteFS) FileFuncOnClosed(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	existingFile := vfs.Join(rootDir, "existingFile")

	err := vfs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	vfs = sfs.GetFsRead()

	t.Run("FuncOnFileClosed", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Close : want error to be nil, got %v", err)
		}

		b := make([]byte, 1)

		err = f.Close()
		CheckPathError(t, "Close", "close", existingFile, os.ErrClosed, err)

		fd := f.Fd()
		if fd != math.MaxUint64 {
			t.Errorf("Fd %s : want Fd to be %d, got %d", existingFile, uint64(math.MaxUint64), fd)
		}

		name := f.Name()
		if name != existingFile {
			t.Errorf("Name %s : want Name to be %s, got %s", existingFile, existingFile, name)
		}

		_, err = f.Read(b)
		CheckPathError(t, "Read", "read", existingFile, os.ErrClosed, err)

		_, err = f.ReadAt(b, 0)
		CheckPathError(t, "ReadAt", "read", existingFile, os.ErrClosed, err)

		_, err = f.Seek(0, io.SeekStart)
		CheckPathError(t, "Seek", "seek", existingFile, os.ErrClosed, err)

		_, err = f.Readdir(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdir", "Readdir", existingFile, avfs.ErrWinPathNotFound, err)
		default:
			if vfs.CurrentUser().IsRoot() {
				if err == nil || err.Error() != avfs.ErrFileClosing.Error() {
					t.Errorf("Readdir %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
				}
			} else {
				CheckPathError(t, "Readdir", "readdirent", existingFile, avfs.ErrFileClosing, err)
			}
		}

		_, err = f.Readdirnames(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", existingFile, avfs.ErrWinPathNotFound, err)
		default:
			if vfs.CurrentUser().IsRoot() {
				if err == nil || err.Error() != avfs.ErrFileClosing.Error() {
					t.Errorf("Readdirnames %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
				}
			} else {
				CheckPathError(t, "Readdirnames", "readdirent", existingFile, avfs.ErrFileClosing, err)
			}
		}

		_, err = f.Stat()
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "GetFileType", existingFile, avfs.ErrFileClosing, err)
		default:
			CheckPathError(t, "Stat", "stat", existingFile, avfs.ErrFileClosing, err)
		}

		err = f.Sync()
		CheckPathError(t, "Sync", "sync", existingFile, os.ErrClosed, err)

		if vfs.HasFeature(avfs.FeatReadOnly) {
			return
		}

		err = f.Chdir()
		CheckPathError(t, "Chdir", "chdir", existingFile, os.ErrClosed, err)

		err = f.Chmod(avfs.DefaultFilePerm)
		CheckPathError(t, "Chmod", "chmod", existingFile, os.ErrClosed, err)

		if vfs.HasFeature(avfs.FeatIdentityMgr) {
			err = f.Chown(0, 0)
			CheckPathError(t, "Chown", "chown", existingFile, os.ErrClosed, err)
		}

		err = f.Sync()
		CheckPathError(t, "Sync", "sync", existingFile, os.ErrClosed, err)

		err = f.Truncate(0)
		CheckPathError(t, "Truncate", "truncate", existingFile, os.ErrClosed, err)

		_, err = f.Write(b)
		CheckPathError(t, "Write", "write", existingFile, os.ErrClosed, err)

		_, err = f.WriteAt(b, 0)
		CheckPathError(t, "WriteAt", "write", existingFile, os.ErrClosed, err)
	})
}

// NotImplemented tests non implemented functions.
func (sfs *SuiteFS) NotImplemented(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

	var (
		oldName = ""
		newName = ""
		newPath = ""
	)

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Chdir(rootDir)
		CheckPathError(t, "Chdir", "chdir", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Chmod(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Chmod", "chmod", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Chtimes(rootDir, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", rootDir, avfs.ErrPermDenied, err)

		fsC, ok := vfs.(avfs.Cloner)
		if ok {
			fsClone := fsC.Clone()
			if fsClone != vfs {
				t.Errorf("Clone : want cloned vfs to be equal to vfs, got different")
			}
		}

		_, err = vfs.Create(rootDir)
		CheckPathError(t, "Create", "open", rootDir, avfs.ErrPermDenied, err)

		u := vfs.CurrentUser()
		if u != dummyidm.NotImplementedUser {
			t.Errorf("CurrentUser : want User to be nil, got %v", u)
		}

		if um := vfs.GetUMask(); um != 0 {
			t.Errorf("GetUMask : want umask to be 0, got %d", um)
		}

		_, err = vfs.Getwd()
		CheckPathError(t, "Getwd", "getwd", "", avfs.ErrPermDenied, err)

		_, err = vfs.Glob("")
		if err != nil {
			t.Errorf("Glob : want error to be nil, got %v", err)
		}

		_, err = vfs.Lstat(rootDir)
		CheckPathError(t, "Lstat", "lstat", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Mkdir(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", rootDir, avfs.ErrPermDenied, err)

		err = vfs.MkdirAll(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", rootDir, avfs.ErrPermDenied, err)

		name := vfs.Name()
		if name != avfs.NotImplemented {
			t.Errorf("Name : want name to be %s, got %s", avfs.NotImplemented, name)
		}

		_, err = vfs.Open(rootDir)
		CheckPathError(t, "Open", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.OpenFile(rootDir, os.O_RDONLY, avfs.DefaultFilePerm)
		CheckPathError(t, "OpenFile", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.ReadDir(rootDir)
		CheckPathError(t, "ReadDir", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.ReadFile(rootDir)
		CheckPathError(t, "ReadFile", "open", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Remove(rootDir)
		CheckPathError(t, "Remove", "remove", rootDir, avfs.ErrPermDenied, err)

		err = vfs.RemoveAll(rootDir)
		CheckPathError(t, "RemoveAll", "removeall", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Rename(rootDir, newPath)
		CheckLinkError(t, "Rename", "rename", rootDir, newPath, avfs.ErrPermDenied, err)

		if vfs.SameFile(nil, nil) {
			t.Errorf("SameFile : want SameFile to be false, got true")
		}

		_, err = vfs.Stat(rootDir)
		CheckPathError(t, "Stat", "stat", rootDir, avfs.ErrPermDenied, err)

		tmp := vfs.GetTempDir()
		if tmp != avfs.TmpDir {
			t.Errorf("GetTempDir : want error to be %v, got %v", avfs.NotImplemented, tmp)
		}

		_, err = vfs.TempDir(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempDir : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = vfs.TempFile(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempFile : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = vfs.Truncate(rootDir, 0)
		CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrPermDenied, err)

		vfs.UMask(0)

		_, err = vfs.User(UsrTest)
		if err != avfs.ErrPermDenied {
			t.Errorf("User : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		walkFunc := func(rootDir string, info os.FileInfo, err error) error { return nil }

		err = vfs.Walk(rootDir, walkFunc)
		if err != nil {
			t.Errorf("User : want error to be nil, got %v", err)
		}

		err = vfs.WriteFile(rootDir, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, "WriteFile", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.Abs(rootDir)
		if err != nil {
			t.Errorf("Name : want error to be nil, got %v", err)
		}

		base := vfs.Base(rootDir)
		if base != rootDir[1:] {
			t.Errorf("Base : want Base to be %s, got %s", rootDir[1:], base)
		}

		clean := vfs.Clean(rootDir)
		if clean != rootDir {
			t.Errorf("Clean : want Clean to be %s, got %s", rootDir, clean)
		}

		dir := vfs.Dir(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Dir : want Dir to be %s, got %s", string(avfs.PathSeparator), dir)
		}

		b := vfs.IsAbs(rootDir)
		if !b {
			t.Errorf("IsAbs : want result to be true, got false")
		}

		b = vfs.IsPathSeparator(avfs.PathSeparator)
		if !b {
			t.Errorf("IsPathSeparator : want result to be true, got false")
		}

		join := vfs.Join(rootDir, rootDir)
		if join != rootDir+rootDir {
			t.Errorf("Join : want join to be %s, got %s", rootDir+rootDir, join)
		}

		_, err = vfs.Rel(rootDir, rootDir)
		if err != nil {
			t.Errorf("Rel : want error to be nil, got %v", err)
		}

		dir, _ = vfs.Split(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Split : want dir to be %c, got %s", avfs.PathSeparator, dir)
		}
	}

	if !vfs.HasFeature(avfs.FeatHardlink) {
		err := vfs.Link(oldName, newName)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckLinkError(t, "Link", "link", oldName, newName, avfs.ErrWinPathNotFound, err)
		default:
			CheckLinkError(t, "Link", "link", oldName, newName, avfs.ErrPermDenied, err)
		}
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		err := vfs.Chown(rootDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrPermDenied, err)
		}

		err = vfs.Lchown(rootDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrPermDenied, err)
		}

		f, err := vfs.Open(rootDir)
		if err == nil {
			// We just need a file handle, be it valid or not, however if the handle is valid, it must be closed.
			defer f.Close()
		}

		err = f.Chown(0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrPermDenied, err)
		}
	}

	if !vfs.HasFeature(avfs.FeatSymlink) && vfs.OSType() != avfs.OsWindows {
		_, err := vfs.EvalSymlinks(rootDir)
		CheckPathError(t, "EvalSymlinks", "lstat", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.Readlink(rootDir)
		CheckPathError(t, "Readlink", "readlink", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Symlink(rootDir, newPath)
		CheckLinkError(t, "Symlink", "symlink", rootDir, newPath, avfs.ErrPermDenied, err)
	}

	if !vfs.HasFeature(avfs.FeatChroot) {
		err := vfs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)
	}

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f, _ := vfs.Open(rootDir)

		err := f.Chdir()
		CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrPermDenied, err)

		err = f.Chmod(0)
		CheckPathError(t, "Chmod", "chmod", f.Name(), avfs.ErrPermDenied, err)

		err = f.Close()
		CheckPathError(t, "Close", "close", f.Name(), avfs.ErrPermDenied, err)

		fd := f.Fd()
		if fd != 0 {
			t.Errorf("Fd : want Fd to be 0, got %v", fd)
		}

		n := f.Name()
		if n != avfs.NotImplemented {
			t.Errorf("Name : want error to be %v, got %v", avfs.NotImplemented, n)
		}

		_, err = f.Read([]byte{})
		CheckPathError(t, "Read", "read", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.ReadAt([]byte{}, 0)
		CheckPathError(t, "ReadAt", "read", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Readdir(0)
		CheckPathError(t, "Readdir", "readdirent", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Readdirnames(0)
		CheckPathError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Seek(0, io.SeekStart)
		CheckPathError(t, "Seek", "seek", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Stat()
		CheckPathError(t, "Stat", "stat", f.Name(), avfs.ErrPermDenied, err)

		err = f.Sync()
		CheckPathError(t, "Sync", "sync", f.Name(), avfs.ErrPermDenied, err)

		err = f.Truncate(0)
		CheckPathError(t, "Truncate", "truncate", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Write([]byte{})
		CheckPathError(t, "Write", "write", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, "WriteAt", "write", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteString("")
		CheckPathError(t, "WriteString", "write", f.Name(), avfs.ErrPermDenied, err)
	}
}

// SuiteNilPtrFile test calls to File methods when f is nil.
func SuiteNilPtrFile(t *testing.T, f avfs.File) {
	err := f.Chdir()
	CheckInvalid(t, "Chdir", err)

	err = f.Chmod(0)
	CheckInvalid(t, "Chmod", err)

	err = f.Chown(0, 0)
	CheckInvalid(t, "Chown", err)

	err = f.Close()
	CheckInvalid(t, "Close", err)

	CheckPanic(t, "f.Name()", func() { _ = f.Name() })

	fd := f.Fd()
	if fd != math.MaxUint64 {
		t.Errorf("Fd : want fd to be %d, got %d", 0, fd)
	}

	_, err = f.Read([]byte{})
	CheckInvalid(t, "Read", err)

	_, err = f.ReadAt([]byte{}, 0)
	CheckInvalid(t, "ReadAt", err)

	_, err = f.Readdir(0)
	CheckInvalid(t, "Readdir", err)

	_, err = f.Readdirnames(0)
	CheckInvalid(t, "Readdirnames", err)

	_, err = f.Seek(0, io.SeekStart)
	CheckInvalid(t, "Seek", err)

	_, err = f.Stat()
	CheckInvalid(t, "Stat", err)

	err = f.Sync()
	CheckInvalid(t, "Sync", err)

	err = f.Truncate(0)
	CheckInvalid(t, "Truncate", err)

	_, err = f.Write([]byte{})
	CheckInvalid(t, "Write", err)

	_, err = f.WriteAt([]byte{}, 0)
	CheckInvalid(t, "WriteAt", err)

	_, err = f.WriteString("")
	CheckInvalid(t, "WriteString", err)
}
