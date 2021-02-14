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
	"github.com/avfs/avfs/vfsutils"
)

// TestPerm runs all file systems permission tests.
func (sfs *SuiteFS) TestPerm(t *testing.T) {
	sfs.TestPermRead(t)
	sfs.TestPermWrite(t)
}

// TestPermWrite runs all file systems permission tests with write access.
func (sfs *SuiteFS) TestPermWrite(t *testing.T) {
	sfs.TestChown(t)
	sfs.TestLchown(t)
	sfs.TestChmod(t)
	sfs.TestWriteDenied(t)
	sfs.TestChroot(t)
}

// TestPermRead runs all file systems permission tests with read access.
func (sfs *SuiteFS) TestPermRead(t *testing.T) {
	sfs.TestAccessDir(t)
	sfs.TestAccessFile(t)
	sfs.TestStatT(t)
}

// TestChown tests Chown function.
func (sfs *SuiteFS) TestChown(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !sfs.canTestPerm {
		err := vfs.Chown(rootDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Chown", "chown", rootDir, avfs.ErrPermDenied, err)
		}

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	uis := UserInfos()

	t.Run("ChownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

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
				statT := vfsutils.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("ChownFile", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

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
				statT := vfsutils.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("ChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t)

		err := vfs.Chown(nonExistingFile, 0, 0)
		CheckPathError(t, "Chown", "chown", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})
}

// Lchown tests Lchown function.
func (sfs *SuiteFS) TestLchown(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !sfs.canTestPerm {
		err := vfs.Lchown(rootDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrWinNotSupported, err)
		default:
			CheckPathError(t, "Lchown", "lchown", rootDir, avfs.ErrPermDenied, err)
		}

		return
	}

	dirs := CreateDirs(t, vfs, rootDir)
	files := CreateFiles(t, vfs, rootDir)
	symlinks := CreateSymlinks(t, vfs, rootDir)
	uis := UserInfos()

	t.Run("LchownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

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
				statT := vfsutils.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LchownFile", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

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
				statT := vfsutils.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LchownSymlink", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

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
				statT := vfsutils.AsStatT(sys)

				uid, gid := int(statT.Uid), int(statT.Gid)
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t)

		err := vfs.Lchown(nonExistingFile, 0, 0)
		CheckPathError(t, "Lchown", "lchown", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})
}

// TestChmod tests Chmod function.
func (sfs *SuiteFS) TestChmod(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Chmod(rootDir, avfs.DefaultDirPerm)
		CheckPathError(t, "Chmod", "chmod", rootDir, avfs.ErrPermDenied, err)

		return
	}

	existingFile := sfs.CreateEmptyFile(t)

	if vfs.HasFeature(avfs.FeatReadOnly) {
		vfsR := sfs.vfsRead

		err := vfsR.Chmod(existingFile, avfs.DefaultFilePerm)
		CheckPathError(t, "Chmod", "chmod", existingFile, avfs.ErrPermDenied, err)

		return
	}

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
		for shift := 6; shift >= 0; shift -= 3 {
			for mode := os.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift

				err := vfs.Chmod(existingFile, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", existingFile, err)
				}

				fst, err := vfs.Stat(existingFile)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", existingFile, err)
				}

				gotMode := fst.Mode() & os.ModePerm
				if gotMode != wantMode {
					t.Errorf("Stat %s : want mode to be %03o, got %03o", existingFile, wantMode, gotMode)
				}
			}
		}
	})

	t.Run("ChmodNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t)

		err := vfs.Chmod(nonExistingFile, avfs.DefaultDirPerm)
		CheckPathError(t, "Chmod", "chmod", nonExistingFile, avfs.ErrNoSuchFileOrDir, err)
	})

	// Cleanup permissions for RemoveAll()
	_ = vfs.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfs.Chmod(path, 0o777)

		return nil
	})
}

