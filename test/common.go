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
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// Chtimes tests Chtimes function.
func (sfs *SuiteFs) Chtimes() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

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
}

// EvalSymlink tests EvalSymlink function.
func (sfs *SuiteFs) EvalSymlink() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	if !vfs.HasFeature(avfs.FeatSymlink) {
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
func (sfs *SuiteFs) Lstat() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
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
}

// Readlink tests Readlink function.
func (sfs *SuiteFs) Readlink() {
	if !sfs.vfsW.HasFeature(avfs.FeatSymlink) {
		return
	}

	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("ReadlinkLink", func(t *testing.T) {
		for _, sl := range symlinks {
			oldPath := vfs.Join(rootDir, sl.OldName)
			newPath := vfs.Join(rootDir, sl.NewName)

			gotPath, err := vfs.Readlink(newPath)
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
			path := vfs.Join(rootDir, dir.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, "ReadLink", "readlink", path, os.ErrInvalid, err)
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, "ReadLink", "readlink", path, os.ErrInvalid, err)
		}
	})
}

// Remove tests Remove function.
func (sfs *SuiteFs) Remove() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)

	if s, ok := vfs.(fmt.Stringer); ok {
		fmt.Println(s.String())
	}

	t.Run("RemoveFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)

			_, err := vfs.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err = vfs.Remove(path)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", path, err)
			}

			_, err = vfs.Stat(path)

			switch vfs.OSType() {
			case avfs.OsWindows:
				CheckPathError(t, "Stat", "CreateFile", path, avfs.ErrNoSuchFileOrDir, err)
			default:
				CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
			}
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		if s, ok := vfs.(fmt.Stringer); ok {
			fmt.Println(s.String())
		}

		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			dirInfos, err := vfs.ReadDir(path)
			if err != nil {
				t.Fatalf("ReadDir %s : want error to be nil, got %v", path, err)
			}

			err = vfs.Remove(path)

			isLeaf := len(dirInfos) == 0
			if isLeaf {
				if err != nil {
					t.Errorf("Remove %s : want error to be nil, got %v", path, err)
				}

				_, err = vfs.Stat(path)
				CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
			} else {
				CheckPathError(t, "Remove", "remove", path, avfs.ErrDirNotEmpty, err)

				_, err = vfs.Stat(path)
				if err != nil {
					t.Errorf("Remove %s : want error to be nil, got %v", path, err)
				}
			}
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			newPath := vfs.Join(rootDir, sl.NewName)

			err := vfs.Remove(newPath)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", newPath, err)
			}

			_, err = vfs.Stat(newPath)
			CheckPathError(t, "Stat", "stat", newPath, avfs.ErrNoSuchFileOrDir, err)
		}
	})
}

// RemoveAll tests RemoveAll function.
func (sfs *SuiteFs) RemoveAll() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)

	t.Run("RemoveAll", func(t *testing.T) {
		err := vfs.RemoveAll(rootDir)
		if err != nil {
			t.Fatalf("RemoveAll %s : want error to be nil, got %v", rootDir, err)
		}

		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)
			_, err := vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)
			_, err := vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, sl := range symlinks {
			path := vfs.Join(rootDir, sl.NewName)
			_, err := vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("RemoveAllErrors", func(t *testing.T) {
		err := vfs.MkdirAll(rootDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir %s : want error to be nil, got %v", rootDir, err)
		}

		existingFile := vfs.Join(rootDir, "existingFile.txt")
		err = vfs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", existingFile, err)
		}

		err = vfs.RemoveAll(existingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", existingFile, err)
		}
	})
}

// RemoveAllEdgeCases tests edge cases of RemoveAll function.
func (sfs *SuiteFs) RemoveAllEdgeCases() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		err := vfs.RemoveAll("")
		if err != nil {
			t.Errorf("RemoveAll '' : want error to be nil, got %v", err)
		}

		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			_, err = vfs.Stat(path)
			if err != nil {
				t.Fatalf("RemoveAll %s : want error to be nil, got %v", path, err)
			}
		}
	})
}

