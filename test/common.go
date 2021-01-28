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

// Readlink tests Readlink function.
func (sfs *SuiteFS) Readlink(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !sfs.vfsW.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.Readlink(rootDir)
		CheckPathError(t, "Readlink", "readlink", rootDir, avfs.ErrPermDenied, err)

		return
	}

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

	t.Run("ReadLinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		_, err := vfs.Readlink(nonExistingFile)
		CheckPathError(t, "Readlink", "readlink", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})
}

// Remove tests Remove function.
func (sfs *SuiteFS) Remove(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Remove(rootDir)
		CheckPathError(t, "Remove", "remove", rootDir, avfs.ErrPermDenied, err)

		return
	}

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

	t.Run("RemoveNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		err := vfs.Remove(nonExistingFile)
		CheckPathError(t, "Remove", "remove", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})
}

// RemoveAll tests RemoveAll function.
func (sfs *SuiteFS) RemoveAll(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.RemoveAll(rootDir)
		CheckPathError(t, "RemoveAll", "removeall", rootDir, avfs.ErrPermDenied, err)

		return
	}

	baseDir := vfs.Join(rootDir, "RemoveAll")
	dirs := CreateDirs(t, vfs, baseDir)
	files := CreateFiles(t, vfs, baseDir)
	symlinks := CreateSymlinks(t, vfs, baseDir)

	t.Run("RemoveAll", func(t *testing.T) {
		err := vfs.RemoveAll(baseDir)
		if err != nil {
			t.Fatalf("RemoveAll %s : want error to be nil, got %v", baseDir, err)
		}

		for _, dir := range dirs {
			path := vfs.Join(baseDir, dir.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, file := range files {
			path := vfs.Join(baseDir, file.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		for _, sl := range symlinks {
			path := vfs.Join(baseDir, sl.NewName)

			_, err = vfs.Stat(path)
			CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
		}

		_, err = vfs.Stat(baseDir)
		CheckPathError(t, "Stat", "stat", baseDir, avfs.ErrNoSuchFileOrDir, err)
	})

	t.Run("RemoveAllOneFile", func(t *testing.T) {
		err := vfs.MkdirAll(baseDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir %s : want error to be nil, got %v", baseDir, err)
		}

		existingFile := CreateEmptyFile(t, vfs, rootDir)

		err = vfs.RemoveAll(existingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", existingFile, err)
		}
	})

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		CreateDirs(t, vfs, baseDir)

		err := vfs.Chdir(baseDir)
		if err != nil {
			t.Fatalf("Chdir %s : want error to be nil, got %v", baseDir, err)
		}

		err = vfs.RemoveAll("")
		if err != nil {
			t.Errorf("RemoveAll '' : want error to be nil, got %v", err)
		}

		// Verify that nothing was removed.
		for _, dir := range dirs {
			path := vfs.Join(baseDir, dir.Path)

			_, err = vfs.Stat(path)
			if err != nil {
				t.Fatalf("RemoveAll %s : want error to be nil, got %v", path, err)
			}
		}
	})

	t.Run("RemoveAllNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		err := vfs.RemoveAll(nonExistingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", nonExistingFile, err)
		}
	})
}

// Rename tests Rename function.
func (sfs *SuiteFS) Rename(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Rename(rootDir, rootDir)
		CheckLinkError(t, "Rename", "rename", rootDir, rootDir, avfs.ErrPermDenied, err)

		return
	}

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

	t.Run("RenameNonExistingFile", func(t *testing.T) {
		srcNonExistingFile := vfs.Join(rootDir, "srcNonExistingFile1")
		dstNonExistingFile := vfs.Join(rootDir, "dstNonExistingFile1")

		err := vfs.Rename(srcNonExistingFile, dstNonExistingFile)
		CheckLinkError(t, "Rename", "rename", srcNonExistingFile, dstNonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})

	t.Run("RenameDirToExistingDir", func(t *testing.T) {
		srcExistingDir := vfs.Join(rootDir, "srcExistingDir2")
		dstExistingDir := vfs.Join(rootDir, "dstExistingDir2")

		err := vfs.Mkdir(srcExistingDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir : want error to be nil, got %v", err)
		}

		err = vfs.Mkdir(dstExistingDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir : want error to be nil, got %v", err)
		}

		err = vfs.Rename(srcExistingDir, dstExistingDir)
		CheckLinkError(t, "Rename", "rename", srcExistingDir, dstExistingDir, avfs.ErrFileExists, err)
	})

	t.Run("RenameFileToExistingFile", func(t *testing.T) {
		srcExistingFile := vfs.Join(rootDir, "srcExistingFile3")
		dstExistingFile := vfs.Join(rootDir, "dstExistingFile3")
		data := []byte("data")

		err := vfs.WriteFile(srcExistingFile, data, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		err = vfs.WriteFile(dstExistingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		err = vfs.Rename(srcExistingFile, dstExistingFile)
		if err != nil {
			t.Errorf("Rename : want error to be nil, got %v", err)
		}

		_, err = vfs.Stat(srcExistingFile)
		CheckPathError(t, "Stat", "stat", srcExistingFile, avfs.ErrNoSuchFileOrDir, err)

		info, err := vfs.Stat(dstExistingFile)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		if int(info.Size()) != len(data) {
			t.Errorf("Stat : want size to be %d, got %d", len(data), info.Size())
		}
	})

	t.Run("RenameFileToExistingDir", func(t *testing.T) {
		srcExistingFile := vfs.Join(rootDir, "srcExistingFile4")
		dstExistingDir := vfs.Join(rootDir, "dstExistingDir4")

		err := vfs.WriteFile(srcExistingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		err = vfs.Mkdir(dstExistingDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("Mkdir : want error to be nil, got %v", err)
		}

		err = vfs.Rename(srcExistingFile, dstExistingDir)
		CheckLinkError(t, "Rename", "rename", srcExistingFile, dstExistingDir, avfs.ErrFileExists, err)
	})
}

// Stat tests Stat function.
func (sfs *SuiteFS) Stat(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Stat(rootDir)
		CheckPathError(t, "Stat", "stat", rootDir, avfs.ErrPermDenied, err)

		return
	}

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

	t.Run("StatNonExistingFile", func(t *testing.T) {
		nonExistingFile := vfs.Join(rootDir, "nonExistingFile")

		_, err := vfs.Stat(nonExistingFile)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "CreateFile", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		default:
			CheckPathError(t, "Stat", "stat", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
		}
	})

	t.Run("StatsubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(rootDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Stat(subDirOnFile)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Stat", "CreateFile", subDirOnFile, avfs.ErrNotADirectory, err)
		default:
			CheckPathError(t, "Stat", "stat", subDirOnFile, avfs.ErrNotADirectory, err)
		}
	})
}

// Symlink tests Symlink function.
func (sfs *SuiteFS) Symlink(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatSymlink) {
		err := vfs.Symlink(rootDir, rootDir)
		CheckLinkError(t, "Symlink", "symlink", rootDir, rootDir, avfs.ErrPermDenied, err)

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
func (sfs *SuiteFS) WriteOnReadOnly(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
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
func (sfs *SuiteFS) Umask(t *testing.T) {
	const umaskTest = 0o077

	vfs := sfs.GetFsWrite()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		vfs.UMask(0)

		return
	}

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
