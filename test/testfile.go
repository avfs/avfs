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
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// TestFileChdir tests File.Chdir function.
func (sfs *SuiteFS) TestFileChdir(t *testing.T, testDir string) {
	dirs := sfs.createSampleDirs(t, testDir)
	vfs := sfs.vfsTest

	t.Run("FileChdir", func(t *testing.T) {
		for _, dir := range dirs {
			f, err := vfs.OpenFile(dir.Path, os.O_RDONLY, 0)
			RequireNoError(t, err, "OpenFile %s", dir.Path)

			defer f.Close()

			err = f.Chdir()
			switch vfs.OSType() {
			case avfs.OsWindows:
				CheckPathError(t, err).Op("chdir").Path(dir.Path).Err(avfs.ErrWinNotSupported)

				continue
			default:
				AssertNoError(t, err, "Chdir %s", dir.Path)
			}

			curDir, err := vfs.Getwd()
			AssertNoError(t, err, "Getwd %s", dir.Path)

			if curDir != dir.Path {
				t.Errorf("Getwd : want current directory to be %s, got %s", dir.Path, curDir)
			}
		}
	})

	t.Run("FileChdirOnFile", func(t *testing.T) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chdir()
		CheckPathError(t, err).Op("chdir").Path(fileName).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)
	})

	t.Run("FileChdirClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		err := f.Chdir()
		CheckPathError(t, err).Op("chdir").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChdirNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Chdir()
		AssertInvalid(t, err, "Chdir")
	})
}

// TestFileChmod tests File.Chmod function.
func (sfs *SuiteFS) TestFileChmod(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chmod(0)
		CheckPathError(t, err).Op("chmod").Path(fileName).ErrPermDenied()

		return
	}

	t.Run("FileChmodClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		err := f.Chmod(avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("chmod").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChmodNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Chmod(0)
		AssertInvalid(t, err, "Chmod")
	})
}

// TestFileChown tests File.Chown function.
func (sfs *SuiteFS) TestFileChown(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(fileName).ErrPermDenied()

		return
	}

	t.Run("FileChown", func(t *testing.T) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		u := vfs.User()
		uid, gid := u.Uid(), u.Gid()

		err := f.Chown(uid, gid)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("chown").Path(f.Name()).Err(avfs.ErrWinNotSupported, avfs.OsWindows)
		default:
			RequireNoError(t, err, "Chown %s", fileName)
		}
	})

	t.Run("FileChownClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		err := f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileChownNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Chown(0, 0)
		AssertInvalid(t, err, "Chown")
	})
}

// TestFileCloseRead tests File.Close function for read only files.
func (sfs *SuiteFS) TestFileCloseRead(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := sfs.existingFile(t, testDir, data)
	vfs := sfs.vfsTest

	t.Run("FileCloseReadOnly", func(t *testing.T) {
		openInfo, err := vfs.Stat(path)
		RequireNoError(t, err, "Stat %s", path)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		err = f.Close()
		RequireNoError(t, err, "Close %s", path)

		closeInfo, err := vfs.Stat(path)
		RequireNoError(t, err, "Stat %s", path)

		if !reflect.DeepEqual(openInfo, closeInfo) {
			t.Errorf("Stat %s : open info != close info\n%v\n%v", path, openInfo, closeInfo)
		}

		err = f.Close()
		CheckPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed)
	})

	t.Run("FileCloseNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Close()
		AssertInvalid(t, err, "Close")
	})
}

