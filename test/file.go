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
	"io/fs"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// TestFileChdir tests File.Chdir function.
func (sfs *SuiteFS) TestFileChdir(t *testing.T, testDir string) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chdir()
		CheckPathError(t, err).Op("chdir").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)

	vfs = sfs.vfsTest

	t.Run("FileChdir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

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
		f, fileName := sfs.OpenedEmptyFile(t, testDir)
		defer f.Close()

		err := f.Chdir()
		CheckPathError(t, err).Op("chdir").Path(fileName).Err(avfs.ErrNotADirectory)
	})

	t.Run("FileChdirClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		err := f.Chdir()
		CheckPathError(t, err).Op("chdir").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChdirNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chdir()

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("chdir").Path(defaultFile).Err(avfs.ErrWinNotSupported)
		default:
			CheckInvalid(t, "Chdir", err)
		}
	})
}

// TestFileChmod tests File.Chmod function.
func (sfs *SuiteFS) TestFileChmod(t *testing.T, testDir string) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chmod(0)
		CheckPathError(t, err).Op("chmod").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.OpenedEmptyFile(t, testDir)

		err := f.Chmod(0)
		CheckPathError(t, err).Op("chmod").Path(fileName).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("FileChmodClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		err := f.Chmod(avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("chmod").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChmodNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chmod(0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("chmod").Path(defaultFile).Err(avfs.ErrWinNotSupported)
		default:
			CheckInvalid(t, "Chmod", err)
		}
	})
}

// TestFileChown tests File.Chown function.
func (sfs *SuiteFS) TestFileChown(t *testing.T, testDir string) {
	if sfs.OSType() == avfs.OsWindows {
		return
	}

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.OpenedEmptyFile(t, testDir)

		err := f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(fileName).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("FileChown", func(t *testing.T) {
		f, fileName := sfs.OpenedEmptyFile(t, testDir)

		u := vfs.CurrentUser()
		uid, gid := u.Uid(), u.Gid()

		err := f.Chown(uid, gid)
		if err != nil {
			t.Errorf("Chown %s : want error to be nil, got %v", fileName, err)
		}
	})

	t.Run("FileChownClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		err := f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChownNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Chown(0, 0)
		CheckInvalid(t, "Chown", err)
	})
}

// TestFileCloseRead tests File.Close function for read only files.
func (sfs *SuiteFS) TestFileCloseRead(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Close()
		CheckPathError(t, err).Op("close").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.ExistingFile(t, testDir, data)

	t.Run("FileCloseReadOnly", func(t *testing.T) {
		vfs = sfs.vfsTest

		openInfo, err := vfs.Stat(path)
		if err != nil {
			t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
		}

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
		CheckPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed)
	})

	t.Run("FileCloseNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Close()
		CheckInvalid(t, "Close", err)
	})
}

// TestFileCloseWrite tests File.Close function for read/write files.
func (sfs *SuiteFS) TestFileCloseWrite(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.ExistingFile(t, testDir, data)

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
		CheckPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed)
	})
}

// TestFileFd tests File.Fd function.
func (sfs *SuiteFS) TestFileFd(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		fd := f.Fd()
		if fd != 0 {
			t.Errorf("Fd %s : want Fd to be 0, got %v", avfs.NotImplemented, fd)
		}

		return
	}

	f, fileName := sfs.ClosedFile(t, testDir)

	fd := f.Fd()
	if fd != math.MaxUint64 {
		t.Errorf("Fd %s : want Fd to be %d, got %d", fileName, uint64(math.MaxUint64), fd)
	}
}

// TestFileName tests File.Name function.
func (sfs *SuiteFS) TestFileName(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		name := f.Name()
		if name != avfs.NotImplemented {
			t.Errorf("Name : want name to be %s, got %s", avfs.NotImplemented, name)
		}

		return
	}

	f, wantName := sfs.ClosedFile(t, testDir)

	name := f.Name()
	if name != wantName {
		t.Errorf("Name %s : want Name to be %s, got %s", wantName, wantName, name)
	}
}

