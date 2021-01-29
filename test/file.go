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
	"io"
	"os"
	"testing"

	"github.com/avfs/avfs"
)

// Chdir tests File.Chdir function.
func (sfs *SuiteFS) FileChdir(t *testing.T) {
	if sfs.OSType() == avfs.OsWindows {
		t.Logf("File.Chdir() is not supported by windows, skipping")

		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f, _ := vfs.Open(rootDir)

		err := f.Chdir()
		CheckPathError(t, "Chdir", "chdir", f.Name(), avfs.ErrPermDenied, err)

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	existingFile := CreateEmptyFile(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("FileChdir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			f, err := vfs.Open(path)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", path, err)
			}

			err = f.Chdir()
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", path, err)
			}

			f.Close()

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

// ReadDir tests FileReadDir function.
func (sfs *SuiteFS) FileReadDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	rndTree := CreateRndDir(t, vfs, rootDir)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)
	existingFile := rndTree.Files[0]

	vfs = sfs.GetFsRead()

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

// FileReaddirnames tests Readdirnames function.
func (sfs *SuiteFS) FileReaddirnames(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	rndTree := CreateRndDir(t, vfs, rootDir)
	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)
	existingFile := rndTree.Files[0]

	vfs = sfs.GetFsRead()

	t.Run("FileReaddirnamesAll", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("FileReaddirnames : want error to be nil, got %v", err)
		}

		names, err := f.Readdirnames(-1)
		if err != nil {
			t.Errorf("FileReaddirnames : want error to be nil, got %v", err)
		}

		if wAll != len(names) {
			t.Errorf("FileReaddirnames : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("FileReaddirnamesN", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("FileReaddirnames : want error to be nil, got %v", err)
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
