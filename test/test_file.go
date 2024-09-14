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

func (ts *Suite) TestFile(t *testing.T) {
	ts.RunTests(t, UsrTest,
		ts.TestFileChdir,
		ts.TestFileCloseWrite,
		ts.TestFileCloseRead,
		ts.TestFileFd,
		ts.TestFileName,
		ts.TestFileRead,
		ts.TestFileReadAt,
		ts.TestFileReadDir,
		ts.TestFileReaddirnames,
		ts.TestFileSeek,
		ts.TestFileStat,
		ts.TestFileSync,
		ts.TestFileTruncate,
		ts.TestFileWrite,
		ts.TestFileWriteAt,
		ts.TestFileWriteString,
		ts.TestFileWriteTime)

	// Tests to be run as root
	adminUser := ts.idm.AdminUser()
	ts.RunTests(t, adminUser.Name(),
		ts.TestFileChmod,
		ts.TestFileChown,
	)
}

// TestFileChdir tests File.Chdir function.
func (ts *Suite) TestFileChdir(t *testing.T, testDir string) {
	dirs := ts.createSampleDirs(t, testDir)
	vfs := ts.vfsTest

	t.Run("FileChdir", func(t *testing.T) {
		for _, dir := range dirs {
			f, err := vfs.OpenFile(dir.Path, os.O_RDONLY, 0)
			RequireNoError(t, err, "OpenFile %s", dir.Path)

			defer f.Close()

			err = f.Chdir()
			AssertNoError(t, err, "Chdir %s", dir.Path)

			curDir, err := vfs.Getwd()
			AssertNoError(t, err, "Getwd %s", dir.Path)

			if curDir != dir.Path {
				t.Errorf("Getwd : want current directory to be %s, got %s", dir.Path, curDir)
			}
		}
	})

	t.Run("FileChdirOnFile", func(t *testing.T) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chdir()
		AssertPathError(t, err).Op("chdir").Path(fileName).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinDirNameInvalid).Test()
	})

	t.Run("FileChdirClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		err := f.Chdir()
		AssertPathError(t, err).Op("chdir").Path(fileName).Err(fs.ErrClosed).Test()
	})

	t.Run("FileChdirNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Chdir()
		AssertInvalid(t, err, "Chdir")
	})
}

// TestFileChmod tests File.Chmod function.
func (ts *Suite) TestFileChmod(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chmod(0)
		AssertPathError(t, err).Op("chmod").Path(fileName).ErrPermDenied().Test()

		return
	}

	t.Run("FileChmodClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		err := f.Chmod(avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("chmod").Path(fileName).Err(fs.ErrClosed).Test()
	})

	t.Run("FileChmodNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Chmod(0)
		AssertInvalid(t, err, "Chmod")
	})
}

// TestFileChown tests File.Chown function.
func (ts *Suite) TestFileChown(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Chown(0, 0)
		AssertPathError(t, err).Op("chown").Path(fileName).ErrPermDenied().Test()

		return
	}

	t.Run("FileChown", func(t *testing.T) {
		f, _ := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		u := vfs.User()
		uid, gid := u.Uid(), u.Gid()

		err := f.Chown(uid, gid)

		AssertPathError(t, err).Op("chown").
			OSType(avfs.OsLinux).NoError().Test().
			OSType(avfs.OsWindows).Path(f.Name()).Err(avfs.ErrWinNotSupported).Test()
	})

	t.Run("FileChownClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		err := f.Chown(0, 0)
		AssertPathError(t, err).Op("chown").Path(fileName).Err(fs.ErrClosed).Test()
	})

	t.Run("FileChownNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Chown(0, 0)
		AssertInvalid(t, err, "Chown")
	})
}

// TestFileCloseRead tests File.Close function for read only files.
func (ts *Suite) TestFileCloseRead(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := ts.existingFile(t, testDir, data)
	vfs := ts.vfsTest

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
		AssertPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed).Test()
	})

	t.Run("FileCloseNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Close()
		AssertInvalid(t, err, "Close")
	})
}

// TestFileCloseWrite tests File.Close function for read/write files.
func (ts *Suite) TestFileCloseWrite(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	if vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := ts.existingFile(t, testDir, data)

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
		AssertPathError(t, err).Op("close").Path(path).Err(fs.ErrClosed).Test()
	})
}

