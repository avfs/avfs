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
	"math"
	"os"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// Perm runs all file systems permission tests.
func (sfs *SuiteFs) Perm() {
	sfs.PermRead()
	sfs.PermWrite()
}

// PermWrite runs all file systems permission tests with write access.
func (sfs *SuiteFs) PermWrite() {
	t := sfs.t

	if !sfs.canTestPerm {
		t.Log("Info - Fs Perm Write : skipping tests.\nuse 'avfs dockertest' to run tests a root.")
		return
	}

	sfs.Chown()
	sfs.Lchown()
	sfs.Chmod()
	sfs.SuiteWriteDenied()
	sfs.Chroot()
}

// PermRead runs all file systems permission tests with read access.
func (sfs *SuiteFs) PermRead() {
	t := sfs.t

	if !sfs.canTestPerm {
		t.Log("Info - Fs Perm Read : skipping tests.\nuse 'avfs dockertest' to run tests a root.")
		return
	}

	sfs.AccessDir()
	sfs.AccessFile()
	sfs.StatT()
}

// Chown tests Chown function.
func (sfs *SuiteFs) Chown() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.GetFsRoot()
	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	users := GetUsers()

	t.Run("ChownDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := vfs.Join(rootDir, dir.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
					continue
				}

				sys := fst.Sys()
				statT := fsutil.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("ChownFile", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := vfs.Join(rootDir, file.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
					continue
				}

				sys := fst.Sys()
				statT := fsutil.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})
}

// Lchown tests Lchown function.
func (sfs *SuiteFs) Lchown() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.GetFsRoot()

	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)
	users := GetUsers()

	t.Run("LchownDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := vfs.Join(rootDir, dir.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
					continue
				}

				sys := fst.Sys()
				statT := fsutil.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LchownFile", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := vfs.Join(rootDir, file.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
					continue
				}

				sys := fst.Sys()
				statT := fsutil.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LchownSymlink", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, symlink := range symlinks {
				path := vfs.Join(rootDir, symlink.NewName)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
					continue
				}

				sys := fst.Sys()
				statT := fsutil.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})
}

// Chmod tests Chmod function.
func (sfs *SuiteFs) Chmod() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.GetFsRoot()

	t.Run("ChmodDir", func(t *testing.T) {
		for shift := 6; shift >= 0; shift -= 3 {
			for mode := os.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift
				path, err := vfs.TempDir(rootDir, "")
				if err != nil {
					t.Fatalf("TempDir %s : want error to be nil, got %v", rootDir, err)
				}

				err = vfs.Chmod(path, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				}

				gotMode := fst.Mode() & os.ModePerm
				if gotMode != wantMode {
					t.Errorf("Stat %s : want mode to be %03o, got %03o", path, wantMode, gotMode)
				}
			}
		}
	})

	t.Run("ChmodFile", func(t *testing.T) {
		path := vfs.Join(rootDir, "existingFile")

		err := vfs.WriteFile(path, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		for shift := 6; shift >= 0; shift -= 3 {
			for mode := os.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift

				err := vfs.Chmod(path, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				}

				gotMode := fst.Mode() & os.ModePerm
				if gotMode != wantMode {
					t.Errorf("Stat %s : want mode to be %03o, got %03o", path, wantMode, gotMode)
				}
			}
		}
	})

	// Cleanup permissions for RemoveAll()
	_ = vfs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfs.Chmod(path, 0o777)

		return nil
	})
}