// FileNilPtr test calls to File methods when f is a nil File.
func FileNilPtr(t *testing.T, f avfs.File) {
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

	_, err = f.ReadDir(0)
	CheckInvalid(t, "ReadDir", err)

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
func (sfs *SuiteFS) TestFileRead(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Read([]byte{})
		CheckPathError(t, err).Op("read").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		_, err = f.ReadAt([]byte{}, 0)
		CheckPathError(t, err).Op("read").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.ExistingFile(t, testDir, data)

	vfs = sfs.vfsTest

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

	t.Run("FileReadNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.Read(buf)
		CheckInvalid(t, "Read", err)
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

	t.Run("FileReadAtNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.ReadAt(buf, 0)
		CheckInvalid(t, "ReadAt", err)
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
		CheckPathError(t, err).Op("readat").Path(path).ErrAsString(avfs.ErrNegativeOffset)

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}
	})

	t.Run("FileReadOnDir", func(t *testing.T) {
		f, err := vfs.Open(testDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Read(b)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("read").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			CheckPathError(t, err).Op("read").Path(testDir).Err(avfs.ErrIsADirectory)
		}

		_, err = f.ReadAt(b, 0)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("read").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			CheckPathError(t, err).Op("read").Path(testDir).Err(avfs.ErrIsADirectory)
		}
	})

	t.Run("FileReadClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		b := make([]byte, 1)

		_, err := f.Read(b)
		CheckPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed)

		_, err = f.ReadAt(b, 0)
		CheckPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileReadDir tests File.ReadDir function.
func (sfs *SuiteFS) TestFileReadDir(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f, _ := sfs.OpenedEmptyFile(t, testDir)

		_, err := f.ReadDir(0)
		CheckPathError(t, err).Op("readdirent").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	rndTree := sfs.RandomDir(t, testDir)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)

	vfs = sfs.vfsTest

	const maxRead = 7

	t.Run("FileReadDirN", func(t *testing.T) {
		f, err := vfs.Open(testDir)
		if err != nil {
			t.Fatalf("ReadDir : want error to be nil, got %v", err)
		}

		defer f.Close()

		var dirEntries []fs.DirEntry

		for {
			dirEntriesN, err := f.ReadDir(maxRead)
			if err == io.EOF {
				break
			}

			if err != nil {
				t.Fatalf("ReadDir : want error to be nil, got %v", err)
			}

			dirEntries = append(dirEntries, dirEntriesN...)
		}

		var gDirs, gFiles, gSymlinks int
		for _, dirEntry := range dirEntries {
			switch {
			case dirEntry.IsDir():
				gDirs++
			case dirEntry.Type()&fs.ModeSymlink != 0:
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
		f, fileName := sfs.OpenedEmptyFile(t, testDir)
		defer f.Close()

		_, err := f.ReadDir(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("readdir").Path(fileName).Err(avfs.ErrNotADirectory)
		default:
			CheckPathError(t, err).Op("readdirent").Path(fileName).Err(avfs.ErrNotADirectory)
		}
	})

	t.Run("FileReadDirClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		_, err := f.ReadDir(-1)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("readdir").Path(fileName).Err(avfs.ErrWinPathNotFound)
		default:
			CheckPathError(t, err).Op("readdirent").Path(fileName).ErrAsString(avfs.ErrFileClosing)
		}
	})

	t.Run("FileReadDirNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.ReadDir(-1)
		CheckInvalid(t, "ReadDir", err)
	})
}

// TestFileReaddirnames tests File.Readdirnames function.
func (sfs *SuiteFS) TestFileReaddirnames(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Readdirnames(0)
		CheckPathError(t, err).Op("readdirent").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	rndTree := sfs.RandomDir(t, testDir)
	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)
	existingFile := rndTree.Files[0]

	vfs = sfs.vfsTest

	t.Run("FileReaddirnamesAll", func(t *testing.T) {
		f, err := vfs.Open(testDir)
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
		f, err := vfs.Open(testDir)
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
			CheckPathError(t, err).Op("readdir").Path(f.Name()).Err(avfs.ErrNotADirectory)
		default:
			CheckPathError(t, err).Op("readdirent").Path(f.Name()).Err(avfs.ErrNotADirectory)
		}
	})

	t.Run("FileReaddirnamesClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		_, err := f.Readdirnames(-1)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("readdir").Path(fileName).Err(avfs.ErrWinPathNotFound)
		default:
			CheckPathError(t, err).Op("readdirent").Path(fileName).ErrAsString(avfs.ErrFileClosing)
		}
	})

	t.Run("FileReaddirnamesNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Readdirnames(-1)
		CheckInvalid(t, "Readdirnames", err)
	})
}