// TestFileFd tests File.Fd function.
func (ts *Suite) TestFileFd(t *testing.T, testDir string) {
	f, fileName := ts.closedFile(t, testDir)

	fd := f.Fd()
	if fd != ^(uintptr(0)) {
		t.Errorf("Fd %s : want Fd to be %d, got %d", fileName, ^(uintptr(0)), fd)
	}
}

// TestFileName tests File.Name function.
func (ts *Suite) TestFileName(t *testing.T, testDir string) {
	f, wantName := ts.closedFile(t, testDir)

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

	AssertPanic(t, "f.Name()", func() { _ = f.Name() })

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
func (ts *Suite) TestFileRead(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := ts.existingFile(t, testDir, data)
	vfs := ts.vfsTest

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
		f := ts.openedNonExistingFile(t, testDir)
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
		AssertPathError(t, err).Op("read").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrIsADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinIncorrectFunc).Test()
	})

	t.Run("FileReadClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		b := make([]byte, 1)

		_, err := f.Read(b)
		AssertPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed).Test()
	})
}

// TestFileReadAt tests File.ReadAt function.
func (ts *Suite) TestFileReadAt(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := ts.existingFile(t, testDir, data)
	vfs := ts.vfsTest

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
		f := ts.openedNonExistingFile(t, testDir)
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
		AssertPathError(t, err).Op("readat").Path(path).Err(avfs.ErrNegativeOffset).Test()

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
		AssertPathError(t, err).Op("read").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrIsADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinIncorrectFunc).Test()
	})

	t.Run("FileReadAtClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		b := make([]byte, 1)

		_, err := f.ReadAt(b, 0)
		AssertPathError(t, err).Op("read").Path(fileName).Err(fs.ErrClosed).Test()
	})
}

// TestFileReadDir tests File.ReadDir function.
func (ts *Suite) TestFileReadDir(t *testing.T, testDir string) {
	rndTree := ts.randomDir(t, testDir)
	wantDirs := len(rndTree.Dirs())
	wantFiles := len(rndTree.Files())
	wantSymlinks := len(rndTree.SymLinks())
	vfs := ts.vfsTest

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

		var gotDirs, gotFiles, gotSymlinks int

		for _, dirEntry := range dirEntries {
			switch {
			case dirEntry.IsDir():
				gotDirs++
			case dirEntry.Type()&fs.ModeSymlink != 0:
				gotSymlinks++
			default:
				gotFiles++
			}
		}

		if wantDirs != gotDirs {
			t.Errorf("ReadDirN : want number of dirs to be %d, got %d", wantDirs, gotDirs)
		}

		if wantFiles != gotFiles {
			t.Errorf("ReadDirN : want number of files to be %d, got %d", wantFiles, gotFiles)
		}

		if wantSymlinks != gotSymlinks {
			t.Errorf("ReadDirN : want number of symbolic links to be %d, got %d", wantSymlinks, gotSymlinks)
		}
	})

	t.Run("FileReadDirExistingFile", func(t *testing.T) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		_, err := f.ReadDir(-1)
		AssertPathError(t, err).Path(fileName).
			OSType(avfs.OsLinux).Op("readdirent").Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Op("readdir").Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("FileReadDirClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		_, err := f.ReadDir(-1)
		AssertPathError(t, err).Path(fileName).
			OSType(avfs.OsLinux).Op("readdirent").Err(avfs.ErrFileClosing).Test().
			OSType(avfs.OsWindows).Op("readdir").Err(avfs.ErrWinInvalidHandle).Test()
	})

	t.Run("FileReadDirNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.ReadDir(-1)
		AssertInvalid(t, err, "ReadDir")
	})
}

