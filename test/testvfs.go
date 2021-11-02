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
	"bytes"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

// TestChdir tests Chdir and Getwd functions.
func (sfs *SuiteFS) TestChdir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		err := vfs.Chdir(testDir)
		CheckPathError(t, err).Op("chdir").Path(testDir).Err(avfs.ErrPermDenied)

		_, err = vfs.Getwd()
		CheckPathError(t, err).Op("getwd").Path("").Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	existingFile := sfs.EmptyFile(t, testDir)

	t.Run("ChdirAbsolute", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			err := vfs.Chdir(path)
			CheckNoError(t, "Chdir "+path, err)

			curDir, err := vfs.Getwd()
			CheckNoError(t, "Getwd", err)

			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})

	t.Run("ChdirRelative", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(testDir)
			if !CheckNoError(t, "Chdir "+testDir, err) {
				return
			}

			relPath := dir.Path[1:]

			err = vfs.Chdir(relPath)
			CheckNoError(t, "Chdir "+relPath, err)

			curDir, err := vfs.Getwd()
			CheckNoError(t, "Getwd", err)

			path := vfs.Join(testDir, relPath)
			if curDir != path {
				t.Errorf("Getwd : want current directory to be %s, got %s", path, curDir)
			}
		}
	})

	t.Run("ChdirNonExisting", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path, defaultNonExisting)

			oldPath, err := vfs.Getwd()
			CheckNoError(t, "Getwd", err)

			err = vfs.Chdir(path)
			CheckPathError(t, err).Op("chdir").Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

			newPath, err := vfs.Getwd()
			CheckNoError(t, "Getwd", err)

			if newPath != oldPath {
				t.Errorf("Getwd : want current dir to be %s, got %s", oldPath, newPath)
			}
		}
	})

	t.Run("ChdirOnFile", func(t *testing.T) {
		err := vfs.Chdir(existingFile)
		CheckPathError(t, err).Op("chdir").Path(existingFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinDirNameInvalid, avfs.OsWindows)
	})

	t.Run("ChdirPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Chdir(path)
		})
	})
}

// TestChmod tests Chmod function.
func (sfs *SuiteFS) TestChmod(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Chmod(testDir, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("chmod").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	existingFile := sfs.EmptyFile(t, testDir)

	t.Run("ChmodDir", func(t *testing.T) {
		for shift := 6; shift >= 0; shift -= 3 {
			for mode := fs.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift

				path, err := vfs.MkdirTemp(testDir, "")
				if !CheckNoError(t, "MkdirTemp", err) {
					return
				}

				err = vfs.Chmod(path, wantMode)
				CheckNoError(t, "Chmod "+path, err)

				fst, err := vfs.Stat(path)
				CheckNoError(t, "Stat "+path, err)

				gotMode := fst.Mode() & fs.ModePerm

				// On Windows, only the 0200 bit (owner writable) of mode is used.
				if vfs.OSType() == avfs.OsWindows {
					wantMode &= 0o200
					gotMode &= 0o200
				}

				if gotMode != wantMode {
					t.Errorf("Stat %s : want mode to be %03o, got %03o", path, wantMode, gotMode)
				}
			}
		}
	})

	t.Run("ChmodFile", func(t *testing.T) {
		for shift := 6; shift >= 0; shift -= 3 {
			for mode := fs.FileMode(1); mode <= 6; mode++ {
				wantMode := mode << shift

				err := vfs.Chmod(existingFile, wantMode)
				CheckNoError(t, "Chmod "+existingFile, err)

				fst, err := vfs.Stat(existingFile)
				CheckNoError(t, "Stat "+existingFile, err)

				gotMode := fst.Mode() & fs.ModePerm

				// On Windows, only the 0200 bit (owner writable) of mode is used.
				if vfs.OSType() == avfs.OsWindows {
					wantMode &= 0o200
					gotMode &= 0o200
				}

				if gotMode != wantMode {
					t.Errorf("Stat %s : want mode to be %03o, got %03o", existingFile, wantMode, gotMode)
				}
			}
		}
	})

	t.Run("ChmodNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Chmod(nonExistingFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("chmod").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("ChmodPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Chmod(path, 0o777)
		})
	})
}