// Chroot tests Chroot function.
func (sfs *SuiteFs) Chroot() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.GetFsRoot()

	t.Run("ChrootInvalid", func(t *testing.T) {
		existingFile := vfs.Join(rootDir, "existingFile")

		err := vfs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		nonExistingFile := vfs.Join(existingFile, "invalid", "path")

		err = vfs.Chroot(existingFile)
		CheckPathError(t, "Chroot", "chroot", existingFile, avfs.ErrNotADirectory, err)

		err = vfs.Chroot(nonExistingFile)
		CheckPathError(t, "Chroot", "chroot", nonExistingFile, avfs.ErrNotADirectory, err)
	})

	t.Run("ChrootOk", func(t *testing.T) {
		// Some file systems (MemFs) don't permit exit from a chroot.
		// A shallow clone of the file system is then used to perform the chroot
		// without loosing access to the original root of the file system.
		fsSave := vfs.Clone()

		chrootDir := vfs.Join(rootDir, "chroot")

		err := vfs.Mkdir(chrootDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("mkdir %s : want error to be nil, got %v", chrootDir, err)
		}

		const chrootFile = "/file-within-the-chroot.txt"
		chrootFilePath := vfs.Join(chrootDir, chrootFile)

		err = vfs.WriteFile(chrootFilePath, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", chrootFilePath, err)
		}

		// A file descriptor is used to save the real root of the file system.
		// See https://devsidestory.com/exit-from-a-chroot-with-golang/
		fSave, err := vfs.Open("/")
		if err != nil {
			t.Fatalf("Open / : want error to be nil, got %v", err)
		}

		defer fSave.Close()

		err = vfs.Chroot(chrootDir)
		if err != nil {
			t.Errorf("Chroot : want error to be nil, got %v", err)
		}

		_, err = vfs.Stat(chrootFile)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		// if the file system can be cloned it can be restored from the saved one.
		if vfs.HasFeature(avfs.FeatClonable) {
			vfs = fsSave

			return
		}

		// Restore the original file system root if possible.
		err = fSave.Chdir()
		if err != nil {
			t.Errorf("Chdir : want error to be nil, got %v", err)
		}

		err = vfs.Chroot(".")
		if err != nil {
			t.Errorf("Chroot : want error to be nil, got %v", err)
		}

		_, err = vfs.Stat(chrootFilePath)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}
	})
}

// AccessDir tests functions on directories where read is denied.
func (sfs *SuiteFs) AccessDir() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	const baseDir = "baseDir"

	vfsRoot := sfs.GetFsRoot()
	users := GetUsers()

	ut, err := vfsRoot.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)
			path := vfsRoot.Join(rootDir, fileName)

			err = vfsRoot.Mkdir(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("Mkdir %s : want error to be nil, got %v", path, err)
			}

			_, err = vfsRoot.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err := vfsRoot.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = vfsRoot.Chown(path, ut.Uid(), ut.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			vfs, _ := sfs.GetFsAsUser(wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := os.FileMode(1); mode <= 6; mode++ {
					wantMode := mode << shift
					fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)

					path := vfs.Join(rootDir, fileName)
					info, err := vfs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := fsutil.CheckPermission(info, avfs.WantWrite|avfs.WantLookup, vfs.CurrentUser())
					canRead := fsutil.CheckPermission(info, avfs.WantLookup, vfs.CurrentUser())

					path = vfs.Join(rootDir, fileName, baseDir)
					err = vfs.Mkdir(path, avfs.DefaultDirPerm)
					if canWrite {
						if err != nil {
							t.Errorf("Mkdir %s : want error to be nil, got %v", path, err)
						}
					} else {
						CheckPathError(t, "Mkdir", "mkdir", path, avfs.ErrPermDenied, err)
					}

					_, err = vfs.Stat(path)
					switch {
					case canRead && canWrite:
						if err != nil {
							t.Errorf("Stat %s : want error to be nil, got %v", path, err)
						}
					case canRead:
						CheckPathError(t, "Stat", "stat", path, avfs.ErrNoSuchFileOrDir, err)
					default:
						CheckPathError(t, "Stat", "stat", path, avfs.ErrPermDenied, err)
					}
				}
			}
		}
	})

	// Cleanup permissions for RemoveAll()
	_ = vfsRoot.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfsRoot.Chmod(path, 0o777)

		return nil
	})
}

// AccessFile tests functions on files where read is denied.
func (sfs *SuiteFs) AccessFile() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfsRoot := sfs.GetFsRoot()
	users := GetUsers()

	usrTest, err := vfsRoot.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
			path := vfsRoot.Join(rootDir, name)

			err = vfsRoot.WriteFile(path, nil, avfs.DefaultFilePerm)
			if err != nil {
				t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
			}

			err = vfsRoot.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = vfsRoot.Chown(path, usrTest.Uid(), usrTest.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessFile", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			vfs, _ := sfs.GetFsAsUser(wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := 1; mode <= 6; mode++ {
					wantMode := mode << shift
					name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
					path := vfs.Join(rootDir, name)

					info, err := vfs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := fsutil.CheckPermission(info, avfs.WantWrite, vfs.CurrentUser())
					canRead := fsutil.CheckPermission(info, avfs.WantRead, vfs.CurrentUser())

					err = vfs.WriteFile(path, []byte("WriteFile"), os.ModePerm)
					if canWrite {
						if err != nil {
							t.Errorf("WriteFile %s : want error to be nil, got %v", path, err)
						}
					} else {
						CheckPathError(t, "WriteFile", "open", path, avfs.ErrPermDenied, err)
					}

					_, err = vfs.ReadFile(path)
					if canRead {
						if err != nil {
							t.Errorf("ReadFile %s : want error to be nil, got %v", path, err)
						}
					} else {
						CheckPathError(t, "ReadFile", "open", path, avfs.ErrPermDenied, err)
					}
				}
			}
		}
	})

	// Cleanup permissions for RemoveAll()
	_ = vfsRoot.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfsRoot.Chmod(path, 0o777)

		return nil
	})
}