// TestFileReaddirnames tests File.Readdirnames function.
func (ts *Suite) TestFileReaddirnames(t *testing.T, testDir string) {
	rndTree := ts.randomDir(t, testDir)
	wantAll := len(rndTree.Dirs()) + len(rndTree.Files()) + len(rndTree.SymLinks())

	vfs := ts.vfsTest
	existingFile := vfs.Join(testDir, rndTree.Files()[0].Name)

	t.Run("FileReaddirnamesAll", func(t *testing.T) {
		f, err := vfs.OpenFile(testDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", testDir)

		defer f.Close()

		names, err := f.Readdirnames(-1)
		RequireNoError(t, err, "Readdirnames %s", testDir)

		if wantAll != len(names) {
			t.Errorf("TestFileReaddirnames : want number of elements to be %d, got %d", wantAll, len(names))
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

		gotAll := len(names)
		if len(names) != wantAll {
			t.Errorf("ReadDirNamesN : want number of elements to be %d, got %d", wantAll, gotAll)
		}
	})

	t.Run("FileReaddirnamesExistingFile", func(t *testing.T) {
		f, err := vfs.OpenFile(existingFile, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		defer f.Close()

		_, err = f.Readdirnames(-1)
		AssertPathError(t, err).Path(f.Name()).
			OSType(avfs.OsLinux).Op("readdirent").Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Op("readdir").Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("FileReaddirnamesClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		_, err := f.Readdirnames(-1)
		AssertPathError(t, err).Path(fileName).
			OSType(avfs.OsLinux).Op("readdirent").Err(avfs.ErrFileClosing).Test().
			OSType(avfs.OsWindows).Op("readdir").Err(avfs.ErrWinInvalidHandle).Test()
	})

	t.Run("FileReaddirnamesNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.Readdirnames(-1)
		AssertInvalid(t, err, "Readdirnames")
	})
}

// TestFileSeek tests File.Seek function.
func (ts *Suite) TestFileSeek(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	path := ts.existingFile(t, testDir, data)
	vfs := ts.vfsTest

	f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
	RequireNoError(t, err, "OpenFile %s", path)

	defer f.Close()

	pos := int64(0)
	lenData := int64(len(data))

	t.Run("TestFileSeek", func(t *testing.T) {
		for i := range data {
			pos, err = f.Seek(int64(i), io.SeekStart)
			RequireNoError(t, err, "Seek %s", path)

			if int(pos) != i {
				t.Errorf("Seek : want position to be %d, got %d", i, pos)
			}
		}

		for i := range data {
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
		AssertPathError(t, err).Op("seek").Path(f.Name()).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNegativeSeek).Test()

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
		AssertPathError(t, err).Op("seek").Path(f.Name()).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNegativeSeek).Test()

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
		AssertPathError(t, err).Op("seek").Path(f.Name()).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNegativeSeek).Test()

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

		AssertPathError(t, err).Op("seek").Path(f.Name()).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).NoError().Test()

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
		f, fileName := ts.closedFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		AssertPathError(t, err).Op("seek").Path(fileName).Err(fs.ErrClosed).Test()
	})

	t.Run("FileSeekNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err = f.Seek(0, io.SeekStart)
		AssertInvalid(t, err, "Seek")
	})
}

// TestFileStat tests File.Stat function.
func (ts *Suite) TestFileStat(t *testing.T, testDir string) {
	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	_ = ts.createSampleSymlinks(t, testDir)
	vfs := ts.vfsTest

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
		for _, sl := range ts.sampleSymlinksEval(testDir) {
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

			if ts.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", sl.NewPath, wantMode, info.Mode())
			}

			_ = f.Close()
		}
	})

	t.Run("FileStatNonExistingFile", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.Stat()
		AssertInvalid(t, err, "Stat")
	})

	t.Run("FileStatSubDirOnFile", func(t *testing.T) {
		path := vfs.Join(files[0].Path, defaultNonExisting)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		AssertPathError(t, err).Op("open").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()

		_, err = f.Stat()
		AssertInvalid(t, err, "Stat")
	})

	t.Run("FileStatClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		_, err := f.Stat()
		AssertPathError(t, err).Path(fileName).
			OSType(avfs.OsLinux).Op("stat").GoVersion("go1.22").Err(avfs.ErrFileClosing).Test().
			// TODO: Add test for Go >= 1.23
			OSType(avfs.OsWindows).Op("GetFileType").Err(avfs.ErrWinInvalidHandle).Test()
	})

	t.Run("FileStatNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.Stat()
		AssertInvalid(t, err, "Stat")
	})
}