// TestChown tests Chown function.
func (sfs *SuiteFS) TestChown(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	idm := vfs.Idm()

	if !sfs.canTestPerm && !vfs.HasFeature(avfs.FeatChownUser) {
		err := vfs.Chown(testDir, 0, 0)

		CheckPathError(t, err).Op("chown").Path(testDir).
			Err(avfs.ErrOpNotPermitted, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)

		return
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		if vfs.HasFeature(avfs.FeatRealFS) {
			return
		}

		wantUid, wantGid := 42, 42

		err := vfs.Chown(testDir, wantUid, wantGid)
		CheckNoError(t, "Chown "+testDir, err)

		fst, err := vfs.Stat(testDir)
		CheckNoError(t, "Stat "+testDir, err)

		sst := vfs.ToSysStat(fst)

		uid, gid := sst.Uid(), sst.Gid()
		if uid != wantUid || gid != wantGid {
			t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
				testDir, wantUid, wantGid, uid, gid)
		}

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	uis := UserInfos()

	t.Run("ChownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

			u, err := idm.LookupUser(wantName)
			if !CheckNoError(t, "LookupUser "+wantName, err) {
				return
			}

			for _, dir := range dirs {
				path := vfs.Join(testDir, dir.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				CheckNoError(t, "Chown "+path, err)

				fst, err := vfs.Stat(path)
				if !CheckNoError(t, "Stat "+path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
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

			u, err := idm.LookupUser(wantName)
			if !CheckNoError(t, "LookupUser "+wantName, err) {
				return
			}

			for _, file := range files {
				path := vfs.Join(testDir, file.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				CheckNoError(t, "Chown "+path, err)

				fst, err := vfs.Stat(path)
				if !CheckNoError(t, "Stat "+path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("ChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Chown(nonExistingFile, 0, 0)
		CheckPathError(t, err).Op("chown").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("ChownPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Chown(path, 0, 0)
		})
	})
}

// TestChroot tests Chroot function.
func (sfs *SuiteFS) TestChroot(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !sfs.canTestPerm || !vfs.HasFeature(avfs.FeatChroot) {
		err := vfs.Chroot(testDir)
		CheckPathError(t, err).Op("chroot").Path(testDir).Err(avfs.ErrOpNotPermitted)

		return
	}

	t.Run("Chroot", func(t *testing.T) {
		chrootDir := vfs.Join(testDir, "chroot")

		err := vfs.Mkdir(chrootDir, avfs.DefaultDirPerm)
		if !CheckNoError(t, "Mkdir "+chrootDir, err) {
			return
		}

		const chrootFile = "/file-within-the-chroot.txt"
		chrootFilePath := vfs.Join(chrootDir, chrootFile)

		err = vfs.WriteFile(chrootFilePath, nil, avfs.DefaultFilePerm)
		if !CheckNoError(t, "WriteFile "+chrootFilePath, err) {
			return
		}

		// A file descriptor is used to save the real root of the file system.
		// See https://devsidestory.com/exit-from-a-chroot-with-golang/
		fSave, err := vfs.Open("/")
		if !CheckNoError(t, "Open /", err) {
			return
		}

		defer fSave.Close()

		// Some file systems (MemFs) don't allow exit from a chroot.
		// A shallow clone of the file system is then used to perform the chroot
		// without loosing access to the original root of the file system.
		vfsChroot := vfs

		vfsClone, cloned := vfs.(avfs.Cloner)
		if cloned {
			vfsChroot = vfsClone.Clone()
		}

		err = vfsChroot.Chroot(chrootDir)
		CheckNoError(t, "Chroot "+chrootDir, err)

		_, err = vfsChroot.Stat(chrootFile)
		CheckNoError(t, "Chroot "+chrootFile, err)

		// Restore the original file system root if possible.
		if !cloned {
			err = fSave.Chdir()
			CheckNoError(t, "Chdir", err)

			err = vfs.Chroot(".")
			CheckNoError(t, "Chroot", err)
		}

		_, err = vfs.Stat(chrootFilePath)
		CheckNoError(t, "Chroot "+chrootFilePath, err)
	})

	t.Run("ChrootOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		nonExistingFile := vfs.Join(existingFile, "invalid", "path")

		err := vfs.Chroot(existingFile)
		CheckPathError(t, err).Op("chroot").Path(existingFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux)

		err = vfs.Chroot(nonExistingFile)
		CheckPathError(t, err).Op("chroot").Path(nonExistingFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux)
	})
}

// TestChtimes tests Chtimes function.
func (sfs *SuiteFS) TestChtimes(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Chtimes(testDir, time.Now(), time.Now())
		CheckPathError(t, err).Op("chtimes").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("Chtimes", func(t *testing.T) {
		_ = sfs.SampleDirs(t, testDir)
		files := sfs.SampleFiles(t, testDir)
		tomorrow := time.Now().AddDate(0, 0, 1)

		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			err := vfs.Chtimes(path, tomorrow, tomorrow)
			CheckNoError(t, "Chtimes "+path, err)

			infos, err := vfs.Stat(path)
			CheckNoError(t, "Stat "+path, err)

			if infos.ModTime() != tomorrow {
				t.Errorf("Chtimes %s : want modtime to bo %s, got %s", path, tomorrow, infos.ModTime())
			}
		}
	})

	t.Run("ChtimesNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Chtimes(nonExistingFile, time.Now(), time.Now())
		CheckPathError(t, err).Op("chtimes").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("ChtimesPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Chtimes(path, time.Now(), time.Now())
		})
	})
}

// TestClone tests Clone function.
func (sfs *SuiteFS) TestClone(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		return
	}

	if vfsClonable, ok := vfs.(avfs.Cloner); ok {
		vfsCloned := vfsClonable.Clone()

		if _, ok := vfsCloned.(avfs.Cloner); !ok {
			t.Errorf("Clone : want cloned vfs to be of type VFS, got type %v", reflect.TypeOf(vfsCloned))
		}
	}
}

// TestCreate tests Create function.
func (sfs *SuiteFS) TestCreate(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.Create(testDir)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	t.Run("CreatePerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			newFile := vfs.Join(path, defaultFile)

			f, err := vfs.Create(newFile)
			if err == nil {
				f.Close()
			}

			return err
		})
	})
}

// TestCreateTemp tests CreateTemp function.
func (sfs *SuiteFS) TestCreateTemp(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.CreateTemp(testDir, "")
		if err.(*fs.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("CreateTemp : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	t.Run("CreateTempEmptyDir", func(t *testing.T) {
		f, err := vfs.CreateTemp("", "")
		CheckNoError(t, "CreateTemp", err)

		wantDir := vfs.TempDir()
		dir := vfs.Dir(f.Name())

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}
	})

	t.Run("CreateTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.CreateTemp("", badPattern)
		CheckPathError(t, err).Op("createtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator)
	})

	t.Run("CreateTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.CreateTemp(existingFile, "")
		CheckPathError(t, err).Op("open").
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestUser tests User function.
func (sfs *SuiteFS) TestUser(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	u := vfs.User()

	name := u.Name()
	if name == "" {
		t.Errorf("Name : want name to be not empty, got empty")
	}

	uid := u.Uid()
	if uid < 0 {
		t.Errorf("Uid : want uid to be >= 0, got %d", uid)
	}

	gid := u.Gid()
	if uid < 0 {
		t.Errorf("Uid : want gid to be >= 0, got %d", gid)
	}
}

// TestEvalSymlink tests EvalSymlink function.
func (sfs *SuiteFS) TestEvalSymlink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.EvalSymlinks(testDir)

		switch vfs.OSType() {
		case avfs.OsWindows:
			if err != avfs.ErrWinAccessDenied {
				t.Errorf("want error to be %v, got %v", avfs.ErrWinAccessDenied, err)
			}
		default:
			CheckPathError(t, err).Op("lstat").Path(testDir).Err(avfs.ErrPermDenied)
		}

		return
	}

	_ = sfs.SampleDirs(t, testDir)
	_ = sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	t.Run("EvalSymlink", func(t *testing.T) {
		symlinks := GetSampleSymlinksEval(vfs)
		for _, sl := range symlinks {
			wantPath := vfs.Join(testDir, sl.OldName)
			slPath := vfs.Join(testDir, sl.NewName)

			gotPath, err := vfs.EvalSymlinks(slPath)
			if !CheckNoError(t, "EvalSymlinks", err) {
				continue
			}

			if wantPath != gotPath {
				t.Errorf("EvalSymlinks %s : want Path to be %s, got %s", slPath, wantPath, gotPath)
			}
		}
	})
}

// TestTempDir tests TempDir function.
func (sfs *SuiteFS) TestTempDir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	wantTmpDir := os.TempDir()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		tmpDir := vfs.TempDir()
		if tmpDir != wantTmpDir {
			t.Errorf("TempDir : want error to be %v, got %v", wantTmpDir, tmpDir)
		}

		return
	}

	tmpDir := vfs.TempDir()
	if tmpDir != wantTmpDir {
		t.Fatalf("TempDir : want temp dir to be %s, got %s", wantTmpDir, tmpDir)
	}
}