// TestFileSeek tests File.Seek function.
func (sfs *SuiteFS) TestFileSeek(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Seek(0, io.SeekStart)
		CheckPathError(t, err).Op("seek").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.ExistingFile(t, testDir, data)

	vfs = sfs.vfsTest

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
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrWinNegativeSeek)
		default:
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrInvalidArgument)
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
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrWinNegativeSeek)
		default:
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrInvalidArgument)
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
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrWinNegativeSeek)
		default:
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrInvalidArgument)
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
			CheckPathError(t, err).Op("seek").Path(f.Name()).Err(avfs.ErrInvalidArgument)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}
	})

	t.Run("FileSeekOnDir", func(t *testing.T) {
		f, err = vfs.Open(testDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		_, err = f.Seek(0, io.SeekStart)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("seek").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			if err != nil {
				t.Errorf("Seek : want error to be nil, got %v", err)
			}
		}
	})

	t.Run("FileSeekClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		CheckPathError(t, err).Op("seek").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileSeekNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		CheckInvalid(t, "Seek", err)
	})
}

// TestFileStat tests File.Stat function.
func (sfs *SuiteFS) TestFileStat(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Stat()
		CheckPathError(t, err).Op("stat").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest

	t.Run("FileStatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			f, err := vfs.Open(path)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", path, err)
			}

			info, err := f.Stat()
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)

				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.GetUMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("FileStatFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			f, err := vfs.Open(path)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", path, err)
			}

			info, err := f.Stat()
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)

				continue
			}

			if info.Name() != vfs.Base(path) {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ vfs.GetUMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = 0o666
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", path, wantSize, info.Size())
			}
		}
	})

	t.Run("FileStatSymlink", func(t *testing.T) {
		for _, sl := range GetSampleSymlinksEval(vfs) {
			if sl.WantErr != nil {
				continue
			}

			newPath := vfs.Join(testDir, sl.NewName)
			oldPath := vfs.Join(testDir, sl.OldName)

			f, err := vfs.Open(newPath)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", newPath, err)
			}

			info, err := f.Stat()
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)

				continue
			}

			var (
				wantName string
				wantMode fs.FileMode
			)

			if sl.IsSymlink {
				wantName = vfs.Base(newPath)
			} else {
				wantName = vfs.Base(oldPath)
			}

			wantMode = sl.Mode
			if wantName != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", newPath, wantName, info.Name())
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", newPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("FileStatNonExistingFile", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Stat()
		CheckInvalid(t, "Stat", err)
	})

	t.Run("FileStatSubDirOnFile", func(t *testing.T) {
		path := vfs.Join(testDir, files[0].Path, defaultNonExisting)

		f, err := vfs.Open(path)
		CheckPathError(t, err).Op("open").Path(path).Err(avfs.ErrNotADirectory)

		_, err = f.Stat()
		CheckInvalid(t, "Stat", err)
	})

	t.Run("FileStatClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		_, err := f.Stat()

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("GetFileType").Path(fileName).ErrAsString(avfs.ErrFileClosing)
		default:
			CheckPathError(t, err).Op("stat").Path(fileName).ErrAsString(avfs.ErrFileClosing)
		}
	})

	t.Run("FileStatNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Stat()
		CheckInvalid(t, "Stat", err)
	})
}

// TestFileSync tests File.Sync function.
func (sfs *SuiteFS) TestFileSync(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Sync()
		CheckPathError(t, err).Op("sync").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("FileSyncClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		err := f.Sync()
		CheckPathError(t, err).Op("sync").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileSyncNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Sync()
		CheckInvalid(t, "Sync", err)
	})
}

