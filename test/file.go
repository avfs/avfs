//
//  Copyright 2021 The AVFS authors
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
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// TestFileChdir tests File.Chdir function.
func (sfs *SuiteFS) TestFileChdir(t *testing.T) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Chdir()
		CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	existingFile := sfs.CreateEmptyFile(t)

	vfs = sfs.vfsRead

	t.Run("FileChdir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			f, err := vfs.Open(path)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", path, err)
			}

			defer f.Close()

			err = f.Chdir()
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", path, err)
			}

			curDir, err := vfs.Getwd()
			if err != nil {
				t.Errorf("Getwd %s : want error to be nil, got %v", path, err)
			}

			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})

	t.Run("FileChdirOnFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Chdir()
		CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrNotADirectory, err)
	})
}

// TestFileChmod tests File.Chmod function.
func (sfs *SuiteFS) TestFileChmod(t *testing.T) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Chmod(0)
		CheckPathError(t, "Chmod", "chmod", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	_ = rootDir
}

// TestFileChown tests File.Chown function.
func (sfs *SuiteFS) TestFileChown(t *testing.T) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Chown(0, 0)
		CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	_ = rootDir
}

// TestFileCloseRead tests File.Close function for read only files.
func (sfs *SuiteFS) TestFileCloseRead(t *testing.T) {
	_, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Close()
		CheckPathError(t, "Close", "close", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.CreateFile(t, data)

	openInfo, err := vfs.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
	}

	t.Run("FileCloseReadOnly", func(t *testing.T) {
		vfs = sfs.vfsRead

		f, err := vfs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		closeInfo, err := vfs.Stat(path)
		if err != nil {
			t.Errorf("Stat %s : want error to be nil, got %v", path, err)
		}

		if !reflect.DeepEqual(openInfo, closeInfo) {
			t.Errorf("Stat %s : open info != close info\n%v\n%v", path, openInfo, closeInfo)
		}

		err = f.Close()
		CheckPathError(t, "Close", "close", path, os.ErrClosed, err)
	})
}

// TestFileCloseWrite tests File.Close function for read/write files.
func (sfs *SuiteFS) TestFileCloseWrite(t *testing.T) {
	_, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.CreateFile(t, data)

	openInfo, err := vfs.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
	}

	t.Run("FileCloseWrite", func(t *testing.T) {
		f, err := vfs.OpenFile(path, os.O_APPEND|os.O_WRONLY, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		n, err := f.Write(data)
		if err != nil {
			t.Fatalf("Write : want error to be nil, got %v", err)
		}

		if n != len(data) {
			t.Fatalf("Write : want bytes written to be %d, got %d", len(data), n)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		closeInfo, err := vfs.Stat(path)
		if err != nil {
			t.Errorf("Stat %s : want error to be nil, got %v", path, err)
		}

		if reflect.DeepEqual(openInfo, closeInfo) {
			t.Errorf("Stat %s : open info != close info\n%v\n%v", path, openInfo, closeInfo)
		}

		err = f.Close()
		CheckPathError(t, "Close", "close", path, os.ErrClosed, err)
	})
}

// TestFileFd tests File.Fd function.
func (sfs *SuiteFS) TestFileFd(t *testing.T) {
	vfs := sfs.vfsRead

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		fd := f.Fd()
		if fd != 0 {
			t.Errorf("Fd : want Fd to be 0, got %v", fd)
		}

		return
	}
}