// TestLchown tests Lchown function.
func (sfs *SuiteFS) TestLchown(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	idm := vfs.Idm()

	if !sfs.canTestPerm && !vfs.HasFeature(avfs.FeatChownUser) {
		err := vfs.Lchown(testDir, 0, 0)

		CheckPathError(t, err).Op("lchown").Path(testDir).
			Err(avfs.ErrOpNotPermitted, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)

		return
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		if vfs.HasFeature(avfs.FeatRealFS) {
			return
		}

		wantUid, wantGid := 42, 42

		err := vfs.Lchown(testDir, wantUid, wantGid)
		CheckNoError(t, "Lchown "+testDir, err)

		fst, err := vfs.Stat(testDir)
		CheckNoError(t, "Stat "+testDir, err)

		sst := vfs.ToSysStat(fst)

		uid, gid := sst.Uid(), sst.Gid()
		if uid != wantUid || gid != wantGid {
			t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
				testDir, wantUid, wantGid, uid, gid)
		}

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)
	uis := UserInfos()

	t.Run("LchownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

			u, err := idm.LookupUser(wantName)
			if !CheckNoError(t, "LookupUser "+wantName, err) {
				return
			}

			for _, dir := range dirs {
				path := vfs.Join(testDir, dir.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+path, err)

				fst, err := vfs.Lstat(path)
				if !CheckNoError(t, "Lstat "+path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
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

			u, err := idm.LookupUser(wantName)
			if !CheckNoError(t, "LookupUser "+wantName, err) {
				return
			}

			for _, file := range files {
				path := vfs.Join(testDir, file.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+path, err)

				fst, err := vfs.Lstat(path)
				if !CheckNoError(t, "Lstat "+path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
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

			u, err := idm.LookupUser(wantName)
			if !CheckNoError(t, "LookupUser "+wantName, err) {
				return
			}

			for _, symlink := range symlinks {
				path := vfs.Join(testDir, symlink.NewName)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+path, err)

				fst, err := vfs.Lstat(path)
				if !CheckNoError(t, "Lstat "+path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Lchown(nonExistingFile, 0, 0)
		CheckPathError(t, err).Op("lchown").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("LchownPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Lchown(path, 0, 0)
		})
	})
}

// TestLink tests Link function.
func (sfs *SuiteFS) TestLink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatHardlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Link(testDir, testDir)

		CheckLinkError(t, err).Op("link").Old(testDir).New(testDir).
			Err(avfs.ErrPermDenied, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	pathLinks := sfs.ExistingDir(t, testDir)

	t.Run("LinkCreate", func(t *testing.T) {
		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(oldPath, newPath)
			CheckNoError(t, "Link "+oldPath+" "+newPath, err)

			newContent, err := vfs.ReadFile(newPath)
			CheckNoError(t, "ReadFile "+newPath, err)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkExisting", func(t *testing.T) {
		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(oldPath, newPath)
			CheckLinkError(t, err).Op("link").Old(oldPath).New(newPath).
				Err(avfs.ErrFileExists, avfs.OsLinux).
				Err(avfs.ErrWinCantCreateFile, avfs.OsWindows)
		}
	})

	t.Run("LinkRemove", func(t *testing.T) {
		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Remove(oldPath)
			CheckNoError(t, "Remove "+oldPath, err)

			newContent, err := vfs.ReadFile(newPath)
			CheckNoError(t, "ReadFile "+oldPath, err)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkErrorDir", func(t *testing.T) {
		for _, dir := range dirs {
			oldPath := vfs.Join(testDir, dir.Path)
			newPath := vfs.Join(testDir, defaultDir)

			err := vfs.Link(oldPath, newPath)
			CheckLinkError(t, err).Op("link").Old(oldPath).New(newPath).
				Err(avfs.ErrOpNotPermitted, avfs.OsLinux).
				Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
		}
	})

	t.Run("LinkErrorFile", func(t *testing.T) {
		for _, file := range files {
			InvalidPath := vfs.Join(testDir, file.Path, defaultNonExisting)
			NewInvalidPath := vfs.Join(pathLinks, defaultFile)

			err := vfs.Link(InvalidPath, NewInvalidPath)
			CheckLinkError(t, err).Op("link").Old(InvalidPath).New(NewInvalidPath).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}
	})

	t.Run("LinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)
		existingFile := sfs.EmptyFile(t, testDir)

		err := vfs.Link(nonExistingFile, existingFile)
		CheckLinkError(t, err).Op("link").Old(nonExistingFile).New(existingFile).Err(avfs.ErrNoSuchFileOrDir)
	})
}