// TestChroot tests Chroot function.
func (sfs *SuiteFS) TestChroot(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfs := sfs.vfsWrite

	if !sfs.canTestPerm || !vfs.HasFeature(avfs.FeatChroot) {
		err := vfs.Chroot(rootDir)
		CheckPathError(t, "Chroot", "chroot", rootDir, avfs.ErrPermDenied, err)

		return
	}

	t.Run("ChrootOnFile", func(t *testing.T) {
		existingFile := sfs.CreateEmptyFile(t)
		nonExistingFile := vfs.Join(existingFile, "invalid", "path")

		err := vfs.Chroot(existingFile)
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

// TestAccessDir tests functions on directories where read is denied.
func (sfs *SuiteFS) TestAccessDir(t *testing.T) {
	if !sfs.canTestPerm {
		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	const baseDir = "baseDir"

	vfsWrite := sfs.vfsWrite

	uis := UserInfos()

	ut, err := vfsWrite.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)
			path := vfsWrite.Join(rootDir, fileName)

			err = vfsWrite.Mkdir(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("Mkdir %s : want error to be nil, got %v", path, err)
			}

			_, err = vfsWrite.Stat(path)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err := vfsWrite.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = vfsWrite.Chown(path, ut.Uid(), ut.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name
			vfs, _ := sfs.VFSAsUser(t, wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := os.FileMode(1); mode <= 6; mode++ {
					wantMode := mode << shift
					fileName := fmt.Sprintf("%s-%03o", ut.Name(), wantMode)

					path := vfs.Join(rootDir, fileName)
					info, err := vfs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := vfsutils.CheckPermission(info, avfs.WantWrite|avfs.WantLookup, vfs.CurrentUser())
					canRead := vfsutils.CheckPermission(info, avfs.WantLookup, vfs.CurrentUser())

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
	_ = vfsWrite.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfsWrite.Chmod(path, 0o777)

		return nil
	})
}

// TestAccessFile tests functions on files where read is denied.
func (sfs *SuiteFS) TestAccessFile(t *testing.T) {
	if !sfs.canTestPerm {
		return
	}

	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfsWrite := sfs.vfsWrite

	uis := UserInfos()

	usrTest, err := vfsWrite.LookupUser(UsrTest)
	if err != nil {
		t.Fatalf("LookupUser %s : want error to be nil, got %v", UsrTest, err)
	}

	for shift := 6; shift >= 0; shift -= 3 {
		for mode := os.FileMode(1); mode <= 6; mode++ {
			wantMode := mode << shift
			name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
			path := vfsWrite.Join(rootDir, name)

			err = vfsWrite.WriteFile(path, nil, avfs.DefaultFilePerm)
			if err != nil {
				t.Fatalf("WriteFile %s : want error to be nil, got %v", path, err)
			}

			err = vfsWrite.Chmod(path, wantMode)
			if err != nil {
				t.Fatalf("Chmod %s : want error to be nil, got %v", path, err)
			}

			err = vfsWrite.Chown(path, usrTest.Uid(), usrTest.Gid())
			if err != nil {
				t.Fatalf("Chown %s : want error to be nil, got %v", path, err)
			}
		}
	}

	t.Run("AccessFile", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name
			vfs, _ := sfs.VFSAsUser(t, wantName)

			for shift := 6; shift >= 0; shift -= 3 {
				for mode := 1; mode <= 6; mode++ {
					wantMode := mode << shift
					name := fmt.Sprintf("%s-%03o", usrTest.Name(), wantMode)
					path := vfs.Join(rootDir, name)

					info, err := vfs.Stat(path)
					if err != nil {
						t.Fatalf("Stat %s : want error to be nil, got %v ", path, err)
					}

					canWrite := vfsutils.CheckPermission(info, avfs.WantWrite, vfs.CurrentUser())
					canRead := vfsutils.CheckPermission(info, avfs.WantRead, vfs.CurrentUser())

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
	_ = vfsWrite.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		_ = vfsWrite.Chmod(path, 0o777)

		return nil
	})
}

// TestStatT tests os.FileInfo.Stat().Sys() Uid and Gid values.
func (sfs *SuiteFS) TestStatT(t *testing.T) {
	vfs := sfs.vfsWrite

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	info, err := vfs.Stat(vfs.GetTempDir())
	if err != nil {
		t.Errorf("Stat : want error be nil, got %v", err)
	}

	wantUid, wantGid := uint32(0), uint32(0)
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		wantUid, wantGid = math.MaxUint32, math.MaxUint32
	}

	statT := vfsutils.AsStatT(info.Sys())
	if statT.Uid != wantUid || statT.Gid != wantGid {
		t.Errorf("AsStatT : want Uid = %d, Gid = %d, got Uid = %d, Gid = %d",
			wantUid, wantGid, statT.Uid, statT.Gid)
	}
}

// TestWriteDenied tests functions on directories and files where write is denied.
func (sfs *SuiteFS) TestWriteDenied(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, avfs.UsrRoot)
	defer removeDir()

	vfsWrite := sfs.vfsWrite

	if !vfsWrite.HasFeature(avfs.FeatIdentityMgr) {
		return
	}

	pathDir := vfsWrite.Join(rootDir, "testDir")
	pathNewDirOrFile := vfsWrite.Join(pathDir, "NewDirOrFile")
	pathDirChild := vfsWrite.Join(pathDir, "DirChild")
	pathFile := vfsWrite.Join(rootDir, "File.txt")

	err := vfsWrite.MkdirAll(pathDirChild, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("Mkdir %s : want error to be nil, got %v", pathDir, err)
	}

	err = vfsWrite.Chmod(pathDir, 0o555)
	if err != nil {
		t.Fatalf("Chmod %s : want error to be nil, got %v", pathDir, err)
	}

	err = vfsWrite.WriteFile(pathFile, nil, 0o555)
	if err != nil {
		t.Fatalf("WriteFile %s : want error to be nil, got %v", pathDir, err)
	}

	t.Run("WriteDenied", func(t *testing.T) {
		for _, ui := range UserInfos() {
			vfs, u := sfs.VFSAsUser(t, ui.Name)

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

			CheckLinkError(t, "Link", "link", pathFile, pathNewDirOrFile, avfs.ErrOpNotPermitted, err)

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
				CheckLinkError(t, "TestSymlink", "symlink", pathDir, pathNewDirOrFile, avfs.ErrPermDenied, err)
			}
		}
	})
}
