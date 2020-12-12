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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// Chdir tests Chdir function.
func (sfs *SuiteFs) Chdir() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("ChdirFs", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.Chdir(path)
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

	t.Run("ChdirRelative", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(rootDir)
			if err != nil {
				t.Fatalf("Chdir %s : want error to be nil, got %v", rootDir, err)
			}

			relPath := dir.Path[1:]

			err = vfs.Chdir(relPath)
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", relPath, err)
			}

			curDir, err := vfs.Getwd()
			if err != nil {
				t.Errorf("Getwd : want error to be nil, got %v", err)
			}

			path := vfs.Join(rootDir, relPath)
			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})

	t.Run("ChdirNonExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path, "NonExistingDir")

			oldPath, err := vfs.Getwd()
			if err != nil {
				t.Errorf("Chdir : want error to be nil, got %v", err)
			}

			err = vfs.Chdir(path)
			CheckPathError(t, "Chdir", "chdir", path, avfs.ErrNoSuchFileOrDir, err)

			newPath, err := vfs.Getwd()
			if err != nil {
				t.Errorf("Getwd : want error to be nil, got %v", err)
			}

			if newPath != oldPath {
				t.Errorf("Getwd : want current dir to be %s, got %s", oldPath, newPath)
			}
		}
	})

	t.Run("ChdirFile", func(t *testing.T) {
		if vfs.OSType() == avfs.OsWindows {
			t.Logf("File.Chdir() is not supported by windows, skipping")

			return
		}

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
}

// GetTempDir tests GetTempDir function.
func (sfs *SuiteFs) GetTempDir() {
	t, vfs := sfs.t, sfs.GetFsRead()

	var wantTmp string

	switch vfs.OSType() {
	case avfs.OsDarwin:
		wantTmp, _ = filepath.EvalSymlinks(os.TempDir())
	case avfs.OsWindows:
		wantTmp = os.Getenv("TMP")
	default:
		wantTmp = avfs.TmpDir
	}

	gotTmp := vfs.GetTempDir()
	if gotTmp != wantTmp {
		t.Fatalf("GetTempDir : want temp dir to be %s, got %s", wantTmp, gotTmp)
	}
}

// Mkdir tests Mkdir function.
func (sfs *SuiteFs) Mkdir() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()
	dirs := GetDirs()

	t.Run("MkdirNew", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.Mkdir(path, dir.Mode)
			if err != nil {
				t.Errorf("mkdir : want no error, got %v", err)
			}

			fi, err := vfs.Stat(path)
			if err != nil {
				t.Errorf("stat '%s' : want no error, got %v", path, err)

				continue
			}

			if !fi.IsDir() {
				t.Errorf("stat '%s' : want path to be a directory, got mode %s", path, fi.Mode())
			}

			if fi.Size() < 0 {
				t.Errorf("stat '%s': want directory size to be >= 0, got %d", path, fi.Size())
			}

			now := time.Now()
			if now.Sub(fi.ModTime()) > 2*time.Second {
				t.Errorf("stat '%s' : want time to be %s, got %s", path, time.Now(), fi.ModTime())
			}

			name := vfs.Base(dir.Path)
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", path, name, fi.Name())
			}

			curPath := rootDir
			for start, end, i, isLast := 1, 0, 0, false; !isLast; start, i = end+1, i+1 {
				end, isLast = fsutil.SegmentPath(dir.Path, start)
				part := dir.Path[start:end]
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, part)
				info, err := vfs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= vfs.GetUMask()
				if vfs.OSType() == avfs.OsWindows {
					wantMode = os.ModePerm
				}

				mode := info.Mode() & os.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.Mkdir(path, dir.Mode)
			if !vfs.IsExist(err) {
				t.Errorf("mkdir %s : want IsExist(err) to be true, got error %v", path, err)
			}
		}
	})

	t.Run("MkdirImpossible", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path, "can't", "create", "this")

			err := vfs.Mkdir(path, avfs.DefaultDirPerm)

			switch vfs.OSType() {
			case avfs.OsWindows:
				CheckPathError(t, "Mkdir", "mkdir", path, avfs.ErrWinPathNotFound, err)
			default:
				CheckPathError(t, "Mkdir", "mkdir", path, avfs.ErrNoSuchFileOrDir, err)
			}
		}

		err := vfs.Mkdir("", avfs.DefaultFilePerm)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Mkdir", "mkdir", "", avfs.ErrWinPathNotFound, err)
		default:
			CheckPathError(t, "Mkdir", "mkdir", "", avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// MkdirAll tests MkdirAll function.
func (sfs *SuiteFs) MkdirAll() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := GetDirsAll()

	t.Run("MkdirAll", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.MkdirAll(path, dir.Mode)
			if err != nil {
				t.Errorf("MkdirAll : want error to be nil, got %v", err)
			}

			fi, err := vfs.Stat(path)
			if err != nil {
				t.Fatalf("stat '%s' : want error to be nil, got %v", path, err)
			}

			if !fi.IsDir() {
				t.Errorf("stat '%s' : want path to be a directory, got mode %s", path, fi.Mode())
			}

			if fi.Size() < 0 {
				t.Errorf("stat '%s': want directory size to be >= 0, got %d", path, fi.Size())
			}

			now := time.Now()
			if now.Sub(fi.ModTime()) > 2*time.Second {
				t.Errorf("stat '%s' : want time to be %s, got %s", path, time.Now(), fi.ModTime())
			}

			name := vfs.Base(dir.Path)
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", path, name, fi.Name())
			}

			want := strings.Count(dir.Path, string(avfs.PathSeparator))
			got := len(dir.WantModes)
			if want != got {
				t.Fatalf("stat %s : want %d directories modes, got %d", path, want, got)
			}

			curPath := rootDir
			for start, end, i, isLast := 1, 0, 0, false; !isLast; start, i = end+1, i+1 {
				end, isLast = fsutil.SegmentPath(dir.Path, start)
				part := dir.Path[start:end]
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, part)
				info, err := vfs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= vfs.GetUMask()
				if vfs.OSType() == avfs.OsWindows {
					wantMode = os.ModePerm
				}

				mode := info.Mode() & os.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirAllExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.MkdirAll(path, dir.Mode)
			if err != nil {
				t.Errorf("MkdirAll %s : want error to be nil, got error %v", path, err)
			}
		}
	})
}