// TestFileCloseWrite tests File.Close function for read/write files.
func (sfs *SuiteFS) TestFileCloseWrite(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	if vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := sfs.existingFile(t, testDir, data)

	openInfo, err := vfs.Stat(path)
	RequireNoError(t, err, "Stat %s", path)

	t.Run("FileCloseWrite", func(t *testing.T) {
		f, err := vfs.OpenFile(path, os.O_APPEND|os.O_WRONLY, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		n, err := f.Write(data)
		RequireNoError(t, err, "Write %s", path)

		if n != len(data) {
			t.Fatalf("Write : want bytes written to be %d, got %d", len(data), n)
		}

		err = f.Close()
		RequireNoError(t, err, "Close %s", path)

		closeInfo, err := vfs.Stat(path)
		RequireNoError(t, err, "Stat %s", path)

		if reflect.DeepEqual(openInfo, closeInfo) {
			t.Errorf("Stat %s : open info != close info\n%v\n%v", path, openInfo, closeInfo)
		}

		err = f.Close()
		CheckPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed)
	})
}

// TestFileFd tests File.Fd function.
func (sfs *SuiteFS) TestFileFd(t *testing.T, testDir string) {
	f, fileName := sfs.closedFile(t, testDir)

	fd := f.Fd()
	if fd != ^(uintptr(0)) {
		t.Errorf("Fd %s : want Fd to be %d, got %d", fileName, ^(uintptr(0)), fd)
	}
}

// TestFileName tests File.Name function.
func (sfs *SuiteFS) TestFileName(t *testing.T, testDir string) {
	f, wantName := sfs.closedFile(t, testDir)

	name := f.Name()
	if name != wantName {
		t.Errorf("Name %s : want Name to be %s, got %s", wantName, wantName, name)
	}
}

// FileNilPtr test calls to File methods when f is a nil File.
func FileNilPtr(t *testing.T, f avfs.File) {
	err := f.Chdir()
	AssertInvalid(t, err, "Chdir")

	err = f.Chmod(0)
	AssertInvalid(t, err, "Chmod")

	err = f.Chown(0, 0)
	AssertInvalid(t, err, "Chown")

	err = f.Close()
	AssertInvalid(t, err, "Close")

	CheckPanic(t, "f.Name()", func() { _ = f.Name() })

	fd := f.Fd()
	if fd != ^(uintptr(0)) {
		t.Errorf("Fd : want fd to be %d, got %d", 0, fd)
	}

	_, err = f.Read([]byte{})
	AssertInvalid(t, err, "Read")

	_, err = f.ReadAt([]byte{}, 0)
	AssertInvalid(t, err, "ReadAt")

	_, err = f.ReadDir(0)
	AssertInvalid(t, err, "ReadDir")

	_, err = f.Readdirnames(0)
	AssertInvalid(t, err, "Readdirnames")

	_, err = f.Seek(0, io.SeekStart)
	AssertInvalid(t, err, "Seek")

	_, err = f.Stat()
	AssertInvalid(t, err, "Stat")

	err = f.Sync()
	AssertInvalid(t, err, "Sync")

	err = f.Truncate(0)
	AssertInvalid(t, err, "Truncate")

	_, err = f.Write([]byte{})
	AssertInvalid(t, err, "Write")

	_, err = f.WriteAt([]byte{}, 0)
	AssertInvalid(t, err, "WriteAt")

	_, err = f.WriteString("")
	AssertInvalid(t, err, "WriteString")
}