// TestFileSync tests File.Sync function.
func (ts *Suite) TestFileSync(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Sync()
		AssertPathError(t, err).Op("sync").Path(avfs.NotImplemented).ErrPermDenied().Test()

		return
	}

	t.Run("FileSyncClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		err := f.Sync()
		AssertPathError(t, err).Op("sync").Path(fileName).Err(fs.ErrClosed).Test()
	})

	t.Run("FileSyncNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Sync()
		AssertInvalid(t, err, "Sync")
	})
}

// TestFileTruncate tests File.Truncate function.
func (ts *Suite) TestFileTruncate(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		err := f.Truncate(0)
		AssertPathError(t, err).Op("truncate").Path(fileName).ErrPermDenied().Test()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("FileTruncate", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

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
		AssertPathError(t, err).Op("truncate").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("FileTruncateSizeNegative", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		err = f.Truncate(-1)
		AssertPathError(t, err).Op("truncate").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNegativeSeek).Test()
	})

	t.Run("FileTruncateSizeBiggerFileSize", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

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
		f := ts.openedNonExistingFile(t, testDir)

		err := f.Truncate(0)
		AssertInvalid(t, err, "Truncate")
	})

	t.Run("FileTruncateClosed", func(t *testing.T) {
		f, fileName := ts.closedFile(t, testDir)

		err := f.Truncate(0)
		AssertPathError(t, err).Op("truncate").Path(fileName).Err(fs.ErrClosed).Test()
	})
}

// TestFileWrite tests File.Write function.
func (ts *Suite) TestFileWrite(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		defer f.Close()

		_, err := f.Write([]byte{})
		AssertPathError(t, err).Op("write").Path(fileName).ErrPermDenied().Test()

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
		f := ts.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.Write(buf)
		AssertInvalid(t, err, "Write")
	})

	t.Run("FileWriteReadOnly", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, len(data)*2)
		n, err := f.Write(b)
		AssertPathError(t, err).Op("write").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

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
		AssertPathError(t, err).Op("write").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("FileWriteClosed", func(t *testing.T) {
		b := make([]byte, 1)

		f, fileName := ts.closedFile(t, testDir)
		_, err := f.Write(b)
		AssertPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed).Test()
	})
}

// TestFileWriteAt tests File.WriteAt function.
func (ts *Suite) TestFileWriteAt(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f, fileName := ts.openedEmptyFile(t, testDir)

		_, err := f.WriteAt([]byte{}, 0)
		AssertPathError(t, err).Op("write").Path(fileName).ErrPermDenied().Test()

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
		f := ts.openedNonExistingFile(t, testDir)
		buf := make([]byte, 0)

		_, err := f.WriteAt(buf, 0)
		AssertInvalid(t, err, "WriteAt")
	})

	t.Run("FileWriteAtNegativeOffset", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDWR, avfs.DefaultDirPerm)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		n, err := f.WriteAt(data, -1)
		AssertPathError(t, err).Op("writeat").Path(path).Err(avfs.ErrNegativeOffset).Test()

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}
	})

	t.Run("FileWriteAtAfterEndOfFile", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

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
		path := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", path)

		defer f.Close()

		b := make([]byte, len(data)*2)

		n, err := f.WriteAt(b, 0)
		AssertPathError(t, err).Op("write").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

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
		AssertPathError(t, err).Op("write").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("FileWriteAtClosed", func(t *testing.T) {
		b := make([]byte, 1)

		f, fileName := ts.closedFile(t, testDir)

		_, err := f.WriteAt(b, 0)
		AssertPathError(t, err).Op("write").Path(fileName).Err(fs.ErrClosed).Test()
	})
}

// TestFileWriteString tests File.WriteString function.
func (ts *Suite) TestFileWriteString(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		AssertPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied().Test()

		return
	}

	t.Run("FileWriteNonExisting", func(t *testing.T) {
		f := ts.openedNonExistingFile(t, testDir)

		_, err := f.WriteString("")
		AssertInvalid(t, err, "WriteString")
	})
}

// TestFileWriteTime checks that modification time is updated on write operations.
func (ts *Suite) TestFileWriteTime(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	if vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	var previous time.Time

	f, fileName := ts.openedEmptyFile(t, testDir)

	// CompareTime tests if the modification time of the file has changed.
	compareTime := func(mustChange bool) {
		time.Sleep(10 * time.Millisecond)

		info, err := vfs.Stat(fileName)
		RequireNoError(t, err, "Stat %s", fileName)

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
