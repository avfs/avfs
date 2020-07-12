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
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// SuiteChdir tests Chdir function.
func (cf *ConfigFs) SuiteChdir() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)

	fs = cf.GetFsRead()

	t.Run("ChdirFs", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			err := fs.Chdir(path)
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", path, err)
			}

			curDir, err := fs.Getwd()
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
			err := fs.Chdir(rootDir)
			if err != nil {
				t.Fatalf("Chdir %s : want error to be nil, got %v", rootDir, err)
			}

			relPath := dir.Path[1:]

			err = fs.Chdir(relPath)
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", relPath, err)
			}

			curDir, err := fs.Getwd()
			if err != nil {
				t.Errorf("Getwd : want error to be nil, got %v", err)
			}

			path := fs.Join(rootDir, relPath)
			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})

	t.Run("ChdirNonExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path, "NonExistingDir")

			oldPath, err := fs.Getwd()
			if err != nil {
				t.Errorf("Chdir : want error to be nil, got %v", err)
			}

			err = fs.Chdir(path)
			CheckPathError(t, "Chdir", "chdir", path, avfs.ErrNoSuchFileOrDir, err)

			newPath, err := fs.Getwd()
			if err != nil {
				t.Errorf("Getwd : want error to be nil, got %v", err)
			}

			if newPath != oldPath {
				t.Errorf("Getwd : want current dir to be %s, got %s", oldPath, newPath)
			}
		}
	})

	t.Run("ChdirFile", func(t *testing.T) {
		if fs.OSType() == avfs.OsWindows {
			t.Logf("File.Chdir() is not supported by windows, skipping")
			return
		}

		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			f, err := fs.Open(path)
			if err != nil {
				t.Errorf("Open %s : want error to be nil, got %v", path, err)
			}

			err = f.Chdir()
			if err != nil {
				t.Errorf("Chdir %s : want error to be nil, got %v", path, err)
			}

			f.Close()

			curDir, err := fs.Getwd()
			if err != nil {
				t.Errorf("Getwd %s : want error to be nil, got %v", path, err)
			}

			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})
}

// SuiteGetTempDir tests GetTempDir function.
func (cf *ConfigFs) SuiteGetTempDir() {
	t, fs := cf.t, cf.GetFsRead()

	wantTmp := os.Getenv("TMP")

	gotTmp := fs.GetTempDir()
	if gotTmp != wantTmp {
		t.Fatalf("GetTempDir : want temp dir to be %s, got %s", wantTmp, gotTmp)
	}
}

// SuiteMkdir tests Mkdir function.
func (cf *ConfigFs) SuiteMkdir() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsRead()
	dirs := GetDirs()

	t.Run("MkdirNew", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			err := fs.Mkdir(path, dir.Mode)
			if err != nil {
				t.Errorf("mkdir : want no error, got %v", err)
			}

			fi, err := fs.Stat(path)
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

			name := fs.Base(dir.Path)
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", path, name, fi.Name())
			}

			curPath := rootDir
			for start, end, i, isLast := 1, 0, 0, false; !isLast; start, i = end+1, i+1 {
				end, isLast = fsutil.SegmentPath(dir.Path, start)
				part := dir.Path[start:end]
				wantMode := dir.WantModes[i]

				curPath = fs.Join(curPath, part)
				info, err := fs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= fs.GetUMask()
				mode := info.Mode() & os.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			err := fs.Mkdir(path, dir.Mode)
			if !fs.IsExist(err) {
				t.Errorf("mkdir %s : want IsExist(err) to be true, got error %v", path, err)
			}
		}
	})

	t.Run("MkdirImpossible", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path, "can't", "create", "this")

			err := fs.Mkdir(path, avfs.DefaultDirPerm)
			CheckPathError(t, "Mkdir", "mkdir", path, avfs.ErrNoSuchFileOrDir, err)
		}

		err := fs.Mkdir("", avfs.DefaultFilePerm)
		CheckPathError(t, "Mkdir", "mkdir", "", avfs.ErrNoSuchFileOrDir, err)
	})
}

// SuiteMkdirAll tests MkdirAll function.
func (cf *ConfigFs) SuiteMkdirAll() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := GetDirsAll()

	t.Run("MkdirAll", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			err := fs.MkdirAll(path, dir.Mode)
			if err != nil {
				t.Errorf("MkdirAll : want error to be nil, got %v", err)
			}

			fi, err := fs.Stat(path)
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

			name := fs.Base(dir.Path)
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

				curPath = fs.Join(curPath, part)
				info, err := fs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= fs.GetUMask()
				mode := info.Mode() & os.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirAllExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			err := fs.MkdirAll(path, dir.Mode)
			if err != nil {
				t.Errorf("MkdirAll %s : want error to be nil, got error %v", path, err)
			}
		}
	})
}

// SuiteReadDir tests ReadDir function.
func (cf *ConfigFs) SuiteReadDir() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	const maxRead = 7

	fs := cf.GetFsWrite()

	rndTree, err := fsutil.NewRndTree(fs, RndParamsOneDir)
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

	fs = cf.GetFsRead()

	t.Run("ReadDirAll", func(t *testing.T) {
		rdInfos, err := fs.ReadDir(rootDir)
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
		f, err := fs.Open(rootDir)
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
			dirInfos, err := fs.ReadDir(dir)
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

// SuiteReadDirNames tests Readdirnames function.
func (cf *ConfigFs) SuiteReadDirNames() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	rndTree, err := fsutil.NewRndTree(fs, RndParamsOneDir)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(rootDir)
	if err != nil {
		t.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	wAll := len(rndTree.Dirs) + len(rndTree.Files) + len(rndTree.SymLinks)

	fs = cf.GetFsRead()

	t.Run("ReadDirNamesAll", func(t *testing.T) {
		f, err := fs.Open(rootDir)
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
		f, err := fs.Open(rootDir)
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