// TestFileRead tests File.Read function.
func (sfs *SuiteFS) TestFileRead(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := sfs.existingFile(t, testDir, data)
	vfs := sfs.vfsTest

	t.Run("FileRead", func(t *testing.T) {
		const bufSize = 5

		f, err := vfs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

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
		f := sfs.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.Read(buf)
		AssertInvalid(t, err, "Read")
	})

	t.Run("FileReadOnDir", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Read(b)
		CheckPathError(t, err).Op("read").Path(testDir).
			Err(avfs.ErrIsADirectory, avfs.OsLinux).
			Err(avfs.ErrWinIncorrectFunc, avfs.OsWindows)
	})

	t.Run("FileReadClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		b := make([]byte, 1)

		_, err := f.Read(b)
		CheckPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileReadAt tests File.ReadAt function.
func (sfs *SuiteFS) TestFileReadAt(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := sfs.existingFile(t, testDir, data)
	vfs := sfs.vfsTest

	t.Run("FileReadAt", func(t *testing.T) {
		const bufSize = 3

		f, err := vfs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		var n int
		rb := make([]byte, bufSize)
		for i := len(data); i > 0; i -= bufSize {
			n, err = f.ReadAt(rb, int64(i-bufSize))
			RequireNoError(t, err, "ReadAt %s", path)

			if n != bufSize {
				t.Errorf("ReadAt : want bytes read to be %d, got %d", bufSize, n)
			}

			if !bytes.Equal(rb, data[i-bufSize:i]) {
				t.Errorf("ReadAt : want bytes read to be %d, got %d", bufSize, n)
			}
		}
	})

	t.Run("FileReadAtNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.ReadAt(buf, 0)
		AssertInvalid(t, err, "ReadAt")
	})

	t.Run("FileReadAtAfterEndOfFile", func(t *testing.T) {
		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

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
		CheckPathError(t, err).Op("readat").Path(path).Err(avfs.ErrNegativeOffset)

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}
	})

	t.Run("FileReadAtOnDir", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.ReadAt(b, 0)
		CheckPathError(t, err).Op("read").Path(testDir).
			Err(avfs.ErrIsADirectory, avfs.OsLinux).
			Err(avfs.ErrWinIncorrectFunc, avfs.OsWindows)
	})

	t.Run("FileReadAtClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		b := make([]byte, 1)

		_, err := f.ReadAt(b, 0)
		CheckPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileReadDir tests File.ReadDir function.
func (sfs *SuiteFS) TestFileReadDir(t *testing.T, testDir string) {
	rndTree := sfs.randomDir(t, testDir)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)
	vfs := sfs.vfsTest

	const maxRead = 7

	t.Run("FileReadDirN", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		var dirEntries []fs.DirEntry

		for {
			dirEntriesN, err := f.ReadDir(maxRead)
			if err == io.EOF {
				break
			}

			RequireNoError(t, err, "ReadDir %s", testDir)

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
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		_, err := f.ReadDir(-1)
		CheckPathError(t, err).Path(fileName).
			Op("readdirent", avfs.OsLinux).Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Op("readdir", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("FileReadDirClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		_, err := f.ReadDir(-1)
		CheckPathError(t, err).Path(fileName).
			Op("readdirent", avfs.OsLinux).Err(avfs.ErrFileClosing, avfs.OsLinux).
			Op("readdir", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("FileReadDirNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.ReadDir(-1)
		AssertInvalid(t, err, "ReadDir")
	})
}

// TestFileReaddirnames tests File.Readdirnames function.
func (sfs *SuiteFS) TestFileReaddirnames(t *testing.T, testDir string) {
	rndTree := sfs.randomDir(t, testDir)
	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)
	existingFile := rndTree.Files[0].Name
	vfs := sfs.vfsTest

	t.Run("FileReaddirnamesAll", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		names, err := f.Readdirnames(-1)
		RequireNoError(t, err, "Readdirnames %s", testDir)

		if wAll != len(names) {
			t.Errorf("TestFileReaddirnames : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("FileReaddirnamesN", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		var names []string

		for {
			namesN, err := f.Readdirnames(11)
			if err == io.EOF {
				break
			}

			RequireNoError(t, err, "ReadDirNamesN %s", testDir)

			names = append(names, namesN...)
		}

		if wAll != len(names) {
			t.Errorf("ReadDirNamesN : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("FileReaddirnamesExistingFile", func(t *testing.T) {
		f, err := vfs.OpenFile(existingFile, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		defer f.Close()

		_, err = f.Readdirnames(-1)
		CheckPathError(t, err).Path(f.Name()).
			Op("readdirent", avfs.OsLinux).Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Op("readdir", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("FileReaddirnamesClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		_, err := f.Readdirnames(-1)
		CheckPathError(t, err).Path(fileName).
			Op("readdirent", avfs.OsLinux).Err(avfs.ErrFileClosing, avfs.OsLinux).
			Op("readdir", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("FileReaddirnamesNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.Readdirnames(-1)
		AssertInvalid(t, err, "Readdirnames")
	})
}

// TestFileSeek tests File.Seek function.
func (sfs *SuiteFS) TestFileSeek(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := sfs.existingFile(t, testDir, data)
	vfs := sfs.vfsTest

	f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
	RequireNoError(t, err, "OpenFile %s", path)

	defer f.Close()

	pos := int64(0)
	lenData := int64(len(data))

	t.Run("TestFileSeek", func(t *testing.T) {
		for i := 0; i < len(data); i++ {
			pos, err = f.Seek(int64(i), io.SeekStart)
			RequireNoError(t, err, "Seek %s", path)

			if int(pos) != i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}

		for i := 0; i < len(data); i++ {
			pos, err = f.Seek(-int64(i), io.SeekEnd)
			RequireNoError(t, err, "Seek %s", path)

			if int(pos) != len(data)-i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}

		_, err = f.Seek(0, io.SeekEnd)
		RequireNoError(t, err, "Seek %s", path)

		for i := len(data) - 1; i >= 0; i-- {
			pos, err = f.Seek(-1, io.SeekCurrent)
			RequireNoError(t, err, "Seek %s", path)

			if int(pos) != i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}
	})

	t.Run("FileSeekInvalidStart", func(t *testing.T) {
		pos, err = f.Seek(-1, io.SeekStart)
		CheckPathError(t, err).Op("seek").Path(f.Name()).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinNegativeSeek, avfs.OsWindows)

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}

		wantPos := lenData * 2

		pos, err = f.Seek(wantPos, io.SeekStart)
		RequireNoError(t, err, "Seek %s", path)

		if pos != wantPos {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}
	})

	t.Run("FileSeekInvalidEnd", func(t *testing.T) {
		pos, err = f.Seek(1, io.SeekEnd)
		RequireNoError(t, err, "Seek %s", path)

		wantPos := lenData + 1
		if pos != wantPos {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}

		pos, err = f.Seek(-lenData*2, io.SeekEnd)
		CheckPathError(t, err).Op("seek").Path(f.Name()).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinNegativeSeek, avfs.OsWindows)

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
		CheckPathError(t, err).Op("seek").Path(f.Name()).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinNegativeSeek, avfs.OsWindows)

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}

		pos, err = f.Seek(lenData, io.SeekCurrent)
		RequireNoError(t, err, "Seek %s", path)

		if pos != lenData/2+lenData {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}
	})

	t.Run("FileSeekInvalidWhence", func(t *testing.T) {
		pos, err = f.Seek(0, 10)

		switch vfs.OSType() {
		case avfs.OsWindows:
			RequireNoError(t, err, "Seek %s", path)
		default:
			CheckPathError(t, err).Op("seek").Path(f.Name()).
				Err(avfs.ErrInvalidArgument, avfs.OsLinux)
		}

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}
	})

	t.Run("FileSeekOnDir", func(t *testing.T) {
		f, err = vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		_, err = f.Seek(0, io.SeekStart)
		RequireNoError(t, err, "Seek %s", testDir)
	})

	t.Run("FileSeekClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		CheckPathError(t, err).Op("seek").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileSeekNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		AssertInvalid(t, err, "Seek")
	})
}

// TestFileStat tests File.Stat function.
func (sfs *SuiteFS) TestFileStat(t *testing.T, testDir string) {
	dirs := sfs.createSampleDirs(t, testDir)
	files := sfs.createSampleFiles(t, testDir)
	_ = sfs.CreateSampleSymlinks(t, testDir)
	vfs := sfs.vfsTest

	t.Run("FileStatDir", func(t *testing.T) {
		for _, dir := range dirs {
			f, err := vfs.OpenFile(dir.Path, os.O_RDONLY, 0)
			RequireNoError(t, err, "OpenFile %s", testDir)

			info, err := f.Stat()
			if !AssertNoError(t, err, "Stat %s", dir.Path) {
				_ = f.Close()

				continue
			}

			if vfs.Base(dir.Path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", dir.Path, vfs.Base(dir.Path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", dir.Path, wantMode, info.Mode())
			}

			_ = f.Close()
		}
	})

	t.Run("FileStatFile", func(t *testing.T) {
		for _, file := range files {
			f, err := vfs.OpenFile(file.Path, os.O_RDONLY, 0)
			RequireNoError(t, err, "OpenFile %s", file.Path)

			info, err := f.Stat()
			if !AssertNoError(t, err, "Stat %s", file.Path) {
				_ = f.Close()

				continue
			}

			if info.Name() != vfs.Base(file.Path) {
				t.Errorf("Stat %s : want name to be %s, got %s", file.Path, vfs.Base(file.Path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = avfs.DefaultFilePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", file.Path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", file.Path, wantSize, info.Size())
			}

			_ = f.Close()
		}
	})

	t.Run("FileStatSymlink", func(t *testing.T) {
		for _, sl := range sfs.sampleSymlinksEval(testDir) {
			f, err := vfs.OpenFile(sl.NewPath, os.O_RDONLY, 0)
			RequireNoError(t, err, "OpenFile %s", sl.NewPath)

			info, err := f.Stat()
			if !AssertNoError(t, err, "Stat %s", sl.NewPath) {
				_ = f.Close()

				continue
			}

			wantName := vfs.Base(sl.OldPath)
			if sl.IsSymlink {
				wantName = vfs.Base(sl.NewPath)
			}

			wantMode := sl.Mode
			if wantName != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", sl.NewPath, wantName, info.Name())
			}

			if sfs.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", sl.NewPath, wantMode, info.Mode())
			}

			_ = f.Close()
		}
	})

	t.Run("FileStatNonExistingFile", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.Stat()
		AssertInvalid(t, err, "Stat")
	})

	t.Run("FileStatSubDirOnFile", func(t *testing.T) {
		path := vfs.Join(files[0].Path, defaultNonExisting)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		CheckPathError(t, err).Op("open").Path(path).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)

		_, err = f.Stat()
		AssertInvalid(t, err, "Stat")
	})

	t.Run("FileStatClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		_, err := f.Stat()
		CheckPathError(t, err).Path(fileName).
			Err(avfs.ErrFileClosing, avfs.OsLinux).
			Err(avfs.ErrWinInvalidHandle, avfs.OsWindows).
			Op("stat", avfs.OsLinux).
			Op("GetFileType", avfs.OsWindows)
	})

	t.Run("FileStatNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.Stat()
		AssertInvalid(t, err, "Stat")
	})
}