// TestFileFuncOnClosedFile tests functions on closed files.
func (sfs *SuiteFS) TestFileFuncOnClosedFile(t *testing.T) {
	_, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	existingFile := sfs.CreateEmptyFile(t)

	vfs = sfs.vfsRead

	t.Run("FileFuncOnClosedFile", func(t *testing.T) {
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
			if err.Error() != avfs.ErrFileClosing.Error() {
				t.Errorf("Readdir %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
			}
		}

		_, err = f.Readdirnames(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", existingFile, avfs.ErrWinPathNotFound, err)
		default:
			if err.Error() != avfs.ErrFileClosing.Error() {
				t.Errorf("Readdirnames %s : want error to be %v, got %v", existingFile, avfs.ErrFileClosing, err)
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

// TestFileName tests File.Name function.
func (sfs *SuiteFS) TestFileName(t *testing.T) {
	vfs := sfs.vfsRead

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		name := f.Name()
		if name != avfs.NotImplemented {
			t.Errorf("Name : want error to be %v, got %v", avfs.NotImplemented, name)
		}

		return
	}
}

// TestFileNilPtr test calls to File methods when f is a nil File.
func TestFileNilPtr(t *testing.T, f avfs.File) {
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

// TestFileRead tests File.Read and File.ReadAt functions.
func (sfs *SuiteFS) TestFileRead(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Read([]byte{})
		CheckPathError(t, "Read", "read", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.ReadAt([]byte{}, 0)
		CheckPathError(t, "ReadAt", "read", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.CreateFile(t, data)

	vfs = sfs.vfsRead

	t.Run("FileRead", func(t *testing.T) {
		const bufSize = 5

		f, err := vfs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		buf := make([]byte, bufSize)
		for i := 0; ; i += bufSize {
			n, err1 := f.Read(buf)
			if err1 != nil {
				if err1 == io.EOF {
					break
				}

				t.Errorf("Read : want error to be %v, got %v", io.EOF, err1)
			}

			if !bytes.Equal(buf[:n], data[i:i+n]) {
				t.Errorf("Read : want content to be %s, got %s", buf[:n], data[i:i+n])
			}
		}
	})

	t.Run("FileReadAt", func(t *testing.T) {
		const bufSize = 3

		f, err := vfs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		var n int
		rb := make([]byte, bufSize)
		for i := len(data); i > 0; i -= bufSize {
			n, err = f.ReadAt(rb, int64(i-bufSize))
			if err != nil {
				t.Errorf("ReadAt : want error to be nil, got %v", err)
			}

			if n != bufSize {
				t.Errorf("ReadAt : want bytes read to be %d, got %d", bufSize, n)
			}

			if !bytes.Equal(rb, data[i-bufSize:i]) {
				t.Errorf("ReadAt : want bytes read to be %d, got %d", bufSize, n)
			}
		}
	})

	t.Run("FileReadAfterEndOfFile", func(t *testing.T) {
		f, err := vfs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		off := int64(len(data) * 2)

		n, err := f.ReadAt(b, off)
		if err != io.EOF {
			t.Errorf("ReadAt : want error to be %v, got %v", io.EOF, err)
		}

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}

		n, err = f.ReadAt(b, -1)
		CheckPathError(t, "ReadAt", "readat", path, avfs.ErrNegativeOffset, err)

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}
	})

	t.Run("FileReadOnDir", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Read(b)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Read", "read", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Read", "read", rootDir, avfs.ErrIsADirectory, err)
		}

		_, err = f.ReadAt(b, 0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadAt", "read", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "ReadAt", "read", rootDir, avfs.ErrIsADirectory, err)
		}
	})
}

// TestFileReadDir tests File.ReadDir function.
func (sfs *SuiteFS) TestFileReadDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Readdir(0)
		CheckPathError(t, "Readdir", "readdirent", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	rndTree := sfs.CreateRndDir(t)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)
	existingFile := rndTree.Files[0]

	vfs = sfs.vfsRead

	const maxRead = 7

	t.Run("FileReadDirN", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("ReadDir : want error to be nil, got %v", err)
		}

		defer f.Close()

		var rdInfos []os.FileInfo

		for {
			rdInfoN, err := f.Readdir(maxRead)
			if err == io.EOF {
				break
			}

			if err != nil {
				t.Fatalf("ReadDir : want error to be nil, got %v", err)
			}

			rdInfos = append(rdInfos, rdInfoN...)
		}

		var gDirs, gFiles, gSymlinks int
		for _, rdInfo := range rdInfos {
			mode := rdInfo.Mode()
			switch {
			case mode.IsDir():
				gDirs++
			case mode&os.ModeSymlink != 0:
				gSymlinks++
			default:
				gFiles++
			}
		}

		if wDirs != gDirs {
			t.Errorf("ReadDirN : want number of dirs to be %d, got %d", wDirs, gDirs)
		}

		if wFiles != gFiles {
			t.Errorf("ReadDirN : want number of files to be %d, got %d", wFiles, gFiles)
		}

		if wSymlinks != gSymlinks {
			t.Errorf("ReadDirN : want number of symbolic links to be %d, got %d", wSymlinks, gSymlinks)
		}
	})

	t.Run("FileReadDirExistingFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		_, err = f.Readdir(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdir", "Readdir", f.Name(), avfs.ErrNotADirectory, err)
		default:
			CheckSyscallError(t, "Readdir", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
		}
	})
}

