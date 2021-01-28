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
	"testing"
	"time"

	"github.com/avfs/avfs"
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
