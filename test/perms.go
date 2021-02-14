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
	sfs.TestWriteDenied(t)
}

// TestPermRead runs all file systems permission tests with read access.
func (sfs *SuiteFS) TestPermRead(t *testing.T) {
	sfs.TestAccessDir(t)
	sfs.TestAccessFile(t)
	sfs.TestStatT(t)
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

			//	err = vfs.Chroot(pathDir)
			//	CheckPathError(t, "Chroot", "chroot", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Lchown(pathDir, u.Uid(), u.Gid())
			CheckPathError(t, "Lchown", "lchown", pathDir, avfs.ErrOpNotPermitted, err)

			err = vfs.Link(pathFile, pathNewDirOrFile)

			CheckLinkError(t, "Link", "link", pathFile, pathNewDirOrFile, avfs.ErrOpNotPermitted, err)

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