// TestFileReaddirnames tests File.Readdirnames function.
func (sfs *SuiteFS) TestFileReaddirnames(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Readdirnames(0)
		CheckPathError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	rndTree := sfs.CreateRndDir(t)
	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)
	existingFile := rndTree.Files[0]

	vfs = sfs.vfsRead

	t.Run("FileReaddirnamesAll", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("TestFileReaddirnames : want error to be nil, got %v", err)
		}

		names, err := f.Readdirnames(-1)
		if err != nil {
			t.Errorf("TestFileReaddirnames : want error to be nil, got %v", err)
		}

		if wAll != len(names) {
			t.Errorf("TestFileReaddirnames : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("FileReaddirnamesN", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("TestFileReaddirnames : want error to be nil, got %v", err)
		}

		var names []string

		for {
			namesN, err := f.Readdirnames(11)
			if err == io.EOF {
				break
			}

			if err != nil {
				t.Fatalf("ReadDirNamesN : want error to be nil, got %v", err)
			}

			names = append(names, namesN...)
		}

		if wAll != len(names) {
			t.Errorf("ReadDirNamesN : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("FileReaddirnamesExistingFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		defer f.Close()

		_, err = f.Readdirnames(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Readdirnames", "Readdir", f.Name(), avfs.ErrNotADirectory, err)
		default:
			CheckSyscallError(t, "Readdirnames", "readdirent", f.Name(), avfs.ErrNotADirectory, err)
		}
	})
}

// TestFileSeek tests File.Seek function.
func (sfs *SuiteFS) TestFileSeek(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Seek(0, io.SeekStart)
		CheckPathError(t, "Seek", "seek", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.CreateFile(t, data)

	vfs = sfs.vfsRead

	f, err := vfs.Open(path)
	if err != nil {
		t.Fatalf("Open : want error to be nil, got %v", err)
	}

	defer f.Close()

	var pos int64

	lenData := int64(len(data))

	t.Run("TestFileSeek", func(t *testing.T) {
		for i := 0; i < len(data); i++ {
			pos, err = f.Seek(int64(i), io.SeekStart)
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}

			if int(pos) != i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}

		for i := 0; i < len(data); i++ {
			pos, err = f.Seek(-int64(i), io.SeekEnd)
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}

			if int(pos) != len(data)-i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}

		_, err = f.Seek(0, io.SeekEnd)
		if err != nil {
			t.Fatalf("Seek : want error to be nil, got %v", err)
		}

		for i := len(data) - 1; i >= 0; i-- {
			pos, err = f.Seek(-1, io.SeekCurrent)
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}

			if int(pos) != i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}
	})

	t.Run("FileSeekInvalidStart", func(t *testing.T) {
		pos, err = f.Seek(-1, io.SeekStart)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", f.Name(), avfs.ErrWinNegativeSeek, err)
		default:
			CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}

		wantPos := lenData * 2

		pos, err = f.Seek(wantPos, io.SeekStart)
		if err != nil {
			t.Errorf("Seek : want error to be nil, got %v", err)
		}

		if pos != wantPos {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}
	})

	t.Run("FileSeekInvalidEnd", func(t *testing.T) {
		pos, err = f.Seek(1, io.SeekEnd)
		if err != nil {
			t.Errorf("Seek : want error to be nil, got %v", err)
		}

		wantPos := lenData + 1
		if pos != wantPos {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}

		pos, err = f.Seek(-lenData*2, io.SeekEnd)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", f.Name(), avfs.ErrWinNegativeSeek, err)
		default:
			CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}
	})

	t.Run("FileSeekInvalidCur", func(t *testing.T) {
		wantPos := lenData / 2

		pos, err = f.Seek(wantPos, io.SeekStart)
		if err != nil || pos != wantPos {
			t.Fatalf("Seek : want  pos to be 0 and error to be nil, got %d, %v", pos, err)
		}

		pos, err = f.Seek(-lenData, io.SeekCurrent)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", f.Name(), avfs.ErrWinNegativeSeek, err)
		default:
			CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}

		pos, err = f.Seek(lenData, io.SeekCurrent)
		if err != nil {
			t.Errorf("Seek : want error to be nil, got %v", err)
		}

		if pos != lenData/2+lenData {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}
	})

	t.Run("FileSeekInvalidWhence", func(t *testing.T) {
		pos, err = f.Seek(0, 10)

		switch vfs.OSType() {
		case avfs.OsWindows:
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}
		default:
			CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}
	})

	t.Run("FileSeekOnDir", func(t *testing.T) {
		f, err = vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		_, err = f.Seek(0, io.SeekStart)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Seek", "seek", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}
		}
	})
}

// TestFileStat tests File.Stat function.
func (sfs *SuiteFS) TestFileStat(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Stat()
		CheckPathError(t, "Stat", "stat", f.Name(), avfs.ErrPermDenied, err)
	}

	_ = rootDir
}

