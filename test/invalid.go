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

// FuncNonExistingFile tests functions on non existing files.
func (sfs *SuiteFs) FuncNonExistingFile() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	existingFile := fs.Join(rootDir, "existingFile")

	err := fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	nonExistingFile := fs.Join(rootDir, "nonExistingFile")
	buf := make([]byte, 1)

	fs = sfs.GetFsRead()

	t.Run("FsNonExistingFile", func(t *testing.T) {
		err := fs.Chmod(nonExistingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Chmod", "chmod", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)

		if fs.HasFeature(avfs.FeatIdentityMgr) {
			err = fs.Chown(nonExistingFile, 0, 0)
			CheckPathError(t, "Chown", "chown", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		err = fs.Chtimes(nonExistingFile, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)

		if fs.HasFeature(avfs.FeatHardlink) {
			err = fs.Link(nonExistingFile, existingFile)
			CheckLinkError(t, "Link", "link", nonExistingFile, nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		if fs.HasFeature(avfs.FeatIdentityMgr) && fs.HasFeature(avfs.FeatSymlink) {
			err = fs.Lchown(nonExistingFile, 0, 0)
			CheckPathError(t, "Lchown", "lchown", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		_, err = fs.Lstat(nonExistingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Lstat", "CreateFile", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		default:
			CheckPathError(t, "Lstat", "lstat", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		if fs.HasFeature(avfs.FeatSymlink) {
			_, err = fs.Readlink(nonExistingFile)
			CheckPathError(t, "Readlink", "readlink", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		err = fs.Remove(nonExistingFile)
		CheckPathError(t, "Remove", "remove", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)

		err = fs.RemoveAll(nonExistingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", nonExistingFile, err)
		}

		err = fs.Rename(nonExistingFile, existingFile)
		CheckLinkError(t, "Rename", "rename", nonExistingFile, existingFile, avfs.ErrNoSuchFileOrDir, err)

		_, err = fs.Stat(nonExistingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "CreateFile", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		default:
			CheckPathError(t, "Stat", "stat", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		// err = FsR.Symlink(nonExistingFile, existingFile)
		// CheckLinkError(t, "Symlink", "symlink", nonExistingFile, existingFile, , err)

		err = fs.Truncate(nonExistingFile, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}
		default:
			CheckPathError(t, "Truncate", "truncate", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		err = fs.Walk(nonExistingFile, func(path string, info os.FileInfo, err error) error {
			return nil
		})

		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", nonExistingFile, err)
		}
	})

	t.Run("FileNonExistingFile", func(t *testing.T) {
		f, err := fs.Open(nonExistingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}
		default:
			CheckPathError(t, "Open", "open", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}

		if f == nil {
			t.Fatal("Open : want f to be != nil, got nil")
		}

		err = f.Chdir()

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chdir", "chdir", nonExistingFile, avfs.ErrWinNotSupported, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Chdir : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		err = f.Chmod(0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chmod", "chmod", nonExistingFile, avfs.ErrWinNotSupported, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Chmod : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		if fs.HasFeature(avfs.FeatIdentityMgr) {
			err = f.Chown(0, 0)
			if err != os.ErrInvalid {
				t.Errorf("Chown : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		err = f.Close()

		switch fs.OSType() {
		case avfs.OsWindows:
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}
		default:
			if err != os.ErrInvalid {
				t.Errorf("Close : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Read(buf)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Read", "read", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Read : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.ReadAt(buf, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadAt", "read", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("ReadAt : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Readdir(-1)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdir", "Readdir", nonExistingFile, avfs.ErrWinPathNotFound, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Readdir : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Readdirnames(-1)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", nonExistingFile, avfs.ErrWinPathNotFound, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Readdirnames : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Seek(0, io.SeekStart)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Seek : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Stat()

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "GetFileType", nonExistingFile, avfs.ErrFileClosing, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Stat : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		err = f.Sync()

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Sync", "sync", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Sync : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		err = f.Truncate(0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Truncate : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.Write(buf)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Write : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.WriteAt(buf, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("WriteAt : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		_, err = f.WriteString("")

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteString", "write", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("WriteString : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}

		err = f.Close()

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Close", "close", nonExistingFile, os.ErrClosed, err)
		default:
			if err != os.ErrInvalid {
				t.Errorf("Close : want error to be %v, got %v", os.ErrInvalid, err)
			}
		}
	})
}

// DirFuncOnFile tests directory functions on files.
func (sfs *SuiteFs) DirFuncOnFile() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	existingFile := fs.Join(rootDir, "existingFile")

	err := fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	nonExistingFile := fs.Join(existingFile, "invalid", "path")

	fs = sfs.GetFsRead()

	t.Run("DirFuncOnFileFs", func(t *testing.T) {
		err = fs.Chdir(existingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chdir", "chdir", existingFile, avfs.ErrWinDirNameInvalid, err)
		default:
			CheckPathError(t, "Chdir", "chdir", existingFile, avfs.ErrNotADirectory, err)
		}

		if fs.HasFeature(avfs.FeatSymlink) {
			_, err = fs.Lstat(nonExistingFile)
			CheckPathError(t, "Lstat", "lstat", nonExistingFile, avfs.ErrNotADirectory, err)
		}

		err = fs.Mkdir(nonExistingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", nonExistingFile, avfs.ErrNotADirectory, err)

		err = fs.MkdirAll(existingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", existingFile, avfs.ErrNotADirectory, err)

		err = fs.MkdirAll(nonExistingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", existingFile, avfs.ErrNotADirectory, err)

		_, err = fs.ReadDir(existingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadDir", "Readdir", existingFile, avfs.ErrNotADirectory, err)
		default:
			if fs.CurrentUser().IsRoot() {
				CheckSyscallError(t, "ReadDir", "readdirent", existingFile, avfs.ErrNotADirectory, err)
			} else {
				CheckPathError(t, "ReadDir", "readdirent", existingFile, avfs.ErrNotADirectory, err)
			}
		}

		_, err = fs.Stat(nonExistingFile)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "CreateFile", nonExistingFile, avfs.ErrNotADirectory, err)
		default:
			CheckPathError(t, "Stat", "stat", nonExistingFile, avfs.ErrNotADirectory, err)
		}

		_, err = fs.TempDir(existingFile, "")

		e, ok := err.(*os.PathError)
		if ok {
			CheckPathError(t, "TempDir", "mkdir", e.Path, avfs.ErrNotADirectory, err)
		}

		_, err = fs.TempFile(existingFile, "")

		e, ok = err.(*os.PathError)
		if ok {
			CheckPathError(t, "TempFile", "open", e.Path, avfs.ErrNotADirectory, err)
		}
	})

	f, err := fs.Open(existingFile)
	if err != nil {
		t.Fatalf("Create : want error to be nil, got %v", err)
	}

	defer f.Close()

	t.Run("DirFuncOnFileF", func(t *testing.T) {
		_, err = f.Readdir(-1)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdir", "Readdir", f.Name(), avfs.ErrNotADirectory, err)
		default:
			if fs.CurrentUser().IsRoot() {
				CheckSyscallError(t, "Readdir", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			} else {
				CheckPathError(t, "Readdir", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			}
		}

		_, err = f.Readdirnames(-1)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", f.Name(), avfs.ErrNotADirectory, err)
		default:
			if fs.CurrentUser().IsRoot() {
				CheckSyscallError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			} else {
				CheckPathError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
			}
		}

		err = f.Chdir()

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrNotADirectory, err)
		}
	})
}

// FileFuncOnDir test file functions on directories.
func (sfs *SuiteFs) FileFuncOnDir() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	pathDir := fs.Join(rootDir, "existingDir")

	err := fs.Mkdir(pathDir, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir : want error to be nil, got %v", err)
	}

	fs = sfs.GetFsRead()

	t.Run("FileFuncOnDir", func(t *testing.T) {
		f, err := fs.Open(pathDir)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Read(b)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Read", "read", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Read", "read", pathDir, avfs.ErrIsADirectory, err)
		}

		_, err = f.ReadAt(b, 0)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadAt", "read", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "ReadAt", "read", pathDir, avfs.ErrIsADirectory, err)
		}

		_, err = f.Seek(0, io.SeekStart)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}
		}

		if fs.HasFeature(avfs.FeatReadOnly) {
			return
		}

		err = f.Truncate(0)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Truncate", "truncate", pathDir, os.ErrInvalid, err)
		}

		_, err = f.Write(b)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Write", "write", pathDir, avfs.ErrBadFileDesc, err)
		}

		_, err = f.WriteAt(b, 0)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", pathDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "WriteAt", "write", pathDir, avfs.ErrBadFileDesc, err)
		}
	})
}

// FileFuncOnClosed tests functions on closed files.
func (sfs *SuiteFs) FileFuncOnClosed() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	existingFile := fs.Join(rootDir, "existingFile")

	err := fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	fs = sfs.GetFsRead()

	t.Run("FuncOnFileClosed", func(t *testing.T) {
		f, err := fs.Open(existingFile)
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

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdir", "Readdir", existingFile, avfs.ErrWinPathNotFound, err)
		default:
			if fs.CurrentUser().IsRoot() {
				if err == nil || err.Error() != avfs.ErrFileClosing.Error() {
					t.Errorf("Readdir %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
				}
			} else {
				CheckPathError(t, "Readdir", "readdirent", existingFile, avfs.ErrFileClosing, err)
			}
		}

		_, err = f.Readdirnames(-1)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", existingFile, avfs.ErrWinPathNotFound, err)
		default:
			if fs.CurrentUser().IsRoot() {
				if err == nil || err.Error() != avfs.ErrFileClosing.Error() {
					t.Errorf("Readdirnames %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
				}
			} else {
				CheckPathError(t, "Readdirnames", "readdirent", existingFile, avfs.ErrFileClosing, err)
			}
		}

		_, err = f.Stat()
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "GetFileType", existingFile, avfs.ErrFileClosing, err)
		default:
			CheckPathError(t, "Stat", "stat", existingFile, avfs.ErrFileClosing, err)
		}

		err = f.Sync()
		CheckPathError(t, "Sync", "sync", existingFile, os.ErrClosed, err)

		if fs.HasFeature(avfs.FeatReadOnly) {
			return
		}

		err = f.Chdir()
		CheckPathError(t, "Chdir", "chdir", existingFile, os.ErrClosed, err)

		err = f.Chmod(avfs.DefaultFilePerm)
		CheckPathError(t, "Chmod", "chmod", existingFile, os.ErrClosed, err)

		if fs.HasFeature(avfs.FeatIdentityMgr) {
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
func (sfs *SuiteFs) NotImplemented() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsRead()

	var (
		oldName = ""
		newName = ""
		newPath = ""
	)

	if !fs.HasFeature(avfs.FeatBasicFs) {
		err := fs.Chdir(rootDir)
		CheckPathError(t, "Chdir", "chdir", rootDir, avfs.ErrPermDenied, err)

		err = fs.Chmod(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Chmod", "chmod", rootDir, avfs.ErrPermDenied, err)

		err = fs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)

		err = fs.Chtimes(rootDir, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", rootDir, avfs.ErrPermDenied, err)

		fsC, ok := fs.(avfs.Cloner)
		if ok {
			fsClone := fsC.Clone()
			if fsClone != fs {
				t.Errorf("Clone : want cloned fs to be equal to fs, got different")
			}
		}

		_, err = fs.Create(rootDir)
		CheckPathError(t, "Create", "open", rootDir, avfs.ErrPermDenied, err)

		u := fs.CurrentUser()
		if u != dummyidm.NotImplementedUser {
			t.Errorf("CurrentUser : want User to be nil, got %v", u)
		}

		if um := fs.GetUMask(); um != 0 {
			t.Errorf("GetUMask : want umask to be 0, got %d", um)
		}

		_, err = fs.Getwd()
		CheckPathError(t, "Getwd", "getwd", "", avfs.ErrPermDenied, err)

		_, err = fs.Glob("")
		if err != nil {
			t.Errorf("Glob : want error to be nil, got %v", err)
		}

		_, err = fs.Lstat(rootDir)
		CheckPathError(t, "Lstat", "lstat", rootDir, avfs.ErrPermDenied, err)

		err = fs.Mkdir(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", rootDir, avfs.ErrPermDenied, err)

		err = fs.MkdirAll(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", rootDir, avfs.ErrPermDenied, err)

		name := fs.Name()
		if name != avfs.NotImplemented {
			t.Errorf("Name : want name to be %s, got %s", avfs.NotImplemented, name)
		}

		_, err = fs.Open(rootDir)
		CheckPathError(t, "Open", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = fs.OpenFile(rootDir, os.O_RDONLY, avfs.DefaultFilePerm)
		CheckPathError(t, "OpenFile", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = fs.ReadDir(rootDir)
		CheckPathError(t, "ReadDir", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = fs.ReadFile(rootDir)
		CheckPathError(t, "ReadFile", "open", rootDir, avfs.ErrPermDenied, err)

		err = fs.Remove(rootDir)
		CheckPathError(t, "Remove", "remove", rootDir, avfs.ErrPermDenied, err)

		err = fs.RemoveAll(rootDir)
		CheckPathError(t, "RemoveAll", "removeall", rootDir, avfs.ErrPermDenied, err)

		err = fs.Rename(rootDir, newPath)
		CheckLinkError(t, "Rename", "rename", rootDir, newPath, avfs.ErrPermDenied, err)

		if fs.SameFile(nil, nil) {
			t.Errorf("SameFile : want SameFile to be false, got true")
		}

		_, err = fs.Stat(rootDir)
		CheckPathError(t, "Stat", "stat", rootDir, avfs.ErrPermDenied, err)

		tmp := fs.GetTempDir()
		if tmp != avfs.TmpDir {
			t.Errorf("GetTempDir : want error to be %v, got %v", avfs.NotImplemented, tmp)
		}

		_, err = fs.TempDir(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempDir : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = fs.TempFile(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempFile : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = fs.Truncate(rootDir, 0)
		CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrPermDenied, err)

		fs.UMask(0)

		_, err = fs.User(UsrTest)
		if err != avfs.ErrPermDenied {
			t.Errorf("User : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		walkFunc := func(rootDir string, info os.FileInfo, err error) error { return nil }

		err = fs.Walk(rootDir, walkFunc)
		if err != nil {
			t.Errorf("User : want error to be nil, got %v", err)
		}

		err = fs.WriteFile(rootDir, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, "WriteFile", "open", rootDir, avfs.ErrPermDenied, err)

		_, err = fs.Abs(rootDir)
		if err != nil {
			t.Errorf("Name : want error to be nil, got %v", err)
		}

		base := fs.Base(rootDir)
		if base != rootDir[1:] {
			t.Errorf("Base : want Base to be %s, got %s", rootDir[1:], base)
		}

		clean := fs.Clean(rootDir)
		if clean != rootDir {
			t.Errorf("Clean : want Clean to be %s, got %s", rootDir, clean)
		}

		dir := fs.Dir(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Dir : want Dir to be %s, got %s", string(avfs.PathSeparator), dir)
		}

		b := fs.IsAbs(rootDir)
		if !b {
			t.Errorf("IsAbs : want result to be true, got false")
		}

		b = fs.IsPathSeparator(avfs.PathSeparator)
		if !b {
			t.Errorf("IsPathSeparator : want result to be true, got false")
		}

		join := fs.Join(rootDir, rootDir)
		if join != rootDir+rootDir {
			t.Errorf("Join : want join to be %s, got %s", rootDir+rootDir, join)
		}

		_, err = fs.Rel(rootDir, rootDir)
		if err != nil {
			t.Errorf("Rel : want error to be nil, got %v", err)
		}

		dir, _ = fs.Split(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Split : want dir to be %c, got %s", avfs.PathSeparator, dir)
		}
	}

	if !fs.HasFeature(avfs.FeatHardlink) {
		err := fs.Link(oldName, newName)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckLinkError(t, "Link", "link", oldName, newName, avfs.ErrWinPathNotFound, err)
		default:
			CheckLinkError(t, "Link", "link", oldName, newName, avfs.ErrPermDenied, err)
		}
	}

	if !fs.HasFeature(avfs.FeatIdentityMgr) {
		err := fs.Chown(rootDir, 0, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrPermDenied, err)
		}

		err = fs.Lchown(rootDir, 0, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrPermDenied, err)
		}

		f, err := fs.Open(rootDir)
		if err == nil {
			// We just need a file handle, be it valid or not, however if the handle is valid, it must be closed.
			defer f.Close()
		}

		err = f.Chown(0, 0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrPermDenied, err)
		}
	}

	if !fs.HasFeature(avfs.FeatSymlink) && fs.OSType() != avfs.OsWindows {
		_, err := fs.EvalSymlinks(rootDir)
		CheckPathError(t, "EvalSymlinks", "lstat", rootDir, avfs.ErrPermDenied, err)

		_, err = fs.Readlink(rootDir)
		CheckPathError(t, "Readlink", "readlink", rootDir, avfs.ErrPermDenied, err)

		err = fs.Symlink(rootDir, newPath)
		CheckLinkError(t, "Symlink", "symlink", rootDir, newPath, avfs.ErrPermDenied, err)
	}

	if !fs.HasFeature(avfs.FeatChroot) {
		err := fs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)
	}

	if !fs.HasFeature(avfs.FeatBasicFs) {
		f, _ := fs.Open(rootDir)

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