// StatT tests os.FileInfo.Stat().Sys() Uid and Gid values.
func (sfs *SuiteFs) StatT() {
	t := sfs.t
	vfs := sfs.GetFsWrite()

	info, err := vfs.Stat(vfs.GetTempDir())
	if err != nil {
		t.Errorf("Stat : want error be nil, got %v", err)
	}

	wantUid, wantGid := uint32(0), uint32(0)
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		wantUid, wantGid = math.MaxUint32, math.MaxUint32
	}

	statT := fsutil.AsStatT(info.Sys())
	if statT.Uid != wantUid || statT.Gid != wantGid {
		t.Errorf("AsStatT : want Uid = %d, Gid = %d, got Uid = %d, Gid = %d",
			wantUid, wantGid, statT.Uid, statT.Gid)
	}
}

// SuiteWriteDenied tests functions on directories and files where write is denied.
func (sfs *SuiteFs) SuiteWriteDenied() {
	t, rootDir, removeDir := sfs.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	vfsRoot := sfs.GetFsRoot()
	pathDir := vfsRoot.Join(rootDir, "testDir")
	pathNewDirOrFile := vfsRoot.Join(pathDir, "NewDirOrFile")
	pathDirChild := vfsRoot.Join(pathDir, "DirChild")
	pathFile := vfsRoot.Join(rootDir, "File.txt")

	err := vfsRoot.MkdirAll(pathDirChild, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir %s : want error to be nil, got %v", pathDir, err)
	}

	err = vfsRoot.Chmod(pathDir, 0o555)
	if err != nil {
		t.Fatalf("Chmod %s : want error to be nil, got %v", pathDir, err)
	}

	err = vfsRoot.WriteFile(pathFile, nil, 0o555)
	if err != nil {
		t.Fatalf("WriteFile %s : want error to be nil, got %v", pathDir, err)
	}

	t.Run("WriteDenied", func(t *testing.T) {
		for _, u := range GetUsers() {
			vfs, u := sfs.GetFsAsUser(u.Name)

			err := vfs.Chmod(pathDir, avfs.DefaultDirPerm)
			CheckPathError(t, "Chmod", "chmod", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Chown(pathDir, u.Uid(), u.Gid())
			CheckPathError(t, "Chown", "chown", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Chtimes(pathDir, time.Now(), time.Now())
			CheckPathError(t, "Chtimes", "chtimes", pathDir, avfs.ErrOpNotPermitted, err)

			_, err = vfs.Create(pathNewDirOrFile)
			CheckPathError(t, "Create", "open", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = vfs.Chroot(pathDir)
			CheckPathError(t, "Chroot", "chroot", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Lchown(pathDir, u.Uid(), u.Gid())
			CheckPathError(t, "Lchown", "lchown", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Link(pathFile, pathNewDirOrFile)

			wantErr := avfs.ErrOpNotPermitted
			if vfs.OSType() == avfs.OsLinuxWSL {
				wantErr = avfs.ErrPermDenied
			}

			CheckLinkError(t, "Link", "link", pathFile, pathNewDirOrFile, wantErr, err)

			err = vfs.Mkdir(pathNewDirOrFile, avfs.DefaultDirPerm)
			CheckPathError(t, "Mkdir", "mkdir", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = vfs.MkdirAll(pathNewDirOrFile, avfs.DefaultDirPerm)
			CheckPathError(t, "MkdirAll", "mkdir", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = vfs.Remove(pathDirChild)
			CheckPathError(t, "Remove", "remove", pathDirChild, avfs.ErrPermDenied, err)

			err = vfs.RemoveAll(pathDirChild)
			CheckPathError(t, "RemoveAll", "unlinkat", pathDirChild, avfs.ErrPermDenied, err)

			err = vfs.Rename(pathFile, pathNewDirOrFile)
			CheckLinkError(t, "Rename", "rename", pathFile, pathNewDirOrFile, avfs.ErrPermDenied, err)

			if vfs.HasFeature(avfs.FeatSymlink) {
				err = vfs.Symlink(pathDir, pathNewDirOrFile)
				CheckLinkError(t, "Symlink", "symlink", pathDir, pathNewDirOrFile, avfs.ErrPermDenied, err)
			}
		}
	})
}
