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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// Chdir tests Chdir and Getwd functions.
func (sfs *SuiteFS) Chdir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Chdir(rootDir)
		CheckPathError(t, "Chdir", "chdir", rootDir, avfs.ErrPermDenied, err)

		_, err = vfs.Getwd()
		CheckPathError(t, "Getwd", "getwd", "", avfs.ErrPermDenied, err)

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	existingFile := CreateEmptyFile(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("ChdirAbsolute", func(t *testing.T) {
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

	t.Run("ChdirOnFile", func(t *testing.T) {
		err := vfs.Chdir(existingFile)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chdir", "chdir", existingFile, avfs.ErrWinDirNameInvalid, err)
		default:
			CheckPathError(t, "Chdir", "chdir", existingFile, avfs.ErrNotADirectory, err)
		}
	})
}

// Chtimes tests Chtimes function.
func (sfs *SuiteFS) Chtimes(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Chtimes(rootDir, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", rootDir, avfs.ErrPermDenied, err)

		return
	}

	t.Run("Chtimes", func(t *testing.T) {
		_ = CreateDirs(t, vfs, rootDir)
		files := CreateFiles(t, vfs, rootDir)
		tomorrow := time.Now().AddDate(0, 0, 1)

		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)

			err := vfs.Chtimes(path, tomorrow, tomorrow)
			if err != nil {
				t.Errorf("Chtimes %s : want error to be nil, got %v", path, err)
			}

			infos, err := vfs.Stat(path)
			if err != nil {
				t.Errorf("Chtimes %s : want error to be nil, got %v", path, err)
			}

			if infos.ModTime() != tomorrow {
				t.Errorf("Chtimes %s : want modtime to bo %s, got %s", path, tomorrow, infos.ModTime())
			}
		}
	})

	t.Run("ChtimesNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		err := vfs.Chtimes(nonExistingFile, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})
}

// GetTempDir tests GetTempDir function.
func (sfs *SuiteFS) GetTempDir(t *testing.T) {
	vfs := sfs.GetFsRead()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		tmp := vfs.GetTempDir()
		if tmp != avfs.TmpDir {
			t.Errorf("GetTempDir : want error to be %v, got %v", avfs.NotImplemented, tmp)
		}

		return
	}

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

// EvalSymlink tests EvalSymlink function.
func (sfs *SuiteFS) EvalSymlink(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	if !vfs.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.EvalSymlinks(rootDir)
		CheckPathError(t, "EvalSymlinks", "lstat", rootDir, avfs.ErrPermDenied, err)

		return
	}

	_ = CreateDirs(t, vfs, rootDir)
	_ = CreateFiles(t, vfs, rootDir)
	_ = CreateSymlinks(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("EvalSymlink", func(t *testing.T) {
		symlinks := GetSymlinksEval(vfs)
		for _, sl := range symlinks {
			wantOp := "lstat"
			wantPath := vfs.Join(rootDir, sl.OldName)
			slPath := vfs.Join(rootDir, sl.NewName)

			gotPath, err := vfs.EvalSymlinks(slPath)
			if sl.WantErr == nil && err == nil {
				if wantPath != gotPath {
					t.Errorf("EvalSymlinks %s : want Path to be %s, got %s", slPath, wantPath, gotPath)
				}

				continue
			}

			e, ok := err.(*os.PathError)
			if !ok && sl.WantErr != err {
				t.Errorf("EvalSymlinks %s : want error %v, got %v", slPath, sl.WantErr, err)
			}

			if wantOp != e.Op || wantPath != e.Path || sl.WantErr != e.Err {
				t.Errorf("EvalSymlinks %s : error"+
					"\nwant : Op: %s, Path: %s, Err: %v\ngot  : Op: %s, Path: %s, Err: %v",
					sl.NewName, wantOp, wantPath, sl.WantErr, e.Op, e.Path, e.Err)
			}
		}
	})
}