// ReadDir tests ReadDir function.
func (sfs *SuiteFs) ReadDir() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	const maxRead = 7

	vfs := sfs.GetFsWrite()

	rndTree, err := fsutil.NewRndTree(vfs, fsutil.RndParamsOneDir)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(rootDir)
	if err != nil {
		t.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)

	vfs = sfs.GetFsRead()

	t.Run("ReadDirAll", func(t *testing.T) {
		rdInfos, err := vfs.ReadDir(rootDir)
		if err != nil {
			t.Fatalf("ReadDir : want error to be nil, got %v", err)
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
			t.Errorf("ReadDir : want number of dirs to be %d, got %d", wDirs, gDirs)
		}

		if wFiles != gFiles {
			t.Errorf("ReadDir : want number of files to be %d, got %d", wFiles, gFiles)
		}

		if wSymlinks != gSymlinks {
			t.Errorf("ReadDir : want number of symbolic links to be %d, got %d", wSymlinks, gSymlinks)
		}
	})

	t.Run("ReadDirN", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("ReadDir : want error to be nil, got %v", err)
		}

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

	t.Run("ReadDirEmptySubDirs", func(t *testing.T) {
		for _, dir := range rndTree.Dirs {
			dirInfos, err := vfs.ReadDir(dir)
			if err != nil {
				t.Errorf("ReadDir %s : want error to be nil, got %v", dir, err)
			}

			l := len(dirInfos)
			if l != 0 {
				t.Errorf("ReadDir %s : want count to be O, got %d", dir, l)
			}
		}
	})
}

// ReadDirNames tests Readdirnames function.
func (sfs *SuiteFs) ReadDirNames() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	rndTree, err := fsutil.NewRndTree(vfs, fsutil.RndParamsOneDir)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(rootDir)
	if err != nil {
		t.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)

	vfs = sfs.GetFsRead()

	t.Run("ReadDirNamesAll", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("ReadDirNames : want error to be nil, got %v", err)
		}

		names, err := f.Readdirnames(-1)
		if err != nil {
			t.Errorf("ReadDirNames : want error to be nil, got %v", err)
		}

		if wAll != len(names) {
			t.Errorf("ReadDirNames : want number of elements to be %d, got %d", wAll, len(names))
		}
	})

	t.Run("ReadDirNamesN", func(t *testing.T) {
		f, err := vfs.Open(rootDir)
		if err != nil {
			t.Fatalf("ReadDirNames : want error to be nil, got %v", err)
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
}