// TestFileSync tests File.Sync function.
func (sfs *SuiteFS) TestFileSync(t *testing.T) {
	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Sync()
		CheckPathError(t, "Sync", "sync", f.Name(), avfs.ErrPermDenied, err)
	}
}

// TestFileTruncate tests File.Truncate function.
func (sfs *SuiteFS) TestFileTruncate(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		err := f.Truncate(0)
		CheckPathError(t, "Truncate", "truncate", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileTruncate", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, len(data))
		for i := len(data) - 1; i >= 0; i-- {
			err = f.Truncate(int64(i))
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			_, err = f.ReadAt(b, 0)
			if err != io.EOF {
				t.Errorf("Read : want error to be nil, got %v", err)
			}

			if !bytes.Equal(data[:i], b[:i]) {
				t.Errorf("Truncate : want data to be %s, got %s", data[:i], b[:i])
			}
		}
	})

	t.Run("Truncate", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		for i := len(data); i >= 0; i-- {
			err := vfs.Truncate(path, int64(i))
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			d, err := vfs.ReadFile(path)
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			if len(d) != i {
				t.Errorf("Truncate : want length to be %d, got %d", i, len(d))
			}
		}
	})

	t.Run("TruncateOnDir", func(t *testing.T) {
		err := vfs.Truncate(rootDir, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "open", rootDir, avfs.ErrIsADirectory, err)
		default:
			CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrIsADirectory, err)
		}
	})

	t.Run("FileTruncateOnDir", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Truncate(0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Truncate", "truncate", rootDir, os.ErrInvalid, err)
		}
	})

	t.Run("TruncateSizeNegative", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		err := vfs.Truncate(path, -1)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", path, avfs.ErrWinNegativeSeek, err)
		default:
			CheckPathError(t, "Truncate", "truncate", path, os.ErrInvalid, err)
		}

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Truncate(-1)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", path, avfs.ErrWinNegativeSeek, err)
		default:
			CheckPathError(t, "Truncate", "truncate", path, os.ErrInvalid, err)
		}
	})

	t.Run("TruncateSizeBiggerFileSize", func(t *testing.T) {
		path := sfs.CreateFile(t, data)
		newSize := len(data) * 2

		err := vfs.Truncate(path, int64(newSize))
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		info, err := vfs.Stat(path)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		gotContent, err := vfs.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile : want error to be nil, got %v", err)
		}

		wantAdded := bytes.Repeat([]byte{0}, len(data))
		gotAdded := gotContent[len(data):]
		if !bytes.Equal(wantAdded, gotAdded) {
			t.Errorf("Bytes Added : want %v, got %v", wantAdded, gotAdded)
		}
	})

	t.Run("TruncateNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		err := vfs.Truncate(nonExistingFile, 0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}
		default:
			CheckPathError(t, "Truncate", "truncate", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// TestFileWrite tests File.Write and File.WriteAt functions.
func (sfs *SuiteFS) TestFileWrite(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.Write([]byte{})
		CheckPathError(t, "Write", "write", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, "WriteAt", "write", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileWrite", func(t *testing.T) {
		path := vfs.Join(rootDir, "TestFileWrite.txt")

		f, err := vfs.Create(path)
		if err != nil {
			t.Fatalf("Create : want error to be nil, got %v", err)
		}

		defer f.Close()

		for i := 0; i < len(data); i += 3 {
			buf3 := data[i : i+3]
			var n int

			n, err = f.Write(buf3)
			if err != nil {
				t.Errorf("Write : want error to be nil, got %v", err)
			}

			if len(buf3) != n {
				t.Errorf("Write : want bytes written to be %d, got %d", len(buf3), n)
			}
		}

		rb, err := vfs.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("FileWriteAt", func(t *testing.T) {
		path := vfs.Join(rootDir, "TestFileWriteAt.txt")

		f, err := vfs.OpenFile(path, os.O_CREATE|os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		for i := len(data); i > 0; i -= 3 {
			var n int
			n, err = f.WriteAt(data[i-3:i], int64(i-3))
			if err != nil {
				t.Errorf("WriteAt : want error to be nil, got %v", err)
			}

			if n != 3 {
				t.Errorf("WriteAt : want bytes written to be %d, got %d", 3, n)
			}
		}

		err = f.Close()
		if err != nil {
			t.Errorf("Close : want error to be nil, got %v", err)
		}

		rb, err := vfs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("FileWriteNegativeOffset", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.WriteAt(data, -1)
		CheckPathError(t, "WriteAt", "writeat", path, avfs.ErrNegativeOffset, err)

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}
	})

	t.Run("FileWriteAtAfterEndOfFile", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		off := int64(len(data) * 3)

		n, err := f.WriteAt(data, off)
		if err != nil {
			t.Errorf("WriteAt : want error to be nil, got %v", err)
		}

		if n != len(data) {
			t.Errorf("WriteAt : want bytes written to be %d, got %d", len(data), n)
		}

		want := make([]byte, int(off)+len(data))
		_ = copy(want, data)
		_ = copy(want[off:], data)

		got, err := vfs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(want, got) {
			t.Errorf("want : %s\ngot  : %s", want, got)
		}
	})

	t.Run("FileReadOnly", func(t *testing.T) {
		path := sfs.CreateFile(t, data)

		f, err := vfs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, len(data)*2)
		n, err := f.Write(b)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", path, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "Write", "write", path, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.Read(b)
		if err != nil {
			t.Errorf("Read : want error to be nil, got %v", err)
		}

		if !bytes.Equal(data, b[:n]) {
			t.Errorf("Read : want data to be %s, got %s", data, b[:n])
		}

		n, err = f.WriteAt(b, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", path, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "WriteAt", "write", path, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("WriteAt : want bytes read to be 0, got %d", n)
		}

		n, err = f.ReadAt(b, 0)
		if err != io.EOF {
			t.Errorf("ReadAt : want error to be nil, got %v", err)
		}

		if !bytes.Equal(data, b[:n]) {
			t.Errorf("ReadAt : want data to be %s, got %s", data, b[:n])
		}
	})

	t.Run("FileWriteOnDir", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Write(b)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Write", "write", rootDir, avfs.ErrBadFileDesc, err)
		}

		_, err = f.WriteAt(b, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "WriteAt", "write", rootDir, avfs.ErrBadFileDesc, err)
		}
	})
}