// Lstat tests Lstat function.
func (sfs *SuiteFS) Lstat(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Lstat(rootDir)
		CheckPathError(t, "Lstat", "lstat", rootDir, avfs.ErrPermDenied, err)

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	CreateSymlinks(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("LstatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			info, err := vfs.Lstat(path)
			if err != nil {
				t.Errorf("Lstat %s : want error to be nil, got %v", path, err)

				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | os.ModeDir) &^ vfs.GetUMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = os.ModeDir | os.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("LstatFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)

			info, err := vfs.Lstat(path)
			if err != nil {
				t.Errorf("Lstat %s : want error to be nil, got %v", path, err)

				continue
			}

			if info.Name() != vfs.Base(path) {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ vfs.GetUMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = 0o666
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", path, wantSize, info.Size())
			}
		}
	})

	t.Run("LstatSymlink", func(t *testing.T) {
		for _, sl := range GetSymlinksEval(vfs) {
			newPath := vfs.Join(rootDir, sl.NewName)
			oldPath := vfs.Join(rootDir, sl.OldName)

			info, err := vfs.Lstat(newPath)
			if err != nil {
				if sl.WantErr == nil {
					t.Errorf("Lstat %s : want error to be nil, got %v", newPath, err)
				}

				CheckPathError(t, "Lstat", "stat", newPath, sl.WantErr, err)

				continue
			}

			var (
				wantName string
				wantMode os.FileMode
			)

			if sl.IsSymlink {
				wantName = vfs.Base(newPath)
				wantMode = os.ModeSymlink | os.ModePerm
			} else {
				wantName = vfs.Base(oldPath)
				wantMode = sl.Mode
			}

			if wantName != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", newPath, wantName, info.Name())
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", newPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("LStatNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		_, err := vfs.Lstat(nonExistingFile)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Lstat", "CreateFile", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		default:
			CheckPathError(t, "Lstat", "lstat", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("LStatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(rootDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Lstat(subDirOnFile)
		CheckPathError(t, "Lstat", "lstat", subDirOnFile, avfs.ErrNotADirectory, err)
	})
}

// Mkdir tests Mkdir function.
func (sfs *SuiteFS) Mkdir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Mkdir(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", rootDir, avfs.ErrPermDenied, err)

		return
	}

	existingFile := CreateEmptyFile(t, vfs, rootDir)

	vfs = sfs.GetFsRead()
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
				end, isLast = vfsutils.SegmentPath(dir.Path, start)
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

	t.Run("MkdirOnNonExistingDir", func(t *testing.T) {
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
	})

	t.Run("MkdirEmptyName", func(t *testing.T) {
		err := vfs.Mkdir("", avfs.DefaultFilePerm)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Mkdir", "mkdir", "", avfs.ErrWinPathNotFound, err)
		default:
			CheckPathError(t, "Mkdir", "mkdir", "", avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("MkdirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(existingFile, "subDirOnFile")

		err := vfs.Mkdir(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", subDirOnFile, avfs.ErrNotADirectory, err)
	})
}

// MkdirAll tests MkdirAll function.
func (sfs *SuiteFS) MkdirAll(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.MkdirAll(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", rootDir, avfs.ErrPermDenied, err)

		return
	}

	existingFile := CreateEmptyFile(t, vfs, rootDir)

	vfs = sfs.GetFsWrite()
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
				end, isLast = vfsutils.SegmentPath(dir.Path, start)
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

	t.Run("MkdirAllExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			err := vfs.MkdirAll(path, dir.Mode)
			if err != nil {
				t.Errorf("MkdirAll %s : want error to be nil, got error %v", path, err)
			}
		}
	})

	t.Run("MkdirAllOnFile", func(t *testing.T) {
		err := vfs.MkdirAll(existingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", existingFile, avfs.ErrNotADirectory, err)
	})

	t.Run("MkdirAllSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(existingFile, "subDirOnFile")

		err := vfs.MkdirAll(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", existingFile, avfs.ErrNotADirectory, err)
	})
}

// ReadDir tests ReadDir function.
func (sfs *SuiteFS) ReadDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.ReadDir(rootDir)
		CheckPathError(t, "ReadDir", "open", rootDir, avfs.ErrPermDenied, err)

		return
	}

	rndTree := CreateRndDir(t, vfs, rootDir)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)

	existingFile := rndTree.Files[0]

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

	t.Run("ReadDirExistingFile", func(t *testing.T) {
		_, err := vfs.ReadDir(existingFile)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "ReadDir", "Readdir", existingFile, avfs.ErrNotADirectory, err)
		default:
			CheckSyscallError(t, "ReadDir", "readdirent", existingFile, avfs.ErrNotADirectory, err)
		}
	})
}

// TempDir tests TempDir function.
func (sfs *SuiteFS) TempDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.TempDir(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempDir : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	existingFile := CreateEmptyFile(t, vfs, rootDir)

	t.Run("TempDirOnFile", func(t *testing.T) {
		_, err := vfs.TempDir(existingFile, "")

		e, ok := err.(*os.PathError)
		if !ok {
			t.Fatalf("TempDir : want error type *os.PathError, got %v", reflect.TypeOf(err))
		}

		const op = "mkdir"
		wantErr := avfs.ErrNotADirectory
		if e.Op != op || vfs.Dir(e.Path) != existingFile || e.Err != wantErr {
			wantPathErr := &os.PathError{Op: op, Path: existingFile + "/<random number>", Err: wantErr}
			t.Errorf("TempDir : want error to be %v, got %v", wantPathErr, err)
		}
	})
}

// TempFile tests TempFile function.
func (sfs *SuiteFS) TempFile(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.TempFile(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempFile : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	existingFile := CreateEmptyFile(t, vfs, rootDir)

	t.Run("TempFileOnFile", func(t *testing.T) {
		_, err := vfs.TempFile(existingFile, "")

		e, ok := err.(*os.PathError)
		if !ok {
			t.Fatalf("TempFile : want error type *os.PathError, got %v", reflect.TypeOf(err))
		}

		const op = "open"
		wantErr := avfs.ErrNotADirectory
		if e.Op != op || vfs.Dir(e.Path) != existingFile || e.Err != wantErr {
			wantPathErr := &os.PathError{Op: op, Path: existingFile + "/<random number>", Err: wantErr}
			t.Errorf("TempDir : want error to be %v, got %v", wantPathErr, err)
		}
	})
}