// TestLstat tests Lstat function.
func (sfs *SuiteFS) TestLstat(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Lstat(testDir)
		CheckPathError(t, err).Op("lstat").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest

	t.Run("LstatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			info, err := vfs.Lstat(path)
			if !CheckNoError(t, "Lstat "+path, err) {
				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("LstatFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			info, err := vfs.Lstat(path)
			if !CheckNoError(t, "Lstat "+path, err) {
				continue
			}

			if info.Name() != vfs.Base(path) {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
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
		for _, sl := range GetSampleSymlinksEval(vfs) {
			newPath := vfs.Join(testDir, sl.NewName)
			oldPath := vfs.Join(testDir, sl.OldName)

			info, err := vfs.Lstat(newPath)
			if !CheckNoError(t, "Lstat", err) {
				continue
			}

			wantName := vfs.Base(oldPath)
			wantMode := sl.Mode

			if sl.IsSymlink {
				wantName = vfs.Base(newPath)
				wantMode = fs.ModeSymlink | fs.ModePerm
			}

			if wantName != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", newPath, wantName, info.Name())
			}

			if vfs.OSType() != avfs.OsWindows && wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", newPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("LStatNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Lstat(nonExistingFile)
		CheckPathError(t, err).Path(nonExistingFile).
			Op("lstat", avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Op("CreateFile", avfs.OsWindows).Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("LStatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(testDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Lstat(subDirOnFile)
		CheckPathError(t, err).Path(subDirOnFile).
			Op("lstat", avfs.OsLinux).Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Op("CreateFile", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestMkdir tests Mkdir function.
func (sfs *SuiteFS) TestMkdir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	ut := vfs.Utils()

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Mkdir(testDir, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	dirs := GetSampleDirs()

	t.Run("Mkdir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

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

			curPath := testDir

			pi := ut.NewPathIterator(dir.Path)
			for i := 0; pi.Next(); i++ {
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, pi.Part())
				info, err := vfs.Stat(curPath)
				if !CheckNoError(t, "Stat "+curPath, err) {
					return
				}

				if vfs.OSType() == avfs.OsWindows {
					continue
				}

				wantMode &^= vfs.UMask()

				mode := info.Mode() & fs.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirOnExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			err := vfs.Mkdir(path, dir.Mode)
			if !vfs.IsExist(err) {
				t.Errorf("mkdir %s : want IsExist(err) to be true, got error %v", path, err)
			}
		}
	})

	t.Run("MkdirOnNonExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path, "can't", "create", "this")

			err := vfs.Mkdir(path, avfs.DefaultDirPerm)
			CheckPathError(t, err).Op("mkdir").Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}
	})

	t.Run("MkdirEmptyName", func(t *testing.T) {
		err := vfs.Mkdir("", avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("mkdir").Path("").
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("MkdirOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.Mkdir(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(subDirOnFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("MkdirPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			newDir := vfs.Join(path, "newDir")

			return vfs.Mkdir(newDir, avfs.DefaultDirPerm)
		})
	})
}

// TestMkdirAll tests MkdirAll function.
func (sfs *SuiteFS) TestMkdirAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	ut := vfs.Utils()

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	existingFile := sfs.EmptyFile(t, testDir)
	dirs := GetSampleDirsAll()

	t.Run("MkdirAll", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			err := vfs.MkdirAll(path, dir.Mode)
			CheckNoError(t, "MkdirAll "+path, err)

			fi, err := vfs.Stat(path)
			CheckNoError(t, "Stat "+path, err)

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

			name := vfs.Base(vfs.FromSlash(dir.Path))
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", path, name, fi.Name())
			}

			want := strings.Count(vfs.FromSlash(dir.Path), string(vfs.PathSeparator()))
			got := len(dir.WantModes)
			if want != got {
				t.Fatalf("stat %s : want %d directories modes, got %d", path, want, got)
			}

			curPath := testDir

			pi := ut.NewPathIterator(dir.Path)
			for i := 0; pi.Next(); i++ {
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, pi.Part())
				info, err := vfs.Stat(curPath)
				if !CheckNoError(t, "Stat "+curPath, err) {
					return
				}

				wantMode &^= vfs.UMask()
				if vfs.OSType() == avfs.OsWindows {
					wantMode = fs.ModePerm
				}

				mode := info.Mode() & fs.ModePerm
				if wantMode != mode {
					t.Errorf("stat %s %s : want mode to be %s, got %s", path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirAllExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			err := vfs.MkdirAll(path, dir.Mode)
			CheckNoError(t, "MkdirAll "+path, err)
		}
	})

	t.Run("MkdirAllOnFile", func(t *testing.T) {
		err := vfs.MkdirAll(existingFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(existingFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("MkdirAllSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.MkdirAll(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(existingFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})

	t.Run("MkdirAllPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			newDir := vfs.Join(path, "newDir")

			return vfs.MkdirAll(newDir, avfs.DefaultDirPerm)
		})
	})
}

// TestMkdirTemp tests MkdirTemp function.
func (sfs *SuiteFS) TestMkdirTemp(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.MkdirTemp(testDir, "")
		if err.(*fs.PathError).Err != avfs.ErrPermDenied {
			t.Errorf("MkdirTemp : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	t.Run("MkdirTempEmptyDir", func(t *testing.T) {
		tmpDir, err := vfs.MkdirTemp("", "")
		CheckNoError(t, "MkdirTemp", err)

		wantDir := vfs.TempDir()
		dir := vfs.Dir(tmpDir)

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}
	})

	t.Run("MkdirTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.MkdirTemp("", badPattern)
		CheckPathError(t, err).Op("mkdirtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator)
	})

	t.Run("MkdirTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.MkdirTemp(existingFile, "")
		CheckPathError(t, err).Op("mkdir").
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestName tests Name function.
func (sfs *SuiteFS) TestName(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if vfs.Name() != "" {
		t.Errorf("want name to be empty, got %s", vfs.Name())
	}
}

// TestOpen tests Open function.
func (sfs *SuiteFS) TestOpen(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Open(testDir)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	existingFile := sfs.ExistingFile(t, testDir, data)
	existingDir := sfs.ExistingDir(t, testDir)

	vfs = sfs.vfsTest

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		CheckNoError(t, "Open", err)

		defer f.Close()

		gotData, err := io.ReadAll(f)
		CheckNoError(t, "ReadAll", err)

		if !bytes.Equal(gotData, data) {
			t.Errorf("ReadAll : want error data to be %v, got %v", data, gotData)
		}
	})

	t.Run("OpenFileDirReadOnly", func(t *testing.T) {
		f, err := vfs.Open(existingDir)
		CheckNoError(t, "Open", err)

		defer f.Close()

		dirs, err := f.ReadDir(-1)
		CheckNoError(t, "ReadDir", err)

		if len(dirs) != 0 {
			t.Errorf("ReadDir : want number of directories to be 0, got %d", len(dirs))
		}
	})

	t.Run("OpenNonExistingFile", func(t *testing.T) {
		fileName := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Open(fileName)
		CheckPathError(t, err).Op("open").Path(fileName).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})
}

// TestOpenFileWrite tests OpenFile function for write.
func (sfs *SuiteFS) TestOpenFileWrite(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.OpenFile(testDir, os.O_WRONLY, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	defaultData := []byte("Default data")
	buf3 := make([]byte, 3)

	t.Run("OpenFileWriteOnly", func(t *testing.T) {
		existingFile := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.OpenFile(existingFile, os.O_WRONLY, avfs.DefaultFilePerm)
		CheckNoError(t, "OpenFile", err)

		defer f.Close()

		n, err := f.Write(defaultData)
		CheckNoError(t, "Write", err)

		if n != len(defaultData) {
			t.Errorf("Write : want bytes written to be %d, got %d", len(defaultData), n)
		}

		n, err = f.Read(buf3)
		CheckPathError(t, err).Op("read").Path(existingFile).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("Read : want bytes written to be 0, got %d", n)
		}

		n, err = f.ReadAt(buf3, 3)
		CheckPathError(t, err).Op("read").Path(existingFile).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		CheckNoError(t, "Chmod", err)

		if vfs.HasFeature(avfs.FeatIdentityMgr) {
			u := vfs.User()
			err = f.Chown(u.Uid(), u.Gid())
			CheckNoError(t, "Chown", err)
		}

		fst, err := f.Stat()
		CheckNoError(t, "Stat", err)

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		CheckNoError(t, "Truncate", err)

		err = f.Sync()
		CheckNoError(t, "Sync", err)
	})

	t.Run("OpenFileAppend", func(t *testing.T) {
		existingFile := sfs.ExistingFile(t, testDir, data)
		appendData := []byte("appendData")

		f, err := vfs.OpenFile(existingFile, os.O_WRONLY|os.O_APPEND, avfs.DefaultFilePerm)
		CheckNoError(t, "OpenFile "+existingFile, err)

		n, err := f.Write(appendData)
		CheckNoError(t, "Write", err)

		if n != len(appendData) {
			t.Errorf("Write : want error to be %d, got %d", len(defaultData), n)
		}

		_ = f.Close()

		gotContent, err := vfs.ReadFile(existingFile)
		CheckNoError(t, "ReadFile", err)

		wantContent := append(data, appendData...)
		if !bytes.Equal(wantContent, gotContent) {
			t.Errorf("ReadAll : want content to be %s, got %s", wantContent, gotContent)
		}
	})

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		existingFile := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.Open(existingFile)
		CheckNoError(t, "Open", err)

		defer f.Close()

		n, err := f.Write(defaultData)
		CheckPathError(t, err).Op("write").Path(existingFile).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.WriteAt(defaultData, 3)
		CheckPathError(t, err).Op("write").Path(existingFile).
			Err(avfs.ErrBadFileDesc, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		CheckNoError(t, "Chmod", err)

		if vfs.HasFeature(avfs.FeatIdentityMgr) {
			u := vfs.User()
			err = f.Chown(u.Uid(), u.Gid())
			CheckNoError(t, "Chown", err)
		}

		fst, err := f.Stat()
		CheckNoError(t, "Stat", err)

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(existingFile).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})

	t.Run("OpenFileWriteOnDir", func(t *testing.T) {
		existingDir := sfs.ExistingDir(t, testDir)

		f, err := vfs.OpenFile(existingDir, os.O_WRONLY, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(existingDir).
			Err(avfs.ErrIsADirectory, avfs.OsLinux).
			Err(avfs.ErrWinIsADirectory, avfs.OsWindows)

		if !reflect.ValueOf(f).IsNil() {
			t.Errorf("OpenFile : want file to be nil, got %v", f)
		}
	})

	t.Run("OpenFileExcl", func(t *testing.T) {
		fileExcl := vfs.Join(testDir, "fileExcl")

		f, err := vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		CheckNoError(t, "OpenFile", err)

		f.Close()

		_, err = vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(fileExcl).
			Err(avfs.ErrFileExists, avfs.OsLinux).
			Err(avfs.ErrWinFileExists, avfs.OsWindows)
	})

	t.Run("OpenFileNonExistingPath", func(t *testing.T) {
		nonExistingPath := vfs.Join(testDir, "non/existing/path")

		_, err := vfs.OpenFile(nonExistingPath, os.O_CREATE, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(nonExistingPath).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestReadDir tests ReadDir function.
func (sfs *SuiteFS) TestReadDir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.ReadDir(testDir)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	rndTree := sfs.RandomDir(t, testDir)
	wDirs := len(rndTree.Dirs)
	wFiles := len(rndTree.Files)
	wSymlinks := len(rndTree.SymLinks)

	existingFile := rndTree.Files[0].Name

	t.Run("ReadDirAll", func(t *testing.T) {
		dirEntries, err := vfs.ReadDir(testDir)
		if !CheckNoError(t, "ReadDir", err) {
			return
		}

		var gDirs, gFiles, gSymlinks int
		for _, dirEntry := range dirEntries {
			_, err = dirEntry.Info()
			if !CheckNoError(t, "Info", err) {
				continue
			}

			switch {
			case dirEntry.IsDir():
				gDirs++
			case dirEntry.Type()&fs.ModeSymlink != 0:
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
			if !CheckNoError(t, "ReadDir", err) {
				continue
			}

			l := len(dirInfos)
			if l != 0 {
				t.Errorf("ReadDir %s : want count to be O, got %d", dir, l)
			}
		}
	})

	t.Run("ReadDirExistingFile", func(t *testing.T) {
		_, err := vfs.ReadDir(existingFile)
		CheckPathError(t, err).Path(existingFile).
			Op("readdirent", avfs.OsLinux).Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Op("readdir", avfs.OsWindows).Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestReadFile tests ReadFile function.
func (sfs *SuiteFS) TestReadFile(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.ReadFile(testDir)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")
	path := vfs.Join(testDir, "TestReadFile.txt")

	t.Run("ReadFile", func(t *testing.T) {
		rb, err := vfs.ReadFile(path)
		if err == nil {
			t.Errorf("ReadFile : want error to be %v, got nil", avfs.ErrNoSuchFileOrDir)
		}

		if len(rb) != 0 {
			t.Errorf("ReadFile : want read bytes to be 0, got %d", len(rb))
		}

		vfs = sfs.vfsSetup

		path = sfs.ExistingFile(t, testDir, data)

		vfs = sfs.vfsTest

		rb, err = vfs.ReadFile(path)
		CheckNoError(t, "ReadFile", err)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// TestReadlink tests Readlink function.
func (sfs *SuiteFS) TestReadlink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.Readlink(testDir)

		CheckPathError(t, err).Op("readlink").Path(testDir).
			Err(avfs.ErrPermDenied, avfs.OsLinux).
			Err(avfs.ErrWinNotReparsePoint, avfs.OsWindows)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest

	t.Run("ReadlinkLink", func(t *testing.T) {
		for _, sl := range symlinks {
			oldPath := vfs.Join(testDir, sl.OldName)
			newPath := vfs.Join(testDir, sl.NewName)

			gotPath, err := vfs.Readlink(newPath)
			CheckNoError(t, "Readlink", err)

			if oldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", newPath, oldPath, gotPath)
			}
		}
	})

	t.Run("ReadlinkDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, err).Op("readlink").Path(path).
				Err(avfs.ErrInvalidArgument, avfs.OsLinux).
				Err(avfs.ErrWinNotReparsePoint, avfs.OsWindows)
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, err).Op("readlink").Path(path).
				Err(avfs.ErrInvalidArgument, avfs.OsLinux).
				Err(avfs.ErrWinNotReparsePoint, avfs.OsWindows)
		}
	})

	t.Run("ReadLinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Readlink(nonExistingFile)
		CheckPathError(t, err).Op("readlink").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})
}

// TestRemove tests Remove function.
func (sfs *SuiteFS) TestRemove(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Remove(testDir)
		CheckPathError(t, err).Op("remove").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)

	t.Run("RemoveFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			_, err := vfs.Stat(path)
			if !CheckNoError(t, "Stat", err) {
				continue
			}

			err = vfs.Remove(path)
			CheckNoError(t, "Remove", err)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat().Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			dirInfos, err := vfs.ReadDir(path)
			if !CheckNoError(t, "ReadDir "+path, err) {
				continue
			}

			err = vfs.Remove(path)

			isLeaf := len(dirInfos) == 0
			if isLeaf {
				CheckNoError(t, "Remove "+path, err)

				_, err = vfs.Stat(path)
				CheckPathError(t, err).OpStat().Path(path).
					Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
					Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

				continue
			}

			CheckPathError(t, err).Op("remove").Path(path).
				Err(avfs.ErrDirNotEmpty, avfs.OsLinux).
				Err(avfs.ErrWinDirNotEmpty, avfs.OsWindows)

			_, err = vfs.Stat(path)
			CheckNoError(t, "Stat "+path, err)
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			newPath := vfs.Join(testDir, sl.NewName)

			err := vfs.Remove(newPath)
			CheckNoError(t, "Remove "+newPath, err)

			_, err = vfs.Stat(newPath)
			CheckPathError(t, err).OpStat().Path(newPath).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
		}
	})

	t.Run("RemoveNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Remove(nonExistingFile)
		CheckPathError(t, err).Op("remove").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("RemovePerm", func(t *testing.T) {
		t.Skip("TODO: Remove")

		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.Remove(path)
		})
	})
}

// TestRemoveAll tests RemoveAll function.
func (sfs *SuiteFS) TestRemoveAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.RemoveAll(testDir)
		CheckPathError(t, err).Op("removeall").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	baseDir := vfs.Join(testDir, "RemoveAll")
	dirs := sfs.SampleDirs(t, baseDir)
	files := sfs.SampleFiles(t, baseDir)
	symlinks := sfs.SampleSymlinks(t, baseDir)

	t.Run("RemoveAll", func(t *testing.T) {
		err := vfs.RemoveAll(baseDir)
		if !CheckNoError(t, "RemoveAll "+baseDir, err) {
			return
		}

		for _, dir := range dirs {
			path := vfs.Join(baseDir, dir.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat().Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}

		for _, file := range files {
			path := vfs.Join(baseDir, file.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat().Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}

		for _, sl := range symlinks {
			path := vfs.Join(baseDir, sl.NewName)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat().Path(path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}

		_, err = vfs.Stat(baseDir)
		CheckPathError(t, err).OpStat().Path(baseDir).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("RemoveAllOneFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		err := vfs.RemoveAll(existingFile)
		CheckNoError(t, "RemoveAll "+existingFile, err)
	})

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		_ = sfs.SampleDirs(t, baseDir)

		err := vfs.Chdir(baseDir)
		if !CheckNoError(t, "Chdir "+baseDir, err) {
			return
		}

		err = vfs.RemoveAll("")
		CheckNoError(t, "RemoveAll ''", err)

		// Verify that nothing was removed.
		for _, dir := range dirs {
			path := vfs.Join(baseDir, dir.Path)

			_, err = vfs.Stat(path)
			CheckNoError(t, "Stat", err)
		}
	})

	t.Run("RemoveAllNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.RemoveAll(nonExistingFile)
		CheckNoError(t, "RemoveAll "+nonExistingFile, err)
	})

	t.Run("RemoveAllPerm", func(t *testing.T) {
		t.Skip("TODO: RemoveAll")

		if !sfs.canTestPerm {
			return
		}

		pt := NewPermTest(sfs, testDir)
		pt.Test(t, func(path string) error {
			return vfs.RemoveAll(path)
		})
	})
}

// TestRename tests Rename function.
func (sfs *SuiteFS) TestRename(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Rename(testDir, testDir)
		CheckLinkError(t, err).Op("rename").Old(testDir).New(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("data")

	t.Run("RenameDir", func(t *testing.T) {
		dirs := sfs.SampleDirs(t, testDir)

		for i := len(dirs) - 1; i >= 0; i-- {
			oldPath := vfs.Join(testDir, dirs[i].Path)
			newPath := oldPath + "New"

			err := vfs.Rename(oldPath, newPath)
			CheckNoError(t, "Rename", err)

			_, err = vfs.Stat(oldPath)
			CheckPathError(t, err).OpStat().Path(oldPath).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

			_, err = vfs.Stat(newPath)
			CheckNoError(t, "Stat "+newPath, err)
		}
	})

	t.Run("RenameFile", func(t *testing.T) {
		_ = sfs.SampleDirs(t, testDir)
		files := sfs.SampleFiles(t, testDir)

		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(testDir, vfs.Base(oldPath))

			err := vfs.Rename(oldPath, newPath)
			CheckNoError(t, "Rename "+oldPath+" "+newPath, err)

			_, err = vfs.Stat(oldPath)

			switch {
			case oldPath == newPath:
				CheckNoError(t, "Stat "+oldPath, err)
			default:
				CheckPathError(t, err).OpStat().Path(oldPath).
					Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
					Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
			}

			_, err = vfs.Stat(newPath)
			CheckNoError(t, "Stat "+newPath, err)
		}
	})

	t.Run("RenameNonExistingFile", func(t *testing.T) {
		srcNonExistingFile := vfs.Join(testDir, "srcNonExistingFile1")
		dstNonExistingFile := vfs.Join(testDir, "dstNonExistingFile2")

		err := vfs.Rename(srcNonExistingFile, dstNonExistingFile)
		CheckLinkError(t, err).Op("rename").Old(srcNonExistingFile).New(dstNonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("RenameDirToExistingDir", func(t *testing.T) {
		srcExistingDir := sfs.ExistingDir(t, testDir)
		dstExistingDir := sfs.ExistingDir(t, testDir)

		err := vfs.Rename(srcExistingDir, dstExistingDir)

		CheckLinkError(t, err).Op("rename").Old(srcExistingDir).New(dstExistingDir).
			Err(avfs.ErrFileExists, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})

	t.Run("RenameFileToExistingFile", func(t *testing.T) {
		srcExistingFile := sfs.ExistingFile(t, testDir, data)
		dstExistingFile := sfs.EmptyFile(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingFile)
		CheckNoError(t, "Rename "+srcExistingFile+" "+dstExistingFile, err)

		_, err = vfs.Stat(srcExistingFile)
		CheckPathError(t, err).OpStat().Path(srcExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

		info, err := vfs.Stat(dstExistingFile)
		CheckNoError(t, "Stat "+dstExistingFile, err)

		if int(info.Size()) != len(data) {
			t.Errorf("Stat : want size to be %d, got %d", len(data), info.Size())
		}
	})

	t.Run("RenameFileToExistingDir", func(t *testing.T) {
		srcExistingFile := sfs.ExistingFile(t, testDir, data)
		dstExistingDir := sfs.ExistingDir(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingDir)
		CheckLinkError(t, err).Op("rename").Old(srcExistingFile).New(dstExistingDir).
			Err(avfs.ErrFileExists, avfs.OsLinux).
			Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
	})
}

// TestSameFile tests SameFile function.
func (sfs *SuiteFS) TestSameFile(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		if vfs.SameFile(nil, nil) {
			t.Errorf("SameFile : want SameFile to be false, got true")
		}

		return
	}

	testDir1 := vfs.Join(testDir, "dir1")
	testDir2 := vfs.Join(testDir, "dir2")

	sfs.SampleDirs(t, testDir1)
	files := sfs.SampleFiles(t, testDir1)

	sfs.SampleDirs(t, testDir2)

	t.Run("SameFileLink", func(t *testing.T) {
		if !vfs.HasFeature(avfs.FeatHardlink) {
			return
		}

		for _, file := range files {
			path1 := vfs.Join(testDir1, file.Path)
			path2 := vfs.Join(testDir2, file.Path)

			info1, err := vfs.Stat(path1)
			if !CheckNoError(t, "Stat "+path1, err) {
				continue
			}

			err = vfs.Link(path1, path2)
			CheckNoError(t, "Link "+path1+" "+path2, err)

			info2, err := vfs.Stat(path2)
			if !CheckNoError(t, "Stat "+path2, err) {
				continue
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			err = vfs.Remove(path2)
			CheckNoError(t, "Remove "+path2, err)
		}
	})

	t.Run("SameFileSymlink", func(t *testing.T) {
		if !vfs.HasFeature(avfs.FeatSymlink) {
			return
		}

		for _, file := range files {
			path1 := vfs.Join(testDir1, file.Path)
			path2 := vfs.Join(testDir2, file.Path)

			info1, err := vfs.Stat(path1)
			if !CheckNoError(t, "Stat "+path1, err) {
				continue
			}

			err = vfs.Symlink(path1, path2)
			if !CheckNoError(t, "Symlink "+path1+" "+path2, err) {
				continue
			}

			info2, err := vfs.Stat(path2)
			if !CheckNoError(t, "Stat "+path2, err) {
				continue
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			info3, err := vfs.Lstat(path2)
			if !CheckNoError(t, "Lstat "+path2, err) {
				continue
			}

			if vfs.SameFile(info1, info3) {
				t.Fatalf("SameFile %s, %s : not the same file\n%v\n%v", path1, path2, info1, info3)
			}

			err = vfs.Remove(path2)
			CheckNoError(t, "Remove "+path2, err)
		}
	})
}

// TestStat tests Stat function.
func (sfs *SuiteFS) TestStat(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Stat(testDir)
		CheckPathError(t, err).OpStat().Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			info, err := vfs.Stat(path)
			if !CheckNoError(t, "Stat "+path, err) {
				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", path, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			info, err := vfs.Stat(path)
			if !CheckNoError(t, "Stat "+path, err) {
				continue
			}

			if info.Name() != vfs.Base(path) {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
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
		for _, sl := range GetSampleSymlinksEval(vfs) {
			newPath := vfs.Join(testDir, sl.NewName)
			oldPath := vfs.Join(testDir, sl.OldName)

			info, err := vfs.Stat(newPath)
			if !CheckNoError(t, "Stat "+newPath, err) {
				continue
			}

			wantName := vfs.Base(oldPath)
			if sl.IsSymlink {
				wantName = vfs.Base(newPath)
			}

			wantMode := sl.Mode
			if wantName != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", newPath, wantName, info.Name())
			}

			if sfs.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", newPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Stat(nonExistingFile)
		CheckPathError(t, err).OpStat().Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("StatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(testDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Stat(subDirOnFile)
		CheckPathError(t, err).OpStat().Path(subDirOnFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestSymlink tests Symlink function.
func (sfs *SuiteFS) TestSymlink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Symlink(testDir, testDir)
		CheckLinkError(t, err).Op("symlink").Old(testDir).New(testDir).
			Err(avfs.ErrPermDenied, avfs.OsLinux).
			Err(avfs.ErrWinPrivilegeNotHeld, avfs.OsWindows)

		return
	}

	_ = sfs.SampleDirs(t, testDir)
	_ = sfs.SampleFiles(t, testDir)

	t.Run("Symlink", func(t *testing.T) {
		symlinks := GetSampleSymlinks(vfs)
		for _, sl := range symlinks {
			oldPath := vfs.Join(testDir, sl.OldName)
			newPath := vfs.Join(testDir, sl.NewName)

			err := vfs.Symlink(oldPath, newPath)
			CheckNoError(t, "Symlink "+oldPath+" "+newPath, err)

			gotPath, err := vfs.Readlink(newPath)
			CheckNoError(t, "Readlink "+newPath, err)

			if oldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", newPath, oldPath, gotPath)
			}
		}
	})
}

// TestToSysStat tests ToSysStat function.
func (sfs *SuiteFS) TestToSysStat(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		sst := vfs.ToSysStat(nil)

		if _, ok := sst.(*avfs.DummySysStat); !ok {
			t.Errorf("ToSysStat : want result of type DummySysStat, got %s", reflect.TypeOf(sst).Name())
		}

		return
	}

	existingFile := sfs.EmptyFile(t, testDir)

	fst, err := vfs.Stat(existingFile)
	if err != nil {
		t.Errorf("Stat : want error be nil, got %v", err)
	}

	u := vfs.User()
	if vfs.HasFeature(avfs.FeatReadOnly) {
		u = sfs.vfsSetup.User()
	}

	wantUid, wantGid := u.Uid(), u.Gid()

	sst := vfs.ToSysStat(fst)

	uid, gid := sst.Uid(), sst.Gid()
	if uid != wantUid || gid != wantGid {
		t.Errorf("ToSysStat : want Uid = %d, Gid = %d, got Uid = %d, Gid = %d",
			wantUid, wantGid, uid, gid)
	}

	wantLink := uint64(1)
	if sst.Nlink() != wantLink {
		t.Errorf("ToSysStat : want Nlink to be %d, got %d", wantLink, sst.Nlink())
	}
}

// TestTruncate tests Truncate function.
func (sfs *SuiteFS) TestTruncate(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Truncate(testDir, 0)
		CheckPathError(t, err).Op("truncate").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("Truncate", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

		for i := len(data); i >= 0; i-- {
			err := vfs.Truncate(path, int64(i))
			CheckNoError(t, "Truncate "+path, err)

			d, err := vfs.ReadFile(path)
			CheckNoError(t, "ReadFile "+path, err)

			if len(d) != i {
				t.Errorf("Truncate : want length to be %d, got %d", i, len(d))
			}
		}
	})

	t.Run("TruncateNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Truncate(nonExistingFile, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckNoError(t, "Truncate "+nonExistingFile, err)
		default:
			CheckPathError(t, err).Op("truncate").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
		}
	})

	t.Run("TruncateOnDir", func(t *testing.T) {
		err := vfs.Truncate(testDir, 0)
		CheckPathError(t, err).Path(testDir).
			Op("truncate", avfs.OsLinux).Err(avfs.ErrIsADirectory, avfs.OsLinux).
			Op("open", avfs.OsWindows).Err(avfs.ErrWinIsADirectory, avfs.OsWindows)
	})

	t.Run("TruncateSizeNegative", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

		err := vfs.Truncate(path, -1)
		CheckPathError(t, err).Op("truncate").Path(path).
			Err(avfs.ErrInvalidArgument, avfs.OsLinux).
			Err(avfs.ErrWinNegativeSeek, avfs.OsWindows)
	})

	t.Run("TruncateSizeBiggerFileSize", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)
		newSize := len(data) * 2

		err := vfs.Truncate(path, int64(newSize))
		CheckNoError(t, "Truncate "+path, err)

		info, err := vfs.Stat(path)
		CheckNoError(t, "Stat "+path, err)

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		gotContent, err := vfs.ReadFile(path)
		CheckNoError(t, "ReadFile "+path, err)

		wantAdded := bytes.Repeat([]byte{0}, len(data))
		gotAdded := gotContent[len(data):]
		if !bytes.Equal(wantAdded, gotAdded) {
			t.Errorf("Bytes Added : want %v, got %v", wantAdded, gotAdded)
		}
	})
}

// TestUmask tests SetUMask and UMask functions.
func (sfs *SuiteFS) TestUmask(t *testing.T, testDir string) {
	const umaskTest = 0o077

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		vfs.SetUMask(0)

		if um := vfs.UMask(); um <= 0 {
			t.Errorf("UMask : want umask to be > 0, got %d", um)
		}

		return
	}

	umaskStart := vfs.UMask()
	vfs.SetUMask(umaskTest)

	u := vfs.UMask()
	if u != umaskTest {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskTest, u)
	}

	vfs.SetUMask(umaskStart)

	u = vfs.UMask()
	if u != umaskStart {
		t.Errorf("umaskTest : want umask to be %o, got %o", umaskStart, u)
	}
}

// TestSetUser tests SetUser and User functions.
func (sfs *SuiteFS) TestSetUser(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	idm := vfs.Idm()

	suffix := "User" + vfs.Type()

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		userName := "InvalidUser" + suffix

		_, err := vfs.SetUser(userName)
		if err != avfs.ErrPermDenied {
			t.Errorf("SetUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	defer vfs.SetUser(idm.AdminUser().Name()) //nolint:errcheck // Ignore errors.

	CreateGroups(t, idm, suffix)
	CreateUsers(t, idm, suffix)

	t.Run("UserNotExists", func(t *testing.T) {
		const userName = "notExistingUser"

		wantErr := avfs.UnknownUserError(userName)

		_, err := idm.LookupUser(userName)
		if err != wantErr {
			t.Fatalf("LookupUser %s : want error to be %v, got %v", userName, wantErr, err)
		}

		_, err = vfs.SetUser(userName)
		if err != wantErr {
			t.Errorf("SetUser %s : want error to be %v, got %v", userName, wantErr, err)
		}
	})

	t.Run("UserExists", func(t *testing.T) {
		for _, ui := range UserInfos() {
			userName := ui.Name + suffix

			lu, err := idm.LookupUser(userName)
			if !CheckNoError(t, "LookupUser "+userName, err) {
				continue
			}

			uid := lu.Uid()
			gid := lu.Gid()

			// loop to test change with the same user
			for i := 0; i < 2; i++ {
				u, err := vfs.SetUser(userName)
				if !CheckNoError(t, "SetUser "+userName, err) {
					continue
				}

				if u.Name() != userName {
					t.Errorf("SetUser %s : want name to be %s, got %s", userName, userName, u.Name())
				}

				if u.Uid() != uid {
					t.Errorf("SetUser %s : want uid to be %d, got %d", userName, uid, u.Uid())
				}

				if u.Gid() != gid {
					t.Errorf("SetUser %s : want gid to be %d, got %d", userName, gid, u.Gid())
				}

				cu := vfs.User()
				if cu.Name() != userName {
					t.Errorf("SetUser %s : want name to be %s, got %s", userName, userName, cu.Name())
				}

				if cu.Uid() != uid {
					t.Errorf("SetUser %s : want uid to be %d, got %d", userName, uid, cu.Uid())
				}

				if cu.Gid() != gid {
					t.Errorf("SetUser %s : want gid to be %d, got %d", userName, gid, cu.Gid())
				}
			}
		}
	})
}

// TestWriteFile tests WriteFile function.
func (sfs *SuiteFS) TestWriteFile(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.WriteFile(testDir, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("WriteFile", func(t *testing.T) {
		path := vfs.Join(testDir, "WriteFile.txt")

		err := vfs.WriteFile(path, data, avfs.DefaultFilePerm)
		CheckNoError(t, "WriteFile "+path, err)

		rb, err := vfs.ReadFile(path)
		CheckNoError(t, "ReadFile "+path, err)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// TestWriteOnReadOnly tests all write functions of a read only file system.
func (sfs *SuiteFS) TestWriteOnReadOnly(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	existingFile := sfs.EmptyFile(t, testDir)

	if !vfs.HasFeature(avfs.FeatReadOnly) {
		t.Errorf("HasFeature : want read only file system")
	}

	t.Run("ReadOnlyFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if !CheckNoError(t, "Open "+existingFile, err) {
			return
		}

		err = f.Chmod(0o777)
		CheckPathError(t, err).Op("chmod").Path(f.Name()).Err(avfs.ErrPermDenied)

		err = f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(f.Name()).Err(avfs.ErrPermDenied)

		err = f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(f.Name()).Err(avfs.ErrPermDenied)

		_, err = f.Write([]byte{})
		CheckPathError(t, err).Op("write").Path(f.Name()).Err(avfs.ErrPermDenied)

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, err).Op("write").Path(f.Name()).Err(avfs.ErrPermDenied)

		_, err = f.WriteString("")
		CheckPathError(t, err).Op("write").Path(f.Name()).Err(avfs.ErrPermDenied)
	})
}

// TestWriteString tests WriteString function.
func (sfs *SuiteFS) TestWriteString(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := vfs.Join(testDir, "TestWriteString.txt")

	t.Run("WriteString", func(t *testing.T) {
		f, err := vfs.Create(path)
		CheckNoError(t, "Create "+path, err)

		n, err := f.WriteString(string(data))
		CheckNoError(t, "WriteString "+path, err)

		if len(data) != n {
			t.Errorf("WriteString : want written bytes to be %d, got %d", len(data), n)
		}

		f.Close()

		rb, err := vfs.ReadFile(path)
		CheckNoError(t, "ReadFile "+path, err)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}