// TestFileWriteString tests File.WriteString function.
func (sfs *SuiteFS) TestFileWriteString(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenNonExistingFile(t)

		_, err := f.WriteString("")
		CheckPathError(t, "WriteString", "write", f.Name(), avfs.ErrPermDenied, err)
	}

	_ = rootDir
}

// TestFileWriteTime checks that modification time is updated on write operations.
func (sfs *SuiteFS) TestFileWriteTime(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	existingFile := vfs.Join(rootDir, "ExistingFile.txt")

	var start, end int64

	f, err := vfs.Create(existingFile)
	if err != nil {
		t.Fatalf("Create : want error to be nil, got %v", err)
	}

	// CompareTime tests if the modification time of the file has changed.
	CompareTime := func(mustChange bool) {
		time.Sleep(10 * time.Millisecond)

		info, err := f.Stat() //nolint:govet // Shadows previous declaration of err.
		if err != nil {
			if errors.Unwrap(err).Error() != avfs.ErrFileClosing.Error() {
				t.Fatalf("Stat : want error to be nil, got %v", err)
			}

			info, err = vfs.Stat(existingFile)
			if err != nil {
				t.Fatalf("Stat : want error to be nil, got %v", err)
			}
		}

		start = end
		end = info.ModTime().UnixNano()

		// dont compare for the first time.
		if start == 0 {
			return
		}

		if mustChange && (start >= end) {
			t.Errorf("Stat %s : want start time < end time\nstart : %v\nend : %v", existingFile, start, end)
		}

		if !mustChange && (start != end) {
			t.Errorf("Stat %s : want start time == end time\nstart : %v\nend : %v", existingFile, start, end)
		}
	}

	CompareTime(true)

	t.Run("TimeWrite", func(t *testing.T) {
		_, err = f.Write(data)
		if err != nil {
			t.Fatalf("Write : want error to be nil, got %v", err)
		}

		CompareTime(true)
	})

	t.Run("TimeWriteAt", func(t *testing.T) {
		_, err = f.WriteAt(data, 5)
		if err != nil {
			t.Fatalf("WriteAt : want error to be nil, got %v", err)
		}

		CompareTime(true)
	})

	t.Run("TimeTruncate", func(t *testing.T) {
		err = f.Truncate(5)
		if err != nil {
			t.Fatalf("Truncate : want error to be nil, got %v", err)
		}

		CompareTime(true)
	})

	t.Run("TimeClose", func(t *testing.T) {
		err = f.Close()
		if err != nil {
			t.Fatalf("Close : want error to be nil, got %v", err)
		}

		CompareTime(false)
	})
}