// TestFileTruncate tests File.Truncate function.
func (sfs *SuiteFS) TestFileTruncate(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.OpenedEmptyFile(t, testDir)

		err := f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(fileName).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileTruncate", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

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

	t.Run("FileTruncateOnDir", func(t *testing.T) {
		f, err := vfs.Open(testDir)
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Truncate(0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("truncate").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			CheckPathError(t, err).Op("truncate").Path(testDir).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("FileTruncateSizeNegative", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Truncate(-1)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("truncate").Path(path).Err(avfs.ErrWinNegativeSeek)
		default:
			CheckPathError(t, err).Op("truncate").Path(path).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("FileTruncateSizeBiggerFileSize", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

		newSize := len(data) * 2

		err = f.Truncate(int64(newSize))
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		info, err := f.Stat()
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		f.Close()

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

	t.Run("FileTruncateNonExistingFile", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		err := f.Truncate(0)
		CheckInvalid(t, "Truncate", err)
	})

	t.Run("FileTruncateClosed", func(t *testing.T) {
		f, fileName := sfs.ClosedFile(t, testDir)

		err := f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileWrite tests File.Write and File.WriteAt functions.
func (sfs *SuiteFS) TestFileWrite(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.Write([]byte{})
		CheckPathError(t, err).Op("write").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, err).Op("write").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.OpenedEmptyFile(t, testDir)

		_, err := f.Write([]byte{})
		CheckPathError(t, err).Op("write").Path(fileName).Err(avfs.ErrPermDenied)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, err).Op("write").Path(fileName).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileWrite", func(t *testing.T) {
		path := vfs.Join(testDir, "TestFileWrite.txt")

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

	t.Run("FileWriteNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.Write(buf)
		CheckInvalid(t, "Write", err)
	})

	t.Run("FileWriteAtNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.WriteAt(buf, 0)
		CheckInvalid(t, "WriteAt", err)
	})

	t.Run("FileWriteAt", func(t *testing.T) {
		path := vfs.Join(testDir, "TestFileWriteAt.txt")

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
		path := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.WriteAt(data, -1)
		CheckPathError(t, err).Op("writeat").Path(path).ErrAsString(avfs.ErrNegativeOffset)

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}
	})

	t.Run("FileWriteAtAfterEndOfFile", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

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
		path := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, len(data)*2)
		n, err := f.Write(b)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("write").Path(path).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("write").Path(path).Err(avfs.ErrBadFileDesc)
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
			CheckPathError(t, err).Op("write").Path(path).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("write").Path(path).Err(avfs.ErrBadFileDesc)
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
		f, err := vfs.Open(testDir)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Write(b)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("write").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			CheckPathError(t, err).Op("write").Path(testDir).Err(avfs.ErrBadFileDesc)
		}

		_, err = f.WriteAt(b, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("write").Path(testDir).Err(avfs.ErrWinInvalidHandle)
		default:
			CheckPathError(t, err).Op("write").Path(testDir).Err(avfs.ErrBadFileDesc)
		}
	})

	t.Run("FileWriteClosed", func(t *testing.T) {
		b := make([]byte, 1)

		f, fileName := sfs.ClosedFile(t, testDir)
		_, err := f.Write(b)
		CheckPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed)

		_, err = f.WriteAt(b, 0)
		CheckPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileWriteString tests File.WriteString function.
func (sfs *SuiteFS) TestFileWriteString(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		CheckPathError(t, err).Op("write").Path(avfs.NotImplemented).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("FileWriteNonExisting", func(t *testing.T) {
		f := sfs.OpenedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		CheckInvalid(t, "WriteString", err)
	})
}

// TestFileWriteTime checks that modification time is updated on write operations.
func (sfs *SuiteFS) TestFileWriteTime(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	var start, end int64

	data := []byte("AAABBBCCCDDD")
	existingFile := vfs.Join(testDir, defaultFile)

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