// TestFileSync tests File.Sync function.
func (sfs *SuiteFS) TestFileSync(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Sync()
		CheckPathError(t, err).Op("sync").Path(avfs.NotImplemented).ErrPermDenied()

		return
	}

	t.Run("FileSyncClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		err := f.Sync()
		CheckPathError(t, err).Op("sync").Path(fileName).Err(fs.ErrClosed)
	})

	t.Run("FileSyncNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Sync()
		AssertInvalid(t, err, "Sync")
	})
}

// TestFileTruncate tests File.Truncate function.
func (sfs *SuiteFS) TestFileTruncate(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(fileName).ErrPermDenied()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileTruncate", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, len(data))
		for i := len(data) - 1; i >= 0; i-- {
			err = f.Truncate(int64(i))
			RequireNoError(t, err, "Truncate %s", path)

			_, err = f.ReadAt(b, 0)
			if err != io.EOF {
				t.Errorf("Read : want error to be EOF, got %v", err)
			}

			if !bytes.Equal(data[:i], b[:i]) {
				t.Errorf("Truncate : want data to be %s, got %s", data[:i], b[:i])
			}
		}
	})

	t.Run("FileTruncateOnDir", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		err = f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(testDir).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})

	t.Run("FileTruncateSizeNegative", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		err = f.Truncate(-1)
		CheckPathError(t, err).Op("truncate").Path(path).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinNegativeSeek, avfs.OsWindows)
	})

	t.Run("FileTruncateSizeBiggerFileSize", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		newSize := len(data) * 2

		err = f.Truncate(int64(newSize))
		RequireNoError(t, err, "Truncate %s", path)

		info, err := f.Stat()
		RequireNoError(t, err, "Stat %s", path)

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		f.Close()

		gotContent, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		wantAdded := bytes.Repeat([]byte{0}, len(data))
		gotAdded := gotContent[len(data):]
		if !bytes.Equal(wantAdded, gotAdded) {
			t.Errorf("Bytes Added : want %v, got %v", wantAdded, gotAdded)
		}
	})

	t.Run("FileTruncateNonExistingFile", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		err := f.Truncate(0)
		AssertInvalid(t, err, "Truncate")
	})

	t.Run("FileTruncateClosed", func(t *testing.T) {
		f, fileName := sfs.closedFile(t, testDir)

		err := f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileWrite tests File.Write function.
func (sfs *SuiteFS) TestFileWrite(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		defer f.Close()

		_, err := f.Write([]byte{})
		CheckPathError(t, err).Op("write").Path(fileName).ErrPermDenied()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileWrite", func(t *testing.T) {
		path := vfs.Join(testDir, "TestFileWrite.txt")

		f, err := vfs.Create(path)
		RequireNoError(t, err, "Create %s", path)

		defer f.Close()

		for i := 0; i < len(data); i += 3 {
			buf3 := data[i : i+3]
			var n int

			n, err = f.Write(buf3)
			RequireNoError(t, err, "Write %s", path)

			if len(buf3) != n {
				t.Errorf("Write : want bytes written to be %d, got %d", len(buf3), n)
			}
		}

		rb, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("FileWriteNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.Write(buf)
		AssertInvalid(t, err, "Write")
	})

	t.Run("FileWriteReadOnly", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, len(data)*2)
		n, err := f.Write(b)
		CheckPathError(t, err).Op("write").Path(path).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.Read(b)
		RequireNoError(t, err, "Read %s", path)

		if !bytes.Equal(data, b[:n]) {
			t.Errorf("Read : want data to be %s, got %s", data, b[:n])
		}
	})

	t.Run("FileWriteOnDir", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.Write(b)
		CheckPathError(t, err).Op("write").Path(testDir).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})

	t.Run("FileWriteClosed", func(t *testing.T) {
		b := make([]byte, 1)

		f, fileName := sfs.closedFile(t, testDir)
		_, err := f.Write(b)
		CheckPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileWriteAt tests File.WriteAt function.
func (sfs *SuiteFS) TestFileWriteAt(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := sfs.openedEmptyFile(t, testDir)

		_, err := f.WriteAt([]byte{}, 0)
		CheckPathError(t, err).Op("write").Path(fileName).ErrPermDenied()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileWriteAt", func(t *testing.T) {
		path := vfs.Join(testDir, "TestFileWriteAt.txt")

		f, err := vfs.OpenFile(path, os.O_CREATE|os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		for i := len(data); i > 0; i -= 3 {
			var n int
			n, err = f.WriteAt(data[i-3:i], int64(i-3))
			RequireNoError(t, err, "WriteAt %s", path)

			if n != 3 {
				t.Errorf("WriteAt : want bytes written to be %d, got %d", 3, n)
			}
		}

		err = f.Close()
		RequireNoError(t, err, "Close %s", path)

		rb, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("FileWriteAtNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.WriteAt(buf, 0)
		AssertInvalid(t, err, "WriteAt")
	})

	t.Run("FileWriteAtNegativeOffset", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultDirPerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		n, err := f.WriteAt(data, -1)
		CheckPathError(t, err).Op("writeat").Path(path).Err(avfs.ErrNegativeOffset)

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}
	})

	t.Run("FileWriteAtAfterEndOfFile", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		off := int64(len(data) * 3)

		n, err := f.WriteAt(data, off)
		RequireNoError(t, err, "WriteAt %s", path)

		if n != len(data) {
			t.Errorf("WriteAt : want bytes written to be %d, got %d", len(data), n)
		}

		want := make([]byte, int(off)+len(data))
		_ = copy(want, data)
		_ = copy(want[off:], data)

		got, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(want, got) {
			t.Errorf("want : %s\ngot  : %s", want, got)
		}
	})

	t.Run("FileWriteAtReadOnly", func(t *testing.T) {
		path := sfs.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, len(data)*2)

		n, err := f.WriteAt(b, 0)
		CheckPathError(t, err).Op("write").Path(path).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("WriteAt : want bytes read to be 0, got %d", n)
		}

		n, err = f.ReadAt(b, 0)
		if err != io.EOF {
			t.Errorf("ReadAt : want error to be EOF, got %v", err)
		}

		if !bytes.Equal(data, b[:n]) {
			t.Errorf("ReadAt : want data to be %s, got %s", data, b[:n])
		}
	})

	t.Run("FileWriteAtOnDir", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		b := make([]byte, 1)

		_, err = f.WriteAt(b, 0)
		CheckPathError(t, err).Op("write").Path(testDir).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})

	t.Run("FileWriteAtClosed", func(t *testing.T) {
		b := make([]byte, 1)

		f, fileName := sfs.closedFile(t, testDir)

		_, err := f.WriteAt(b, 0)
		CheckPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed)
	})
}

