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
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// SuiteOpenFileRead tests OpenFile function for read.
func (cf *ConfigFs) SuiteOpenFileRead() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	existingFile := fs.Join(rootDir, "ExistingFile.txt")

	err := fs.WriteFile(existingFile, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	existingDir := fs.Join(rootDir, "existingDir")

	err = fs.Mkdir(existingDir, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir : want error to be nil, got %v", err)
	}

	fs = cf.GetFsRead()

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		f, err := fs.Open(existingFile)
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		gotData, err := ioutil.ReadAll(f)
		if err != nil {
			t.Errorf("ReadAll : want error to be nil, got %v", err)
		}

		if !bytes.Equal(gotData, data) {
			t.Errorf("ReadAll : want error data to be %v, got %v", data, gotData)
		}
	})

	t.Run("OpenFileDirReadOnly", func(t *testing.T) {
		f, err := fs.OpenFile(existingDir, os.O_RDONLY, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		dirs, err := f.Readdir(-1)
		if err != nil {
			t.Errorf("Readdir : want error to be nil, got %v", err)
		}

		if len(dirs) != 0 {
			t.Errorf("Readdir : want number of directories to be 0, got %d", len(dirs))
		}
	})
}

// SuiteOpenFileWrite tests OpenFile function for write.
func (cf *ConfigFs) SuiteOpenFileWrite() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	whateverData := []byte("whatever")
	existingFile := fs.Join(rootDir, "ExistingFile.txt")
	buf3 := make([]byte, 3)

	err := fs.WriteFile(existingFile, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	t.Run("OpenFileWriteOnly", func(t *testing.T) {
		f, err := fs.OpenFile(existingFile, os.O_WRONLY, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.Write(whateverData)
		if err != nil {
			t.Errorf("Write : want error to be nil, got %v", err)
		}

		if n != len(whateverData) {
			t.Errorf("Write : want bytes written to be %d, got %d", len(whateverData), n)
		}

		n, err = f.Read(buf3)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Read", "read", existingFile, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "Read", "read", existingFile, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("Read : want bytes written to be 0, got %d", n)
		}

		n, err = f.ReadAt(buf3, 3)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadAt", "read", existingFile, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "ReadAt", "read", existingFile, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}

		err = f.Chmod(0o777)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chmod", "chmod", existingFile, avfs.ErrWinNotSupported, err)
		default:
			if err != nil {
				t.Errorf("Chmod : want error to be nil, got %v", err)
			}
		}

		if fs.HasFeature(avfs.FeatIdentityMgr) {
			u := fs.CurrentUser()
			err = f.Chown(u.Uid(), u.Gid())
			if err != nil {
				t.Errorf("Chown : want error to be nil, got %v", err)
			}
		}

		fst, err := f.Stat()
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		wantName := fs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		if err != nil {
			t.Errorf("Chmod : want error to be nil, got %v", err)
		}

		err = f.Sync()
		if err != nil {
			t.Errorf("Sync : want error to be nil, got %v", err)
		}
	})

	t.Run("OpenFileAppend", func(t *testing.T) {
		err := fs.WriteFile(existingFile, data, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("Chmod : want error to be nil, got %v", err)
		}

		f, err := fs.OpenFile(existingFile, os.O_WRONLY|os.O_APPEND, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		n, err := f.Write(whateverData)
		if err != nil {
			t.Errorf("Write : want error to be nil, got %v", err)
		}

		if n != len(whateverData) {
			t.Errorf("Write : want error to be %d, got %d", len(whateverData), n)
		}

		_ = f.Close()

		gotContent, err := fs.ReadFile(existingFile)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		wantContent := append(data, whateverData...)
		if !bytes.Equal(wantContent, gotContent) {
			t.Errorf("ReadAll : want content to be %s, got %s", wantContent, gotContent)
		}
	})

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		f, err := fs.Open(existingFile)
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.Write(whateverData)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Write", "write", existingFile, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "Write", "write", existingFile, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.WriteAt(whateverData, 3)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "WriteAt", "write", existingFile, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "WriteAt", "write", existingFile, avfs.ErrBadFileDesc, err)
		}

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}

		err = f.Chmod(0o777)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chmod", "chmod", existingFile, avfs.ErrWinNotSupported, err)
		default:
			if err != nil {
				t.Errorf("Chmod : want error to be nil, got %v", err)
			}
		}

		if fs.HasFeature(avfs.FeatIdentityMgr) {
			u := fs.CurrentUser()
			err = f.Chown(u.Uid(), u.Gid())
			if err != nil {
				t.Errorf("Chown : want error to be nil, got %v", err)
			}
		}

		fst, err := f.Stat()
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		wantName := fs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", existingFile, avfs.ErrWinAccessDenied, err)
		default:
			CheckPathError(t, "Truncate", "truncate", existingFile, os.ErrInvalid, err)
		}
	})

	t.Run("OpenFileDir", func(t *testing.T) {
		existingDir := fs.Join(rootDir, "existingDir")

		err := fs.Mkdir(existingDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir : want error to be nil, got %v", err)
		}

		f, err := fs.OpenFile(existingDir, os.O_WRONLY, avfs.DefaultFilePerm)
		CheckPathError(t, "OpenFile", "open", existingDir, avfs.ErrIsADirectory, err)

		if !reflect.ValueOf(f).IsNil() {
			t.Errorf("OpenFile : want file to be nil, got %v", f)
		}
	})

	t.Run("OpenFileExcl", func(t *testing.T) {
		fileExcl := fs.Join(rootDir, "fileExcl")

		f, err := fs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		f.Close()

		_, err = fs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "OpenFile", "open", fileExcl, avfs.ErrWinFileExists, err)
		default:
			CheckPathError(t, "OpenFile", "open", fileExcl, avfs.ErrFileExists, err)
		}
	})

	t.Run("OpenFileNonExistingPath", func(t *testing.T) {
		nonExistingPath := fs.Join(rootDir, "non/existing/path")
		_, err := fs.OpenFile(nonExistingPath, os.O_CREATE, avfs.DefaultFilePerm)

		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "OpenFile", "open", nonExistingPath, avfs.ErrWinPathNotFound, err)
		default:
			CheckPathError(t, "OpenFile", "open", nonExistingPath, avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// SuiteWriteFile tests WriteFile function.
func (cf *ConfigFs) SuiteWriteFile() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")

	t.Run("WriteFile", func(t *testing.T) {
		path := fs.Join(rootDir, "WriteFile.txt")

		err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("WriteFile : want error to be nil, got %v", err)
		}

		rb, err := fs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// SuiteWriteFile tests WriteString function.
func (cf *ConfigFs) SuiteWriteString() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestWriteString.txt")

	t.Run("WriteString", func(t *testing.T) {
		f, err := fs.Create(path)
		if err != nil {
			t.Errorf("Create %s : want error to be nil, got %v", path, err)
		}

		n, err := f.WriteString(string(data))
		if err != nil {
			t.Errorf("WriteString : want error to be nil, got %v", err)
		}

		if len(data) != n {
			t.Errorf("WriteString : want written bytes to be %d, got %d", len(data), n)
		}

		f.Close()

		rb, err := fs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// SuiteReadFile tests ReadFile function.
func (cf *ConfigFs) SuiteReadFile() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsRead()

	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestReadFile.txt")

	t.Run("ReadFile", func(t *testing.T) {
		rb, err := fs.ReadFile(path)
		if err == nil {
			t.Errorf("ReadFile : want error to be %v, got nil", avfs.ErrNoSuchFileOrDir)
		}

		if len(rb) != 0 {
			t.Errorf("ReadFile : want read bytes to be 0, got %d", len(rb))
		}

		fs = cf.GetFsWrite()

		err = fs.WriteFile(path, data, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		fs = cf.GetFsRead()

		rb, err = fs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// SuiteFileWrite tests Write, WriteAt functions.
func (cf *ConfigFs) SuiteFileWrite() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")

	t.Run("FileWriteSeq", func(t *testing.T) {
		path := fs.Join(rootDir, "TestFileWriteSeq.txt")

		f, err := fs.Create(path)
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

		rb, err := fs.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("FileWriteAt", func(t *testing.T) {
		path := fs.Join(rootDir, "TestFileWriteAt.txt")

		f, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("OpenFile : want error to be nil, got %v", err)
		}

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

		rb, err := fs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// SuiteFileWriteEdgeCases tests Write and WriteAt functions edge cases.
func (cf *ConfigFs) SuiteFileWriteEdgeCases() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	path := fs.Join(rootDir, "TestFileWriteEdgeCases.txt")
	data := []byte("AAABBBCCCDDD")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	t.Run("FileWriteEdgeCases", func(t *testing.T) {
		f, err := fs.OpenFile(path, os.O_RDWR, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.WriteAt(data, -1)
		CheckPathError(t, "WriteAt", "writeat", path, avfs.ErrNegativeOffset, err)

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}

		off := int64(len(data) * 3)

		n, err = f.WriteAt(data, off)
		if err != nil {
			t.Errorf("WriteAt : want error to be nil, got %v", err)
		}

		if n != len(data) {
			t.Errorf("WriteAt : want bytes written to be %d, got %d", len(data), n)
		}

		want := make([]byte, int(off)+len(data))
		_ = copy(want, data)
		_ = copy(want[off:], data)

		got, err := fs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(want, got) {
			t.Errorf("want : %s\ngot  : %s", want, got)
		}
	})
}

// SuiteFileRead tests Read, ReadAt functions.
func (cf *ConfigFs) SuiteFileRead() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestFileReadSeq.txt")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	fs = cf.GetFsRead()

	t.Run("FileRead", func(t *testing.T) {
		const bufSize = 5

		f, err := fs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
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

		f, err := fs.OpenFile(path, os.O_RDONLY, avfs.DefaultFilePerm)
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
}

// SuiteFileReadEdgeCases tests Read and ReadAt functions edge cases.
func (cf *ConfigFs) SuiteFileReadEdgeCases() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	path := fs.Join(rootDir, "TestFileReadEdgeCases.txt")
	data := []byte("AAABBBCCCDDD")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	fs = cf.GetFsRead()

	t.Run("FileReadEdgeCases", func(t *testing.T) {
		f, err := fs.Open(path)
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

	t.Run("FileReadOnly", func(t *testing.T) {
		f, err := fs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}
		defer f.Close()

		b := make([]byte, len(data)*2)
		n, err := f.Write(b)

		switch fs.OSType() {
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

		switch fs.OSType() {
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
}

// SuiteFileSeek tests Seek function.
func (cf *ConfigFs) SuiteFileSeek() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestFileSeek.txt")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	fs = cf.GetFsRead()

	f, err := fs.Open(path)
	if err != nil {
		t.Fatalf("Open : want error to be nil, got %v", err)
	}

	defer f.Close()

	var pos int64

	t.Run("FileSeek", func(t *testing.T) {
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

	t.Run("FileSeekInvalid", func(t *testing.T) {
		lenData := int64(len(data))

		// Invalid SeekStart

		pos, err = f.Seek(-1, io.SeekStart)
		CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)

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

		// Invalid SeekEnd

		pos, err = f.Seek(1, io.SeekEnd)
		if err != nil {
			t.Errorf("Seek : want error to be nil, got %v", err)
		}

		wantPos = lenData + 1
		if pos != wantPos {
			t.Errorf("Seek : want pos to be %d, got %d", wantPos, pos)
		}

		pos, err = f.Seek(-lenData*2, io.SeekEnd)
		CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}

		// Invalid SeekCur

		wantPos = lenData / 2
		pos, err = f.Seek(wantPos, io.SeekStart)
		if err != nil || pos != wantPos {
			t.Fatalf("Seek : want  pos to be 0 and error to be nil, got %d, %v", pos, err)
		}

		pos, err = f.Seek(-lenData, io.SeekCurrent)
		CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)

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

		// Invalid Whence

		pos, err = f.Seek(0, 10)
		CheckPathError(t, "Seek", "seek", f.Name(), os.ErrInvalid, err)

		if pos != 0 {
			t.Errorf("Seek : want pos to be %d, got %d", 0, pos)
		}
	})
}

// SuiteFileCloseRead tests file Close function for read only files.
func (cf *ConfigFs) SuiteFileCloseRead() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestFileCloseRead.txt")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	openInfo, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
	}

	t.Run("FileCloseReadOnly", func(t *testing.T) {
		fs = cf.GetFsRead()

		f, err := fs.Open(path)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		closeInfo, err := fs.Stat(path)
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

// SuiteFileCloseWrite tests file Close function for read/write files.
func (cf *ConfigFs) SuiteFileCloseWrite() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestFileCloseWrite.txt")

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	openInfo, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
	}

	t.Run("FileCloseWrite", func(t *testing.T) {
		f, err := fs.OpenFile(path, os.O_APPEND|os.O_WRONLY, avfs.DefaultFilePerm)
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

		closeInfo, err := fs.Stat(path)
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

// SuiteFileTruncate tests Truncate function.
func (cf *ConfigFs) SuiteFileTruncate() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	data := []byte("AAABBBCCCDDD")
	path := fs.Join(rootDir, "TestFileTruncate.txt")

	if err := fs.WriteFile(path, data, avfs.DefaultFilePerm); err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	t.Run("FileTruncate", func(t *testing.T) {
		f, err := fs.OpenFile(path, os.O_RDWR, avfs.DefaultFilePerm)
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

	err := fs.WriteFile(path, data, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	t.Run("FsTruncate", func(t *testing.T) {
		for i := len(data); i >= 0; i-- {
			err := fs.Truncate(path, int64(i))
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			d, err := fs.ReadFile(path)
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			if len(d) != i {
				t.Errorf("Truncate : want length to be %d, got %d", i, len(d))
			}
		}
	})
	t.Run("FsTruncateErrors", func(t *testing.T) {
		err := fs.Truncate(rootDir, 0)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "open", rootDir, avfs.ErrIsADirectory, err)
		default:
			CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrIsADirectory, err)
		}

		f, err := fs.Open(rootDir)
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		defer f.Close()

		err = f.Truncate(0)
		switch fs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Truncate", "truncate", rootDir, avfs.ErrWinInvalidHandle, err)
		default:
			CheckPathError(t, "Truncate", "truncate", rootDir, os.ErrInvalid, err)
		}
	})
}

// SuiteLink tests Link function.
func (cf *ConfigFs) SuiteLink() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	if !fs.HasFeature(avfs.FeatHardlink) {
		return
	}

	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)

	pathLinks := fs.Join(rootDir, "links")

	err := fs.Mkdir(pathLinks, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("mkdir %s : want error to be nil, got %v", pathLinks, err)
	}

	t.Run("LinkCreate", func(t *testing.T) {
		for _, file := range files {
			oldPath := fs.Join(rootDir, file.Path)
			newPath := fs.Join(pathLinks, fs.Base(file.Path))

			err := fs.Link(oldPath, newPath)
			if err != nil {
				t.Errorf("Link %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			newContent, err := fs.ReadFile(newPath)
			if err != nil {
				t.Errorf("Readfile %s : want error to be nil, got %v", newPath, err)
			}

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkExisting", func(t *testing.T) {
		for _, file := range files {
			oldPath := fs.Join(rootDir, file.Path)
			newPath := fs.Join(pathLinks, fs.Base(file.Path))

			err := fs.Link(oldPath, newPath)
			CheckLinkError(t, "Link", "link", oldPath, newPath, avfs.ErrFileExists, err)
		}
	})

	t.Run("LinkRemove", func(t *testing.T) {
		for _, file := range files {
			oldPath := fs.Join(rootDir, file.Path)
			newPath := fs.Join(pathLinks, fs.Base(file.Path))

			err := fs.Remove(oldPath)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", oldPath, err)
			}

			newContent, err := fs.ReadFile(newPath)
			if err != nil {
				t.Errorf("Readfile %s : want error to be nil, got %v", newPath, err)
			}

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkErrorDir", func(t *testing.T) {
		for _, dir := range dirs {
			oldPath := fs.Join(rootDir, dir.Path)
			newPath := fs.Join(rootDir, "WhateverDir")

			err := fs.Link(oldPath, newPath)
			CheckLinkError(t, "Link", "link", oldPath, newPath, avfs.ErrOpNotPermitted, err)
		}
	})

	t.Run("LinkErrorFile", func(t *testing.T) {
		for _, file := range files {
			InvalidPath := fs.Join(rootDir, file.Path, "OldInvalidPath")
			NewInvalidPath := fs.Join(pathLinks, "WhateverFile")

			err := fs.Link(InvalidPath, NewInvalidPath)
			CheckLinkError(t, "Link", "link", InvalidPath, NewInvalidPath, avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// SuiteSameFile tests SameFile function.
func (cf *ConfigFs) SuiteSameFile() {
	t, rootDir1, removeDir1 := cf.CreateRootDir(UsrTest)
	defer removeDir1()

	fs := cf.GetFsWrite()
	CreateDirs(t, fs, rootDir1)
	files := CreateFiles(t, fs, rootDir1)

	_, rootDir2, removeDir2 := cf.CreateRootDir(UsrTest)
	defer removeDir2()
	CreateDirs(t, fs, rootDir2)

	t.Run("SameFileLink", func(t *testing.T) {
		if !fs.HasFeature(avfs.FeatHardlink) {
			return
		}

		for _, file := range files {
			path1 := fs.Join(rootDir1, file.Path)
			path2 := fs.Join(rootDir2, file.Path)

			info1, err := fs.Stat(path1)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			err = fs.Link(path1, path2)
			if err != nil {
				t.Fatalf("Link %s : want error to be nil, got %v", path1, err)
			}

			info2, err := fs.Stat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if !fs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			err = fs.Remove(path2)
			if err != nil {
				t.Fatalf("Remove %s : want error to be nil, got %v", path2, err)
			}
		}
	})

	t.Run("SameFileSymlink", func(t *testing.T) {
		if !fs.HasFeature(avfs.FeatSymlink) {
			return
		}

		for _, file := range files {
			path1 := fs.Join(rootDir1, file.Path)
			path2 := fs.Join(rootDir2, file.Path)

			info1, err := fs.Stat(path1)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			err = fs.Symlink(path1, path2)
			if err != nil {
				t.Fatalf("Symlink %s : want error to be nil, got %v", path1, err)
			}

			info2, err := fs.Stat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if !fs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			info3, err := fs.Lstat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if fs.SameFile(info1, info3) {
				t.Fatalf("SameFile %s, %s : not the same file\n%v\n%v", path1, path2, info1, info3)
			}

			err = fs.Remove(path2)
			if err != nil {
				t.Fatalf("Remove %s : want error to be nil, got %v", path2, err)
			}
		}
	})
}

// SuiteFileWriteTime checks that modification time is updated on write operations.
func (cf *ConfigFs) SuiteFileWriteTime() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	data := []byte("AAABBBCCCDDD")
	existingFile := fs.Join(rootDir, "ExistingFile.txt")

	var start, end int64

	f, err := fs.Create(existingFile)
	if err != nil {
		t.Fatalf("Create : want error to be nil, got %v", err)
	}

	// CompareTime tests if the modification time of the file has changed.
	CompareTime := func(mustChange bool) {
		time.Sleep(10 * time.Millisecond)

		info, err := f.Stat() //nolint:govet
		if err != nil {
			if errors.Unwrap(err).Error() != avfs.ErrFileClosing.Error() {
				t.Fatalf("Stat : want error to be nil, got %v", err)
			}

			info, err = fs.Stat(existingFile)
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
