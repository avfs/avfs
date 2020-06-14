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

// SuitePerm runs all file systems permission tests.
func (cf *ConfigFs) SuitePerm() {
	cf.SuitePermRead()
	cf.SuitePermWrite()
}

// SuitePermWrite runs all file systems permission tests with write access.
func (cf *ConfigFs) SuitePermWrite() {
	t := cf.t

	if !cf.canTestPerm {
		t.Log("Info - Fs Perm Write : skipping tests.\nuse 'make dockertest' to run tests a root.")
		return
	}

	cf.SuiteChown()
	cf.SuiteLchown()
	cf.SuiteChmod()
	cf.SuiteWriteDenied()
	cf.SuiteChroot()
}

// SuitePermRead runs all file systems permission tests with read access.
func (cf *ConfigFs) SuitePermRead() {
	t := cf.t

	if !cf.canTestPerm {
		t.Log("Info - Fs Perm Read : skipping tests.\nuse 'make dockertest' to run tests a root.")
		return
	}

	cf.SuiteAccessDir()
	cf.SuiteAccessFile()
	cf.SuiteStatT()
}

// SuiteChown tests Chown function.
func (cf *ConfigFs) SuiteChown() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fs := cf.GetFsRoot()
	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	users := GetUsers()

	t.Run("ChownDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			u, err := fs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := fs.Join(rootDir, dir.Path)
				err := fs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Stat(path)
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
			u, err := fs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := fs.Join(rootDir, file.Path)

				err := fs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Stat(path)
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

// SuiteLchown tests Lchown function.
func (cf *ConfigFs) SuiteLchown() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fs := cf.GetFsRoot()

	dirs := CreateDirs(t, fs, rootDir)
	files := CreateFiles(t, fs, rootDir)
	symlinks := CreateSymlinks(t, fs, rootDir)
	users := GetUsers()

	t.Run("LchownDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			u, err := fs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := fs.Join(rootDir, dir.Path)
				err := fs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Lstat(path)
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
			u, err := fs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := fs.Join(rootDir, file.Path)

				err := fs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Lstat(path)
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
			u, err := fs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, symlink := range symlinks {
				path := fs.Join(rootDir, symlink.NewName)

				err := fs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Lstat(path)
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

// SuiteChmod tests Chmod function.
func (cf *ConfigFs) SuiteChmod() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fs := cf.GetFsRoot()

	t.Run("ChmodDir", func(t *testing.T) {
		for shift := 6; shift >= 0; shift -= 3 {
			for mode := os.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift
				path, err := fs.TempDir(rootDir, "")
				if err != nil {
					t.Fatalf("TempDir %s : want error to be nil, got %v", rootDir, err)
				}

				err = fs.Chmod(path, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Stat(path)
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
		path := fs.Join(rootDir, "existingFile")

		err := fs.WriteFile(path, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		for shift := 6; shift >= 0; shift -= 3 {
			for mode := os.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift

				err := fs.Chmod(path, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", path, err)
				}

				fst, err := fs.Stat(path)
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
	_ = fs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = fs.Chmod(path, 0o777)

		return nil
	})
}

// SuiteChroot tests Chroot function.
func (cf *ConfigFs) SuiteChroot() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fs := cf.GetFsRoot()

	t.Run("ChrootInvalid", func(t *testing.T) {
		existingFile := fs.Join(rootDir, "existingFile")

		err := fs.WriteFile(existingFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want error to be nil, got %v", err)
		}

		nonExistingFile := fs.Join(existingFile, "invalid", "path")

		err = fs.Chroot(existingFile)
		CheckPathError(t, "Chroot", "chroot", existingFile, avfs.ErrNotADirectory, err)

		err = fs.Chroot(nonExistingFile)
		CheckPathError(t, "Chroot", "chroot", nonExistingFile, avfs.ErrNotADirectory, err)
	})

	t.Run("ChrootOk", func(t *testing.T) {
		// Some file systems (MemFs) don't permit exit from a chroot.
		// A shallow clone of the file system is then used to perform the chroot
		// without loosing access to the original root of the file system.
		fsC, ok := fs.(avfs.Cloner)
		if ok {
			fs = fsC.Clone()
		}

		chrootDir := fs.Join(rootDir, "chroot")

		err := fs.Mkdir(chrootDir, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("mkdir %s : want error to be nil, got %v", chrootDir, err)
		}

		const chrootFile = "/file-within-the-chroot.txt"
		chrootFilePath := fs.Join(chrootDir, chrootFile)

		err = fs.WriteFile(chrootFilePath, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile %s : want error to be nil, got %v", chrootFilePath, err)
		}

		// A file descriptor is used to save the real root of the file system.
		// See https://devsidestory.com/exit-from-a-chroot-with-golang/
		fSave, err := fs.Open("/")
		if err != nil {
			t.Fatalf("Open / : want error to be nil, got %v", err)
		}

		defer fSave.Close()

		err = fs.Chroot(chrootDir)
		if err != nil {
			t.Errorf("Chroot : want error to be nil, got %v", err)
		}

		_, err = fs.Stat(chrootFile)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		// Restore the original file system root if possible.
		if !fs.HasFeature(avfs.FeatInescapableChroot) {
			err = fSave.Chdir()
			if err != nil {
				t.Errorf("Chdir : want error to be nil, got %v", err)
			}

			err = fs.Chroot(".")
			if err != nil {
				t.Errorf("Chroot : want error to be nil, got %v", err)
			}

			_, err = fs.Stat(chrootFilePath)
			if err != nil {
				t.Errorf("Stat : want error to be nil, got %v", err)
			}
		}
	})
}

// SuiteAccessDir tests functions on directories where read is denied.
func (cf *ConfigFs) SuiteAccessDir() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	const baseDir = "baseDir"

	fsRoot := cf.GetFsRoot()
	users := GetUsers()

	ut, err := fsRoot.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)
			path := fsRoot.Join(rootDir, fileName)

			err = fsRoot.Mkdir(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("Mkdir %s : want error to be nil, got %v", path, err)
			}

			_, err = fsRoot.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err := fsRoot.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = fsRoot.Chown(path, ut.Uid(), ut.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessDir", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			fs, _ := cf.GetFsAsUser(wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := os.FileMode(1); mode <= 6; mode++ {
					wantMode := mode << shift
					fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)

					path := fs.Join(rootDir, fileName)
					info, err := fs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := fsutil.CheckPermission(info, avfs.WantWrite|avfs.WantLookup, fs.CurrentUser())
					canRead := fsutil.CheckPermission(info, avfs.WantLookup, fs.CurrentUser())

					path = fs.Join(rootDir, fileName, baseDir)
					err = fs.Mkdir(path, avfs.DefaultDirPerm)
					if canWrite {
						if err != nil {
							t.Errorf("Mkdir %s : want error to be nil, got %v", path, err)
						}
					} else {
						CheckPathError(t, "Mkdir", "mkdir", path, avfs.ErrPermDenied, err)
					}

					_, err = fs.Stat(path)
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
	_ = fsRoot.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = fsRoot.Chmod(path, 0o777)

		return nil
	})
}

// SuiteAccessFile tests functions on files where read is denied.
func (cf *ConfigFs) SuiteAccessFile() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fsRoot := cf.GetFsRoot()
	users := GetUsers()

	usrTest, err := fsRoot.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
			path := fsRoot.Join(rootDir, name)

			err = fsRoot.WriteFile(path, nil, avfs.DefaultFilePerm)
			if err != nil {
				t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
			}

			err = fsRoot.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = fsRoot.Chown(path, usrTest.Uid(), usrTest.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessFile", func(t *testing.T) {
		for _, user := range users {
			wantName := user.Name
			fs, _ := cf.GetFsAsUser(wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := 1; mode <= 6; mode++ {
					wantMode := mode << shift
					name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
					path := fs.Join(rootDir, name)

					info, err := fs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := fsutil.CheckPermission(info, avfs.WantWrite, fs.CurrentUser())
					canRead := fsutil.CheckPermission(info, avfs.WantRead, fs.CurrentUser())

					err = fs.WriteFile(path, []byte("WriteFile"), os.ModePerm)
					if canWrite {
						if err != nil {
							t.Errorf("WriteFile %s : want error to be nil, got %v", path, err)
						}
					} else {
						CheckPathError(t, "WriteFile", "open", path, avfs.ErrPermDenied, err)
					}

					_, err = fs.ReadFile(path)
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
	_ = fsRoot.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = fsRoot.Chmod(path, 0o777)

		return nil
	})
}

// SuiteStatT tests os.FileInfo.Stat().Sys() Uid and Gid values.
func (cf *ConfigFs) SuiteStatT() {
	t := cf.t
	fs := cf.GetFsWrite()

	info, err := fs.Stat(avfs.TmpDir)
	if err != nil {
		t.Errorf("Stat : want error be nil, got %v", err)
	}

	wantUid, wantGid := uint32(0), uint32(0)
	if !fs.HasFeature(avfs.FeatIdentityMgr) {
		wantUid, wantGid = math.MaxUint32, math.MaxUint32
	}

	statT := fsutil.AsStatT(info.Sys())
	if statT.Uid != wantUid || statT.Gid != wantGid {
		t.Errorf("AsStatT : want Uid = %d, Gid = %d, got Uid = %d, Gid = %d",
			wantUid, wantGid, statT.Uid, statT.Gid)
	}
}

// SuiteWriteDenied tests functions on directories and files where write is denied.
func (cf *ConfigFs) SuiteWriteDenied() {
	t, rootDir, removeDir := cf.CreateRootDir(avfs.UsrRoot)
	defer removeDir()

	fsRoot := cf.GetFsRoot()
	pathDir := fsRoot.Join(rootDir, "testDir")
	pathNewDirOrFile := fsRoot.Join(pathDir, "NewDirOrFile")
	pathDirChild := fsRoot.Join(pathDir, "DirChild")
	pathFile := fsRoot.Join(rootDir, "File.txt")

	err := fsRoot.MkdirAll(pathDirChild, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir %s : want error to be nil, got %v", pathDir, err)
	}

	err = fsRoot.Chmod(pathDir, 0o555)
	if err != nil {
		t.Fatalf("Chmod %s : want error to be nil, got %v", pathDir, err)
	}

	err = fsRoot.WriteFile(pathFile, nil, 0o555)
	if err != nil {
		t.Fatalf("WriteFile %s : want error to be nil, got %v", pathDir, err)
	}

	t.Run("WriteDenied", func(t *testing.T) {
		for _, u := range GetUsers() {
			fs, u := cf.GetFsAsUser(u.Name)

			err := fs.Chmod(pathDir, avfs.DefaultDirPerm)
			CheckPathError(t, "Chmod", "chmod", pathDir, avfs.ErrOpNotPermitted, err)

			err = fs.Chown(pathDir, u.Uid(), u.Gid())
			CheckPathError(t, "Chown", "chown", pathDir, avfs.ErrOpNotPermitted, err)

			err = fs.Chtimes(pathDir, time.Now(), time.Now())
			CheckPathError(t, "Chtimes", "chtimes", pathDir, avfs.ErrOpNotPermitted, err)

			_, err = fs.Create(pathNewDirOrFile)
			CheckPathError(t, "Create", "open", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = fs.Chroot(pathDir)
			CheckPathError(t, "Chroot", "chroot", pathDir, avfs.ErrOpNotPermitted, err)

			err = fs.Lchown(pathDir, u.Uid(), u.Gid())
			CheckPathError(t, "Lchown", "lchown", pathDir, avfs.ErrOpNotPermitted, err)

			err = fs.Link(pathFile, pathNewDirOrFile)
			CheckLinkError(t, "Link", "link", pathFile, pathNewDirOrFile, avfs.ErrOpNotPermitted, err)

			err = fs.Mkdir(pathNewDirOrFile, avfs.DefaultDirPerm)
			CheckPathError(t, "Mkdir", "mkdir", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = fs.MkdirAll(pathNewDirOrFile, avfs.DefaultDirPerm)
			CheckPathError(t, "MkdirAll", "mkdir", pathNewDirOrFile, avfs.ErrPermDenied, err)

			err = fs.Remove(pathDirChild)
			CheckPathError(t, "Remove", "remove", pathDirChild, avfs.ErrPermDenied, err)

			err = fs.RemoveAll(pathDirChild)
			CheckPathError(t, "RemoveAll", "unlinkat", pathDirChild, avfs.ErrPermDenied, err)

			err = fs.Rename(pathFile, pathNewDirOrFile)
			CheckLinkError(t, "Rename", "rename", pathFile, pathNewDirOrFile, avfs.ErrPermDenied, err)

			if fs.HasFeature(avfs.FeatSymlink) {
				err = fs.Symlink(pathDir, pathNewDirOrFile)
				CheckLinkError(t, "Symlink", "symlink", pathDir, pathNewDirOrFile, avfs.ErrPermDenied, err)
			}
		}
	})
}
