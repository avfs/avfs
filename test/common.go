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
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// SuiteChtimes tests Chtimes function.
func (cf *ConfigFs) SuiteChtimes() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	_ = CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	tomorrow := time.Now().AddDate(0, 0, 1)

	for _, file := range files {
		path := fs.Join(rootDir, file.Path)

		err := fs.Chtimes(path, tomorrow, tomorrow)
		if err != nil {
			t.Errorf("Chtimes %s : want error to be nil, got %v", path, err)
		}

		infos, err := fs.Stat(path)
		if err != nil {
			t.Errorf("Chtimes %s : want error to be nil, got %v", path, err)
		}

		if infos.ModTime() != tomorrow {
			t.Errorf("Chtimes %s : want modtime to bo %s, got %s", path, tomorrow, infos.ModTime())
		}
	}
}

// SuiteEvalSymlink tests EvalSymlink function.
func (cf *ConfigFs) SuiteEvalSymlink() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	if !fs.Features(avfs.FeatSymlink) {
		return
	}

	_ = CreateDirs(t, fs, rootDir)
	_ = CreateFiles(t, fs, rootDir)
	_ = CreateSymlinks(t, fs, rootDir)

	fs = cf.GetFsRead()

	t.Run("EvalSymlink", func(t *testing.T) {
		symlinks := GetSymlinksEval(fs)
		for _, sl := range symlinks {
			wantOp := "lstat"
			wantPath := fs.Join(rootDir, sl.OldName)
			slPath := fs.Join(rootDir, sl.NewName)

			gotPath, err := fs.EvalSymlinks(slPath)
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

// SuiteLstat tests Lstat function.
func (cf *ConfigFs) SuiteLstat() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	CreateSymlinks(t, fs, rootDir)

	fs = cf.GetFsRead()

	t.Run("LstatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)
			info, err := fs.Lstat(path)
			if err != nil {
				t.Errorf("Lstat %s : want error to be nil, got %v", path, err)
				continue
			}

			if fs.Base(path) != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, fs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | os.ModeDir) &^ fs.GetUMask()
			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("LstatFile", func(t *testing.T) {
		for _, file := range files {
			path := fs.Join(rootDir, file.Path)
			info, err := fs.Lstat(path)
			if err != nil {
				t.Errorf("Lstat %s : want error to be nil, got %v", path, err)
				continue
			}

			if info.Name() != fs.Base(path) {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, fs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ fs.GetUMask()
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
		for _, sl := range GetSymlinksEval(fs) {
			newPath := fs.Join(rootDir, sl.NewName)
			oldPath := fs.Join(rootDir, sl.OldName)
			info, err := fs.Lstat(newPath)
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
				wantName = fs.Base(newPath)
				wantMode = os.ModeSymlink | os.ModePerm
			} else {
				wantName = fs.Base(oldPath)
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
}

// SuiteReadlink tests Readlink function.
func (cf *ConfigFs) SuiteReadlink() {
	if !cf.fsW.Features(avfs.FeatSymlink) {
		return
	}

	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)

	fs = cf.GetFsRead()

	t.Run("ReadlinkLink", func(t *testing.T) {
		for _, sl := range symlinks {
			oldPath := fs.Join(rootDir, sl.OldName)
			newPath := fs.Join(rootDir, sl.NewName)

			gotPath, err := fs.Readlink(newPath)
			if err != nil {
				t.Errorf("ReadLink %s : want error to be nil, got %v", newPath, err)
			}

			if oldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", newPath, oldPath, gotPath)
			}
		}
	})

	t.Run("ReadlinkDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			_, err := fs.Readlink(path)
			CheckPathError(t, "ReadLink", "readlink", path, os.ErrInvalid, err)
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			path := fs.Join(rootDir, file.Path)

			_, err := fs.Readlink(path)
			CheckPathError(t, "ReadLink", "readlink", path, os.ErrInvalid, err)
		}
	})
}

// SuiteRemove test Remove function.
func (cf *ConfigFs) SuiteRemove() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)

	if s, ok := fs.(fmt.Stringer); ok {
		fmt.Println(s.String())
	}

	t.Run("RemoveFile", func(t *testing.T) {
		for _, file := range files {
			path := fs.Join(rootDir, file.Path)

			_, err := fs.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err = fs.Remove(path)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", path, err)
			}

			_, err = fs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		if s, ok := fs.(fmt.Stringer); ok {
			fmt.Println(s.String())
		}

		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			dirInfos, err := fs.ReadDir(path)
			if err != nil {
				t.Fatalf("ReadDir %s : want error to be nil, got %v", path, err)
			}

			err = fs.Remove(path)

			isLeaf := len(dirInfos) == 0
			if isLeaf {
				if err != nil {
					t.Errorf("Remove %s : want error to be nil, got %v", path, err)
				}

				_, err = fs.Stat(path)
				CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
			} else {
				CheckPathError(t, "Remove", "remove", path, avfs.ErrDirNotEmpty, err)

				_, err = fs.Stat(path)
				if err != nil {
					t.Errorf("Remove %s : want error to be nil, got %v", path, err)
				}
			}
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			newPath := fs.Join(rootDir, sl.NewName)

			err := fs.Remove(newPath)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", newPath, err)
			}

			_, err = fs.Stat(newPath)
			CheckPathError(t, "Stat", "stat", newPath, avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// SuiteRemoveAll test RemoveAll function.
func (cf *ConfigFs) SuiteRemoveAll() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)

	t.Run("RemoveAll", func(t *testing.T) {
		err := fs.RemoveAll(rootDir)
		if err != nil {
			t.Fatalf("RemoveAll %s : want error to be nil, got %v", rootDir, err)
		}

		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)
			_, err := fs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, file := range files {
			path := fs.Join(rootDir, file.Path)
			_, err := fs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, sl := range symlinks {
			path := fs.Join(rootDir, sl.NewName)
			_, err := fs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("RemoveAllErrors", func(t *testing.T) {
		err := fs.MkdirAll(rootDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir %s : want error to be nil, got %v", rootDir, err)
		}

		existingFile := fs.Join(rootDir, "existingFile.txt")
		err = fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", existingFile, err)
		}

		err = fs.RemoveAll(existingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", existingFile, err)
		}
	})
}

// SuiteRemoveAllEdgeCases test RemoveAll function.
func (cf *ConfigFs) SuiteRemoveAllEdgeCases() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		err := fs.RemoveAll("")
		if err != nil {
			t.Errorf("RemoveAll '' : want error to be nil, got %v", err)
		}

		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)

			_, err = fs.Stat(path)
			if err != nil {
				t.Fatalf("RemoveAll %s : want error to be nil, got %v", path, err)
			}
		}
	})
}