// TestFileWriteString tests File.WriteString function.
func (sfs *SuiteFS) TestFileWriteString(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		CheckPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied()

		return
	}

	t.Run("FileWriteNonExisting", func(t *testing.T) {
		f := sfs.openedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		AssertInvalid(t, err, "WriteString")
	})
}

// TestFileWriteTime checks that modification time is updated on write operations.
func (sfs *SuiteFS) TestFileWriteTime(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	if vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	var previous time.Time

	f, fileName := sfs.openedEmptyFile(t, testDir)

	// CompareTime tests if the modification time of the file has changed.
	compareTime := func(mustChange bool) {
		time.Sleep(10 * time.Millisecond)

		info, err := vfs.Stat(fileName)
		if err != nil {
			t.Fatalf("Stat %s : want error to be nil, got %v", fileName, err)
		}

		// Don't compare for the first time.
		if previous.IsZero() {
			previous = info.ModTime()

			return
		}

		current := info.ModTime()
		if mustChange && !current.After(previous) {
			t.Errorf("CompareTime : want previous < current time\ngot prev = %v, curr = %v", previous, current)
		}

		if !mustChange && !current.Equal(previous) {
			t.Errorf("CompareTime : want previous = current time\ngot prev = %v, curr = %v", previous, current)
		}

		previous = current
	}

	compareTime(true)

	data := []byte("AAABBBCCCDDD")

	t.Run("TimeWrite", func(t *testing.T) {
		_, err := f.Write(data)
		RequireNoError(t, err, "Write")

		compareTime(true)
	})

	t.Run("TimeWriteAt", func(t *testing.T) {
		_, err := f.WriteAt(data, 5)
		RequireNoError(t, err, "WriteAt")

		compareTime(true)
	})

	t.Run("TimeTruncate", func(t *testing.T) {
		err := f.Truncate(5)
		RequireNoError(t, err, "Truncate")

		compareTime(true)
	})

	t.Run("TimeClose", func(t *testing.T) {
		err := f.Close()
		RequireNoError(t, err, "Close")

		compareTime(false)
	})
}