// Rename tests Rename function.
func (sfs *SuiteFs) Rename() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	t.Run("RenameDir", func(t *testing.T) {
		dirs := CreateDirs(t, vfs, rootDir)

		for i := len(dirs) - 1; i >= 0; i-- {
			oldPath := vfs.Join(rootDir, dirs[i].Path)
			newPath := oldPath + "New"

			err := vfs.Rename(oldPath, newPath)
			if err != nil {
				t.Errorf("Rename %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			_, err = vfs.Stat(oldPath)

			switch vfs.OSType() {
			case avfs.OsWindows:
				CheckPathError(t, "Stat", "CreateFile", oldPath, avfs.ErrNoSuchFileOrDir, err)
			default:
				CheckPathError(t, "Stat", "stat", oldPath, avfs.ErrNoSuchFileOrDir, err)
			}

			_, err = vfs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})

	t.Run("RenameFile", func(t *testing.T) {
		CreateDirs(t, vfs, rootDir)
		files := CreateFiles(t, vfs, rootDir)

		for _, file := range files {
			oldPath := vfs.Join(rootDir, file.Path)
			newPath := vfs.Join(rootDir, vfs.Base(oldPath))

			err := vfs.Rename(oldPath, newPath)
			if err != nil {
				t.Errorf("Rename %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			_, err = vfs.Stat(oldPath)

			switch {
			case oldPath == newPath:
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", oldPath, err)
				}
			default:

				switch vfs.OSType() {
				case avfs.OsWindows:
					CheckPathError(t, "Stat", "CreateFile", oldPath, avfs.ErrNoSuchFileOrDir, err)
				default:
					CheckPathError(t, "Stat", "stat", oldPath, avfs.ErrNoSuchFileOrDir, err)
				}
			}

			_, err = vfs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})
}

// Stat tests Stat function.
func (sfs *SuiteFs) Stat() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	_ = CreateSymlinks(t, vfs, rootDir)

	vfs = sfs.GetFsRead()

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(rootDir, dir.Path)

			info, err := vfs.Stat(path)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | os.ModeDir) &^ vfs.GetUMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = os.ModeDir | os.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(rootDir, file.Path)

			info, err := vfs.Stat(path)
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

	t.Run("StatSymlink", func(t *testing.T) {
		for _, sl := range GetSymlinksEval(vfs) {
			newPath := vfs.Join(rootDir, sl.NewName)
			oldPath := vfs.Join(rootDir, sl.OldName)

			info, err := vfs.Stat(newPath)
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
}

// Symlink tests Symlink function.
func (sfs *SuiteFs) Symlink() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return
	}

	_ = CreateDirs(t, vfs, rootDir)
	_ = CreateFiles(t, vfs, rootDir)

	t.Run("Symlink", func(t *testing.T) {
		symlinks := GetSymlinks(vfs)
		for _, sl := range symlinks {
			oldPath := vfs.Join(rootDir, sl.OldName)
			newPath := vfs.Join(rootDir, sl.NewName)

			err := vfs.Symlink(oldPath, newPath)
			if err != nil {
				t.Errorf("Symlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			gotPath, err := vfs.Readlink(newPath)
			if err != nil {
				t.Errorf("ReadLink %s : want error to be nil, got %v", newPath, err)
			}

			if oldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", newPath, oldPath, gotPath)
			}
		}
	})
}

// WriteOnReadOnly tests all write functions of a read only file system.
func (sfs *SuiteFs) WriteOnReadOnly() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	existingFile := vfs.Join(rootDir, "existingFile")

	err := vfs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	newFile := vfs.Join(existingFile, "newFile")

	vfs = sfs.GetFsRead()
	if !vfs.HasFeature(avfs.FeatReadOnly) {
		t.Errorf("HasFeature : want read only file system")
	}

	t.Run("ReadOnlyFs", func(t *testing.T) {
		err := vfs.Chmod(existingFile, avfs.DefaultFilePerm)
		CheckPathError(t, "Chmod", "chmod", existingFile, avfs.ErrPermDenied, err)

		err = vfs.Chown(existingFile, 0, 0)
		CheckPathError(t, "Chown", "chown", existingFile, avfs.ErrPermDenied, err)

		err = vfs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)

		err = vfs.Chtimes(existingFile, time.Now(), time.Now())
		CheckPathError(t, "Chtimes", "chtimes", existingFile, avfs.ErrPermDenied, err)

		_, err = vfs.Create(newFile)
		CheckPathError(t, "Create", "open", newFile, avfs.ErrPermDenied, err)

		err = vfs.Lchown(existingFile, 0, 0)
		CheckPathError(t, "Lchown", "lchown", existingFile, avfs.ErrPermDenied, err)

		err = vfs.Link(existingFile, newFile)
		CheckLinkError(t, "Link", "link", existingFile, newFile, avfs.ErrPermDenied, err)

		err = vfs.Mkdir(newFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Mkdir", "mkdir", newFile, avfs.ErrPermDenied, err)

		err = vfs.MkdirAll(newFile, avfs.DefaultDirPerm)
		CheckPathError(t, "MkdirAll", "mkdir", newFile, avfs.ErrPermDenied, err)

		_, err = vfs.OpenFile(newFile, os.O_RDWR, avfs.DefaultFilePerm)
		CheckPathError(t, "OpenFile", "open", newFile, avfs.ErrPermDenied, err)

		err = vfs.Remove(existingFile)
		CheckPathError(t, "Remove", "remove", existingFile, avfs.ErrPermDenied, err)

		err = vfs.RemoveAll(existingFile)
		CheckPathError(t, "RemoveAll", "removeall", existingFile, avfs.ErrPermDenied, err)

		err = vfs.Rename(existingFile, newFile)
		CheckLinkError(t, "Rename", "rename", existingFile, newFile, avfs.ErrPermDenied, err)

		err = vfs.Symlink(existingFile, newFile)
		CheckLinkError(t, "Symlink", "symlink", existingFile, newFile, avfs.ErrPermDenied, err)

		_, err = vfs.TempDir(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempDir : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = vfs.TempFile(rootDir, "")
		if err.(*os.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("TempFile : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = vfs.Truncate(existingFile, 0)
		CheckPathError(t, "Truncate", "truncate", existingFile, avfs.ErrPermDenied, err)

		err = vfs.WriteFile(newFile, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, "WriteFile", "open", newFile, avfs.ErrPermDenied, err)
	})

	t.Run("ReadOnlyFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
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

// Umask tests UMask and GetUMask functions.
func (sfs *SuiteFs) Umask() {
	const umaskTest = 0o077

	t, vfs := sfs.t, sfs.GetFsWrite()

	umaskStart := vfs.GetUMask()
	vfs.UMask(umaskTest)

	u := vfs.GetUMask()
	if u != umaskTest {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskTest, u)
	}

	vfs.UMask(umaskStart)

	u = vfs.GetUMask()
	if u != umaskStart {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskStart, u)
	}
}