// SuiteRename tests Rename function.
func (cf *ConfigFs) SuiteRename() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	t.Run("RenameDir", func(t *testing.T) {
		dirs := CreateDirs(t, fs, rootDir)

		for i := len(dirs) - 1; i >= 0; i-- {
			oldPath := fs.Join(rootDir, dirs[i].Path)
			newPath := oldPath + "New"

			err := fs.Rename(oldPath, newPath)
			if err != nil {
				t.Errorf("Rename %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			_, err = fs.Stat(oldPath)
			CheckPathError(t, "Stat", "stat", oldPath, avfs.ErrNoSuchFileOrDir, err)

			_, err = fs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})

	t.Run("RenameFile", func(t *testing.T) {
		CreateDirs(t, fs, rootDir)
		files := CreateFiles(t, fs, rootDir)

		for _, file := range files {
			oldPath := fs.Join(rootDir, file.Path)
			newPath := fs.Join(rootDir, fs.Base(oldPath))

			err := fs.Rename(oldPath, newPath)
			if err != nil {
				t.Errorf("Rename %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			_, err = fs.Stat(oldPath)

			switch {
			case oldPath == newPath:
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", oldPath, err)
				}
			default:
				CheckPathError(t, "Stat", "stat", oldPath, avfs.ErrNoSuchFileOrDir, err)
			}

			_, err = fs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})
}

// SuiteStat tests Stat function.
func (cf *ConfigFs) SuiteStat() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	_ = CreateSymlinks(t, fs, rootDir)

	fs = cf.GetFsRead()

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := fs.Join(rootDir, dir.Path)
			info, err := fs.Stat(path)

			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				continue
			}

			if fs.Base(path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", path, fs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | os.ModeDir) &^ fs.GetUMask()
			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatFile", func(t *testing.T) {
		for _, file := range files {
			path := fs.Join(rootDir, file.Path)

			info, err := fs.Stat(path)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				continue
			}

			if info.Name() != fs.Base(path) {
				t.Errorf("Stat %s : want name to be %s, got %s", path, fs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ fs.GetUMask()
			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", path, wantSize, info.Size())
			}
		}
	})

	t.Run("StatSymlink", func(t *testing.T) {
		for _, sl := range GetSymlinksEval(fs) {
			newPath := fs.Join(rootDir, sl.NewName)
			oldPath := fs.Join(rootDir, sl.OldName)

			info, err := fs.Stat(newPath)
			if err != nil {
				if sl.WantErr == nil {
					t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
				}
				CheckPathError(t, "Lstat", "stat", newPath, sl.WantErr, err)
				continue
			}

			var (
				wantName string
				wantMode os.FileMode
			)

			if sl.IsSymlink {
				wantName = fs.Base(newPath)
			} else {
				wantName = fs.Base(oldPath)
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
}

// SuiteSymlink tests Symlink function.
func (cf *ConfigFs) SuiteSymlink() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	if !fs.Features(avfs.FeatSymlink) {
		return
	}

	_ = CreateDirs(t, fs, rootDir)
	_ = CreateFiles(t, fs, rootDir)

	t.Run("Symlink", func(t *testing.T) {
		symlinks := GetSymlinks(fs)
		for _, sl := range symlinks {
			oldPath := fs.Join(rootDir, sl.OldName)
			newPath := fs.Join(rootDir, sl.NewName)

			err := fs.Symlink(oldPath, newPath)
			if err != nil {
				t.Errorf("Symlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			gotPath, err := fs.Readlink(newPath)
			if err != nil {
				t.Errorf("ReadLink %s : want error to be nil, got %v", newPath, err)
			}

			if oldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", newPath, oldPath, gotPath)
			}
		}
	})
}

// SuiteWalk tests Walk function.
func (cf *ConfigFs) SuiteWalk() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)

	fs = cf.GetFsRead()
	lnames := len(dirs) + len(files) + len(symlinks)
	wantNames := make([]string, 0, lnames)

	wantNames = append(wantNames, rootDir)
	for _, dir := range dirs {
		wantNames = append(wantNames, fs.Join(rootDir, dir.Path))
	}

	for _, file := range files {
		wantNames = append(wantNames, fs.Join(rootDir, file.Path))
	}

	if fs.Features(avfs.FeatSymlink) {
		for _, sl := range symlinks {
			wantNames = append(wantNames, fs.Join(rootDir, sl.NewName))
		}
	}

	sort.Strings(wantNames)

	t.Run("Walk", func(t *testing.T) {
		gotNames := make(map[string]int)
		err := fs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			gotNames[path]++

			return nil
		})

		if err != nil {
			t.Errorf("Walk %s : want error to be nil, got %v", rootDir, err)
		}

		if len(wantNames) != len(gotNames) {
			t.Errorf("Walk %s : want %d files or dirs, got %d", rootDir, len(wantNames), len(gotNames))
		}

		for _, wantName := range wantNames {
			n, ok := gotNames[wantName]
			if !ok || n != 1 {
				t.Errorf("Walk %s : path %s not found", rootDir, wantName)
			}
		}
	})
}

// SuiteReadOnly tests all write functions of a read only file system..
func (cf *ConfigFs) SuiteReadOnly() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	existingFile := fs.Join(rootDir, "existingFile")

	err := fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	newFile := fs.Join(existingFile, "newFile")

	fs = cf.GetFsRead()
	if !fs.Features(avfs.FeatReadOnly) {
		t.Errorf("Features : want read only file system")
	}

	t.Run("ReadOnlyFs", func(t *testing.T) {
		err := fs.Chmod(existingFile, avfs.DefaultFilePerm)
		CheckPathError(t, "Chmod", "chmod", existingFile, avfs.ErrPermDenied, err)

		err = fs.Chown(existingFile, 0, 0)
		CheckPathError(t, "Chown", "chown", existingFile, avfs.ErrPermDenied, err)

		err = fs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)

		err = fs.Chtimes(existingFile, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", existingFile, avfs.ErrPermDenied, err)

		_, err = fs.Create(newFile)
		CheckPathError(t, "Create", "open", newFile, avfs.ErrPermDenied, err)

		err = fs.Lchown(existingFile, 0, 0)
		CheckPathError(t, "Lchown", "lchown", existingFile, avfs.ErrPermDenied, err)

		err = fs.Link(existingFile, newFile)
		CheckLinkError(t, "Link", "link", existingFile, newFile, avfs.ErrPermDenied, err)

		err = fs.Mkdir(newFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", newFile, avfs.ErrPermDenied, err)

		err = fs.MkdirAll(newFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", newFile, avfs.ErrPermDenied, err)

		_, err = fs.OpenFile(newFile, os.O_RDWR, avfs.DefaultFilePerm)
		CheckPathError(t, "OpenFile", "open", newFile, avfs.ErrPermDenied, err)

		err = fs.Remove(existingFile)
		CheckPathError(t, "Remove", "remove", existingFile, avfs.ErrPermDenied, err)

		err = fs.RemoveAll(existingFile)
		CheckPathError(t, "RemoveAll", "removeall", existingFile, avfs.ErrPermDenied, err)

		err = fs.Rename(existingFile, newFile)
		CheckLinkError(t, "Rename", "rename", existingFile, newFile, avfs.ErrPermDenied, err)

		err = fs.Symlink(existingFile, newFile)
		CheckLinkError(t, "Symlink", "symlink", existingFile, newFile, avfs.ErrPermDenied, err)

		_, err = fs.TempDir(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempDir : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = fs.TempFile(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempFile : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = fs.Truncate(existingFile, 0)
		CheckPathError(t, "Truncate", "truncate", existingFile, avfs.ErrPermDenied, err)

		err = fs.WriteFile(newFile, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, "WriteFile", "open", newFile, avfs.ErrPermDenied, err)
	})

	t.Run("ReadOnlyFile", func(t *testing.T) {
		f, err := fs.Open(existingFile)
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
		}

		err = f.Chmod(0o777)
		CheckPathError(t, "Chmod", "chmod", f.Name(), avfs.ErrPermDenied, err)

		err = f.Chown(0, 0)
		CheckPathError(t, "Chown", "chown", f.Name(), avfs.ErrPermDenied, err)

		err = f.Truncate(0)
		CheckPathError(t, "Truncate", "truncate", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.Write([]byte{})
		CheckPathError(t, "Write", "write", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, "WriteAt", "write", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteString("")
		CheckPathError(t, "WriteString", "write", f.Name(), avfs.ErrPermDenied, err)
	})
}

// SuiteUmask tests UMask and GetUMask functions.
func (cf *ConfigFs) SuiteUmask() {
	const umaskTest = 0o077

	t, fs := cf.t, cf.GetFsWrite()

	umaskStart := fs.GetUMask()
	fs.UMask(umaskTest)

	u := fs.GetUMask()
	if u != umaskTest {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskTest, u)
	}

	fs.UMask(umaskStart)

	u = fs.GetUMask()
	if u != umaskStart {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskStart, u)
	}
}
