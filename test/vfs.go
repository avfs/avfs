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
			err := vfs.Chdir(testDir)
			if err != nil {
				t.Fatalf("Chdir %s : want error to be nil, got %v", testDir, err)
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
			if err != nil {
				t.Errorf("Chdir : want error to be nil, got %v", err)
			}

			err = vfs.Chdir(path)
			CheckPathError(t, err).Op("chdir").Path(path).Err(avfs.ErrNoSuchFileOrDir)

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
			CheckPathError(t, err).Op("chdir").Path(existingFile).Err(avfs.ErrWinDirNameInvalid)
		default:
			CheckPathError(t, err).Op("chdir").Path(existingFile).Err(avfs.ErrNotADirectory)
		}
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
				if err != nil {
					t.Fatalf("MkdirTemp %s : want error to be nil, got %v", testDir, err)
				}

				err = vfs.Chmod(path, wantMode)
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)
				}

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
				if err != nil {
					t.Errorf("Chmod %s : want error to be nil, got %v", existingFile, err)
				}

				fst, err := vfs.Stat(existingFile)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", existingFile, err)
				}

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
		CheckPathError(t, err).Op("chmod").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
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

	if !sfs.canTestPerm && !vfs.HasFeature(avfs.FeatChownUser) {
		err := vfs.Chown(testDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("chown").Path(testDir).Err(avfs.ErrWinNotSupported)
		default:
			CheckPathError(t, err).Op("chown").Path(testDir).Err(avfs.ErrOpNotPermitted)
		}

		return
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		if vfs.HasFeature(avfs.FeatRealFS) {
			return
		}

		wantUid, wantGid := 42, 42

		err := vfs.Chown(testDir, wantUid, wantGid)
		if err != nil {
			t.Errorf("Chown %s : want error to be nil, got %v", testDir, err)
		}

		fst, err := vfs.Stat(testDir)
		if err != nil {
			t.Errorf("Stat %s : want error to be nil, got %v", testDir, err)
		}

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

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := vfs.Join(testDir, dir.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)

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

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := vfs.Join(testDir, file.Path)

				err := vfs.Chown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Chown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Stat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)

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

		// Some file systems (MemFs) don't allow exit from a chroot.
		// A shallow clone of the file system is then used to perform the chroot
		// without loosing access to the original root of the file system.
		vfsChroot := vfs

		vfsClone, cloned := vfs.(avfs.Cloner)
		if cloned {
			vfsChroot = vfsClone.Clone()
		}

		err = vfsChroot.Chroot(chrootDir)
		if err != nil {
			t.Errorf("Chroot : want error to be nil, got %v", err)
		}

		_, err = vfsChroot.Stat(chrootFile)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		// Restore the original file system root if possible.
		if !cloned {
			err = fSave.Chdir()
			if err != nil {
				t.Errorf("Chdir : want error to be nil, got %v", err)
			}

			err = vfs.Chroot(".")
			if err != nil {
				t.Errorf("Chroot : want error to be nil, got %v", err)
			}
		}

		_, err = vfs.Stat(chrootFilePath)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}
	})

	t.Run("ChrootOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		nonExistingFile := vfs.Join(existingFile, "invalid", "path")

		err := vfs.Chroot(existingFile)
		CheckPathError(t, err).Op("chroot").Path(existingFile).Err(avfs.ErrNotADirectory)

		err = vfs.Chroot(nonExistingFile)
		CheckPathError(t, err).Op("chroot").Path(nonExistingFile).Err(avfs.ErrNotADirectory)
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
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Chtimes(nonExistingFile, time.Now(), time.Now())
		CheckPathError(t, err).Op("chtimes").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
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
		if err != nil {
			t.Errorf("want error to be nil, got %v", err)
		}

		wantDir := vfs.TempDir()
		dir := vfs.Dir(f.Name())

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}
	})

	t.Run("CreateTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.CreateTemp("", badPattern)
		CheckPathError(t, err).Op("createtemp").Path(badPattern).ErrAsString(avfs.ErrPatternHasSeparator)
	})

	t.Run("CreateTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.CreateTemp(existingFile, "")
		CheckPathError(t, err).Op("open").Err(avfs.ErrNotADirectory)
	})
}

// TestEvalSymlink tests EvalSymlink function.
func (sfs *SuiteFS) TestEvalSymlink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) || vfs.OSType() == avfs.OsWindows {
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
			wantOp := "lstat"
			wantPath := vfs.Join(testDir, sl.OldName)
			slPath := vfs.Join(testDir, sl.NewName)

			gotPath, err := vfs.EvalSymlinks(slPath)
			if sl.WantErr == nil && err == nil {
				if wantPath != gotPath {
					t.Errorf("EvalSymlinks %s : want Path to be %s, got %s", slPath, wantPath, gotPath)
				}

				continue
			}

			e, ok := err.(*fs.PathError)
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

	if !sfs.canTestPerm && !vfs.HasFeature(avfs.FeatChownUser) {
		err := vfs.Lchown(testDir, 0, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("lchown").Path(testDir).Err(avfs.ErrWinNotSupported)
		default:
			CheckPathError(t, err).Op("lchown").Path(testDir).Err(avfs.ErrOpNotPermitted)
		}

		return
	}

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		if vfs.HasFeature(avfs.FeatRealFS) {
			return
		}

		wantUid, wantGid := 42, 42

		err := vfs.Lchown(testDir, wantUid, wantGid)
		if err != nil {
			t.Errorf("Chown %s : want error to be nil, got %v", testDir, err)
		}

		fst, err := vfs.Stat(testDir)
		if err != nil {
			t.Errorf("Stat %s : want error to be nil, got %v", testDir, err)
		}

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

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, dir := range dirs {
				path := vfs.Join(testDir, dir.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)

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

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, file := range files {
				path := vfs.Join(testDir, file.Path)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)

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

			u, err := vfs.LookupUser(wantName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", wantName, err)
			}

			for _, symlink := range symlinks {
				path := vfs.Join(testDir, symlink.NewName)

				err := vfs.Lchown(path, u.Uid(), u.Gid())
				if err != nil {
					t.Errorf("Lchown %s : want error to be nil, got %v", path, err)
				}

				fst, err := vfs.Lstat(path)
				if err != nil {
					t.Errorf("Stat %s : want error to be nil, got %v", path, err)

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

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckLinkError(t, err).Op("link").Old(testDir).New(testDir).Err(avfs.ErrWinAccessDenied)
		default:
			CheckLinkError(t, err).Op("link").Old(testDir).New(testDir).Err(avfs.ErrPermDenied)
		}

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
			if err != nil {
				t.Errorf("Link %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			newContent, err := vfs.ReadFile(newPath)
			if err != nil {
				t.Errorf("Readfile %s : want error to be nil, got %v", newPath, err)
			}

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
			CheckLinkError(t, err).Op("link").Old(oldPath).New(newPath).Err(avfs.ErrFileExists)
		}
	})

	t.Run("LinkRemove", func(t *testing.T) {
		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Remove(oldPath)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", oldPath, err)
			}

			newContent, err := vfs.ReadFile(newPath)
			if err != nil {
				t.Errorf("Readfile %s : want error to be nil, got %v", newPath, err)
			}

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
			CheckLinkError(t, err).Op("link").Old(oldPath).New(newPath).Err(avfs.ErrOpNotPermitted)
		}
	})

	t.Run("LinkErrorFile", func(t *testing.T) {
		for _, file := range files {
			InvalidPath := vfs.Join(testDir, file.Path, defaultNonExisting)
			NewInvalidPath := vfs.Join(pathLinks, defaultFile)

			err := vfs.Link(InvalidPath, NewInvalidPath)
			CheckLinkError(t, err).Op("link").Old(InvalidPath).New(NewInvalidPath).Err(avfs.ErrNoSuchFileOrDir)
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
			if err != nil {
				t.Errorf("Lstat %s : want error to be nil, got %v", path, err)

				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.GetUMask()
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
		for _, sl := range GetSampleSymlinksEval(vfs) {
			newPath := vfs.Join(testDir, sl.NewName)
			oldPath := vfs.Join(testDir, sl.OldName)

			info, err := vfs.Lstat(newPath)
			if err != nil {
				if sl.WantErr == nil {
					t.Errorf("Lstat %s : want error to be nil, got %v", newPath, err)
				}

				CheckPathError(t, err).OpStat(vfs).Path(newPath).Err(sl.WantErr)

				continue
			}

			var (
				wantName string
				wantMode fs.FileMode
			)

			if sl.IsSymlink {
				wantName = vfs.Base(newPath)
				wantMode = fs.ModeSymlink | fs.ModePerm
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
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Lstat(nonExistingFile)
		CheckPathError(t, err).OpLstat(vfs).Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("LStatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(testDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Lstat(subDirOnFile)
		CheckPathError(t, err).OpLstat(vfs).Path(subDirOnFile).Err(avfs.ErrNotADirectory)
	})
}

// TestMkdir tests Mkdir function.
func (sfs *SuiteFS) TestMkdir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

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
			for start, end, i, isLast := 1, 0, 0, false; !isLast; start, i = end+1, i+1 {
				end, isLast = avfs.SegmentPath(vfs.PathSeparator(), dir.Path, start)
				part := dir.Path[start:end]
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, part)
				info, err := vfs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= vfs.GetUMask()
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

			switch vfs.OSType() {
			case avfs.OsWindows:
				CheckPathError(t, err).Op("mkdir").Path(path).Err(avfs.ErrWinPathNotFound)
			default:
				CheckPathError(t, err).Op("mkdir").Path(path).Err(avfs.ErrNoSuchFileOrDir)
			}
		}
	})

	t.Run("MkdirEmptyName", func(t *testing.T) {
		err := vfs.Mkdir("", avfs.DefaultFilePerm)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("mkdir").Path("").Err(avfs.ErrWinPathNotFound)
		default:
			CheckPathError(t, err).Op("mkdir").Path("").Err(avfs.ErrNoSuchFileOrDir)
		}
	})

	t.Run("MkdirOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.Mkdir(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(subDirOnFile).Err(avfs.ErrNotADirectory)
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

			curPath := testDir
			for start, end, i, isLast := 1, 0, 0, false; !isLast; start, i = end+1, i+1 {
				end, isLast = avfs.SegmentPath(vfs.PathSeparator(), dir.Path, start)
				part := dir.Path[start:end]
				wantMode := dir.WantModes[i]

				curPath = vfs.Join(curPath, part)
				info, err := vfs.Stat(curPath)
				if err != nil {
					t.Fatalf("stat %s : want error to be nil, got %v", curPath, err)
				}

				wantMode &^= vfs.GetUMask()
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
			if err != nil {
				t.Errorf("MkdirAll %s : want error to be nil, got error %v", path, err)
			}
		}
	})

	t.Run("MkdirAllOnFile", func(t *testing.T) {
		err := vfs.MkdirAll(existingFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(existingFile).Err(avfs.ErrNotADirectory)
	})

	t.Run("MkdirAllSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.MkdirAll(subDirOnFile, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(existingFile).Err(avfs.ErrNotADirectory)
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
		if err != nil {
			t.Errorf("want error to be nil, got %v", err)
		}

		wantDir := vfs.TempDir()
		dir := vfs.Dir(tmpDir)

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}
	})

	t.Run("MkdirTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.MkdirTemp("", badPattern)
		CheckPathError(t, err).Op("mkdirtemp").Path(badPattern).ErrAsString(avfs.ErrPatternHasSeparator)
	})

	t.Run("MkdirTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.MkdirTemp(existingFile, "")
		CheckPathError(t, err).Op("mkdir").Err(avfs.ErrNotADirectory)
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
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		gotData, err := io.ReadAll(f)
		if err != nil {
			t.Errorf("ReadAll : want error to be nil, got %v", err)
		}

		if !bytes.Equal(gotData, data) {
			t.Errorf("ReadAll : want error data to be %v, got %v", data, gotData)
		}
	})

	t.Run("OpenFileDirReadOnly", func(t *testing.T) {
		f, err := vfs.Open(existingDir)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		defer f.Close()

		dirs, err := f.ReadDir(-1)
		if err != nil {
			t.Errorf("ReadDir : want error to be nil, got %v", err)
		}

		if len(dirs) != 0 {
			t.Errorf("ReadDir : want number of directories to be 0, got %d", len(dirs))
		}
	})

	t.Run("OpenNonExistingFile", func(t *testing.T) {
		fileName := sfs.NonExistingFile(t, testDir)
		_, err := vfs.Open(fileName)
		CheckPathError(t, err).Op("open").Path(fileName).Err(avfs.ErrNoSuchFileOrDir)
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
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.Write(defaultData)
		if err != nil {
			t.Errorf("Write : want error to be nil, got %v", err)
		}

		if n != len(defaultData) {
			t.Errorf("Write : want bytes written to be %d, got %d", len(defaultData), n)
		}

		n, err = f.Read(buf3)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("read").Path(existingFile).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("read").Path(existingFile).Err(avfs.ErrBadFileDesc)
		}

		if n != 0 {
			t.Errorf("Read : want bytes written to be 0, got %d", n)
		}

		n, err = f.ReadAt(buf3, 3)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("read").Path(existingFile).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("read").Path(existingFile).Err(avfs.ErrBadFileDesc)
		}

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		if err != nil {
			t.Errorf("Chmod : want error to be nil, got %v", err)
		}

		if vfs.HasFeature(avfs.FeatIdentityMgr) {
			u := vfs.CurrentUser()
			err = f.Chown(u.Uid(), u.Gid())
			if err != nil {
				t.Errorf("Chown : want error to be nil, got %v", err)
			}
		}

		fst, err := f.Stat()
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		if err != nil {
			t.Errorf("Chmod : want error to be nil, got %v", err)
		}

		err = f.Sync()
		if err != nil {
			t.Errorf("Sync : want error to be nil, got %v", err)
		}
	})

	t.Run("OpenFileAppend", func(t *testing.T) {
		existingFile := sfs.ExistingFile(t, testDir, data)
		appendData := []byte("appendData")

		f, err := vfs.OpenFile(existingFile, os.O_WRONLY|os.O_APPEND, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		n, err := f.Write(appendData)
		if err != nil {
			t.Errorf("Write : want error to be nil, got %v", err)
		}

		if n != len(appendData) {
			t.Errorf("Write : want error to be %d, got %d", len(defaultData), n)
		}

		_ = f.Close()

		gotContent, err := vfs.ReadFile(existingFile)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		wantContent := append(data, appendData...)
		if !bytes.Equal(wantContent, gotContent) {
			t.Errorf("ReadAll : want content to be %s, got %s", wantContent, gotContent)
		}
	})

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		existingFile := sfs.ExistingFile(t, testDir, data)

		f, err := vfs.Open(existingFile)
		if err != nil {
			t.Errorf("Open : want error to be nil, got %v", err)
		}

		defer f.Close()

		n, err := f.Write(defaultData)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("write").Path(existingFile).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("write").Path(existingFile).Err(avfs.ErrBadFileDesc)
		}

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.WriteAt(defaultData, 3)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("write").Path(existingFile).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("write").Path(existingFile).Err(avfs.ErrBadFileDesc)
		}

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		if err != nil {
			t.Errorf("Chmod : want error to be nil, got %v", err)
		}

		if vfs.HasFeature(avfs.FeatIdentityMgr) {
			u := vfs.CurrentUser()
			err = f.Chown(u.Uid(), u.Gid())
			if err != nil {
				t.Errorf("Chown : want error to be nil, got %v", err)
			}
		}

		fst, err := f.Stat()
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("truncate").Path(existingFile).Err(avfs.ErrWinAccessDenied)
		default:
			CheckPathError(t, err).Op("truncate").Path(existingFile).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("OpenFileWriteOnDir", func(t *testing.T) {
		existingDir := sfs.ExistingDir(t, testDir)

		f, err := vfs.OpenFile(existingDir, os.O_WRONLY, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(existingDir).Err(avfs.ErrIsADirectory)

		if !reflect.ValueOf(f).IsNil() {
			t.Errorf("OpenFile : want file to be nil, got %v", f)
		}
	})

	t.Run("OpenFileExcl", func(t *testing.T) {
		fileExcl := vfs.Join(testDir, "fileExcl")

		f, err := vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err != nil {
			t.Errorf("OpenFile : want error to be nil, got %v", err)
		}

		f.Close()

		_, err = vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("open").Path(fileExcl).Err(avfs.ErrWinFileExists)
		default:
			CheckPathError(t, err).Op("open").Path(fileExcl).Err(avfs.ErrFileExists)
		}
	})

	t.Run("OpenFileNonExistingPath", func(t *testing.T) {
		nonExistingPath := vfs.Join(testDir, "non/existing/path")
		_, err := vfs.OpenFile(nonExistingPath, os.O_CREATE, avfs.DefaultFilePerm)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("open").Path(nonExistingPath).Err(avfs.ErrWinPathNotFound)
		default:
			CheckPathError(t, err).Op("open").Path(nonExistingPath).Err(avfs.ErrNoSuchFileOrDir)
		}
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

	existingFile := rndTree.Files[0]

	t.Run("ReadDirAll", func(t *testing.T) {
		dirEntries, err := vfs.ReadDir(testDir)
		if err != nil {
			t.Fatalf("ReadDir : want error to be nil, got %v", err)
		}

		var gDirs, gFiles, gSymlinks int
		for _, dirEntry := range dirEntries {
			_, err = dirEntry.Info()
			if err != nil {
				t.Fatalf("want error to be nil, got %v", err)
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
			CheckPathError(t, err).Op("readdir").Path(existingFile).Err(avfs.ErrNotADirectory)
		default:
			CheckPathError(t, err).Op("readdirent").Path(existingFile).Err(avfs.ErrNotADirectory)
		}
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
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

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

		wantErr := avfs.ErrPermDenied

		if vfs.OSType() == avfs.OsWindows {
			wantErr = avfs.ErrWinNotReparsePoint
		}

		CheckPathError(t, err).Op("readlink").Path(testDir).Err(wantErr)

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
			path := vfs.Join(testDir, dir.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, err).Op("readlink").Path(path).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			path := vfs.Join(testDir, file.Path)

			_, err := vfs.Readlink(path)
			CheckPathError(t, err).Op("readlink").Path(path).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("ReadLinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Readlink(nonExistingFile)
		CheckPathError(t, err).Op("readlink").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
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
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path, err)
			}

			err = vfs.Remove(path)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", path, err)
			}

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat(vfs).Path(path).Err(avfs.ErrNoSuchFileOrDir)
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		wantDirErr := avfs.ErrDirNotEmpty

		if vfs.OSType() == avfs.OsWindows {
			wantDirErr = avfs.ErrWinDirNotEmpty
		}

		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

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
				CheckPathError(t, err).OpStat(vfs).Path(path).Err(avfs.ErrNoSuchFileOrDir)

				continue
			}

			CheckPathError(t, err).Op("remove").Path(path).Err(wantDirErr)

			_, err = vfs.Stat(path)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", path, err)
			}
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			newPath := vfs.Join(testDir, sl.NewName)

			err := vfs.Remove(newPath)
			if err != nil {
				t.Errorf("Remove %s : want error to be nil, got %v", newPath, err)
			}

			_, err = vfs.Stat(newPath)
			CheckPathError(t, err).OpStat(vfs).Path(newPath).Err(avfs.ErrNoSuchFileOrDir)
		}
	})

	t.Run("RemoveNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Remove(nonExistingFile)
		CheckPathError(t, err).Op("remove").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
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
		if err != nil {
			t.Fatalf("RemoveAll %s : want error to be nil, got %v", baseDir, err)
		}

		wantErr := avfs.ErrNoSuchFileOrDir

		if vfs.OSType() == avfs.OsWindows {
			wantErr = avfs.ErrWinPathNotFound
		}

		for _, dir := range dirs {
			path := vfs.Join(baseDir, dir.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat(vfs).Path(path).Err(wantErr)
		}

		for _, file := range files {
			path := vfs.Join(baseDir, file.Path)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat(vfs).Path(path).Err(wantErr)
		}

		for _, sl := range symlinks {
			path := vfs.Join(baseDir, sl.NewName)

			_, err = vfs.Stat(path)
			CheckPathError(t, err).OpStat(vfs).Path(path).Err(wantErr)
		}

		_, err = vfs.Stat(baseDir)
		CheckPathError(t, err).OpStat(vfs).Path(baseDir).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("RemoveAllOneFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		err := vfs.RemoveAll(existingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", existingFile, err)
		}
	})

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		_ = sfs.SampleDirs(t, baseDir)

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
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.RemoveAll(nonExistingFile)
		if err != nil {
			t.Errorf("RemoveAll %s : want error to be nil, got %v", nonExistingFile, err)
		}
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
			if err != nil {
				t.Errorf("Rename %s %s : want error to be nil, got %v", oldPath, newPath, err)
			}

			_, err = vfs.Stat(oldPath)
			CheckPathError(t, err).OpStat(vfs).Path(oldPath).Err(avfs.ErrNoSuchFileOrDir)

			_, err = vfs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})

	t.Run("RenameFile", func(t *testing.T) {
		_ = sfs.SampleDirs(t, testDir)
		files := sfs.SampleFiles(t, testDir)

		for _, file := range files {
			oldPath := vfs.Join(testDir, file.Path)
			newPath := vfs.Join(testDir, vfs.Base(oldPath))

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
				CheckPathError(t, err).OpStat(vfs).Path(oldPath).Err(avfs.ErrNoSuchFileOrDir)
			}

			_, err = vfs.Stat(newPath)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
			}
		}
	})

	t.Run("RenameNonExistingFile", func(t *testing.T) {
		srcNonExistingFile := vfs.Join(testDir, "srcNonExistingFile1")
		dstNonExistingFile := vfs.Join(testDir, "dstNonExistingFile2")

		err := vfs.Rename(srcNonExistingFile, dstNonExistingFile)
		CheckLinkError(t, err).Op("rename").Old(srcNonExistingFile).New(dstNonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("RenameDirToExistingDir", func(t *testing.T) {
		srcExistingDir := sfs.ExistingDir(t, testDir)
		dstExistingDir := sfs.ExistingDir(t, testDir)

		err := vfs.Rename(srcExistingDir, dstExistingDir)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckLinkError(t, err).Op("rename").Old(srcExistingDir).New(dstExistingDir).Err(avfs.ErrWinAccessDenied)
		default:
			CheckLinkError(t, err).Op("rename").Old(srcExistingDir).New(dstExistingDir).Err(avfs.ErrFileExists)
		}
	})

	t.Run("RenameFileToExistingFile", func(t *testing.T) {
		srcExistingFile := sfs.ExistingFile(t, testDir, data)
		dstExistingFile := sfs.EmptyFile(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingFile)
		if err != nil {
			t.Errorf("Rename : want error to be nil, got %v", err)
		}

		_, err = vfs.Stat(srcExistingFile)
		CheckPathError(t, err).OpStat(vfs).Path(srcExistingFile).Err(avfs.ErrNoSuchFileOrDir)

		info, err := vfs.Stat(dstExistingFile)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		if int(info.Size()) != len(data) {
			t.Errorf("Stat : want size to be %d, got %d", len(data), info.Size())
		}
	})

	t.Run("RenameFileToExistingDir", func(t *testing.T) {
		srcExistingFile := sfs.ExistingFile(t, testDir, data)
		dstExistingDir := sfs.ExistingDir(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingDir)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckLinkError(t, err).Op("rename").Old(srcExistingFile).New(dstExistingDir).Err(avfs.ErrWinAccessDenied)
		default:
			CheckLinkError(t, err).Op("rename").Old(srcExistingFile).New(dstExistingDir).Err(avfs.ErrFileExists)
		}
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
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			err = vfs.Link(path1, path2)
			if err != nil {
				t.Fatalf("Link %s : want error to be nil, got %v", path1, err)
			}

			info2, err := vfs.Stat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			err = vfs.Remove(path2)
			if err != nil {
				t.Fatalf("Remove %s : want error to be nil, got %v", path2, err)
			}
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
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			err = vfs.Symlink(path1, path2)
			if err != nil {
				t.Fatalf("TestSymlink %s : want error to be nil, got %v", path1, err)
			}

			info2, err := vfs.Stat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			info3, err := vfs.Lstat(path2)
			if err != nil {
				t.Fatalf("Stat %s : want error to be nil, got %v", path1, err)
			}

			if vfs.SameFile(info1, info3) {
				t.Fatalf("SameFile %s, %s : not the same file\n%v\n%v", path1, path2, info1, info3)
			}

			err = vfs.Remove(path2)
			if err != nil {
				t.Fatalf("Remove %s : want error to be nil, got %v", path2, err)
			}
		}
	})
}

// TestStat tests Stat function.
func (sfs *SuiteFS) TestStat(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Stat(testDir)
		CheckPathError(t, err).OpStat(vfs).Path(testDir).Err(avfs.ErrPermDenied)

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Path)

			info, err := vfs.Stat(path)
			if err != nil {
				t.Errorf("Stat %s : want error to be nil, got %v", path, err)

				continue
			}

			if vfs.Base(path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", path, vfs.Base(path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.GetUMask()
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
		for _, sl := range GetSampleSymlinksEval(vfs) {
			newPath := vfs.Join(testDir, sl.NewName)
			oldPath := vfs.Join(testDir, sl.OldName)

			info, err := vfs.Stat(newPath)
			if err != nil {
				if sl.WantErr == nil {
					t.Errorf("Stat %s : want error to be nil, got %v", newPath, err)
				}

				CheckPathError(t, err).OpStat(vfs).Path(newPath).Err(sl.WantErr)

				continue
			}

			var (
				wantName string
				wantMode fs.FileMode
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

			if sfs.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", newPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Stat(nonExistingFile)
		CheckPathError(t, err).OpStat(vfs).Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("StatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(testDir, files[0].Path, "subDirOnFile")

		_, err := vfs.Stat(subDirOnFile)
		CheckPathError(t, err).OpStat(vfs).Path(subDirOnFile).Err(avfs.ErrNotADirectory)
	})
}

// TestSymlink tests Symlink function.
func (sfs *SuiteFS) TestSymlink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Symlink(testDir, testDir)
		wantErr := avfs.ErrPermDenied

		if vfs.OSType() == avfs.OsWindows {
			wantErr = avfs.ErrWinPrivilegeNotHeld
		}

		CheckLinkError(t, err).Op("symlink").Old(testDir).New(testDir).Err(wantErr)

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
			if err != nil {
				t.Errorf("TestSymlink %s %s : want error to be nil, got %v", oldPath, newPath, err)
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

	u := vfs.CurrentUser()
	if vfs.HasFeature(avfs.FeatReadOnly) {
		u = sfs.vfsSetup.CurrentUser()
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
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			d, err := vfs.ReadFile(path)
			if err != nil {
				t.Errorf("Truncate : want error to be nil, got %v", err)
			}

			if len(d) != i {
				t.Errorf("Truncate : want length to be %d, got %d", i, len(d))
			}
		}
	})

	t.Run("TruncateNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Truncate(nonExistingFile, 0)
		CheckPathError(t, err).Op("truncate").Path(nonExistingFile).Err(avfs.ErrNoSuchFileOrDir)
	})

	t.Run("TruncateOnDir", func(t *testing.T) {
		err := vfs.Truncate(testDir, 0)

		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("open").Path(testDir).Err(avfs.ErrIsADirectory)
		default:
			CheckPathError(t, err).Op("truncate").Path(testDir).Err(avfs.ErrIsADirectory)
		}
	})

	t.Run("TruncateSizeNegative", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)

		err := vfs.Truncate(path, -1)
		switch vfs.OSType() {
		case avfs.OsWindows:
			CheckPathError(t, err).Op("truncate").Path(path).Err(avfs.ErrWinNegativeSeek)
		default:
			CheckPathError(t, err).Op("truncate").Path(path).Err(avfs.ErrInvalidArgument)
		}
	})

	t.Run("TruncateSizeBiggerFileSize", func(t *testing.T) {
		path := sfs.ExistingFile(t, testDir, data)
		newSize := len(data) * 2

		err := vfs.Truncate(path, int64(newSize))
		if err != nil {
			t.Errorf("Truncate : want error to be nil, got %v", err)
		}

		info, err := vfs.Stat(path)
		if err != nil {
			t.Errorf("Stat : want error to be nil, got %v", err)
		}

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		gotContent, err := vfs.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile : want error to be nil, got %v", err)
		}

		wantAdded := bytes.Repeat([]byte{0}, len(data))
		gotAdded := gotContent[len(data):]
		if !bytes.Equal(wantAdded, gotAdded) {
			t.Errorf("Bytes Added : want %v, got %v", wantAdded, gotAdded)
		}
	})
}

// TestUmask tests UMask and GetUMask functions.
func (sfs *SuiteFS) TestUmask(t *testing.T, testDir string) {
	const umaskTest = 0o077

	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		vfs.UMask(0)

		if um := vfs.GetUMask(); um <= 0 {
			t.Errorf("GetUMask : want umask to be > 0, got %d", um)
		}

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
		if err != nil {
			t.Errorf("WriteFile : want error to be nil, got %v", err)
		}

		rb, err := vfs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

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
		if err != nil {
			t.Fatalf("Open : want error to be nil, got %v", err)
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
		if err != nil {
			t.Errorf("Create %s : want error to be nil, got %v", path, err)
		}

		n, err := f.WriteString(string(data))
		if err != nil {
			t.Errorf("WriteString : want error to be nil, got %v", err)
		}

		if len(data) != n {
			t.Errorf("WriteString : want written bytes to be %d, got %d", len(data), n)
		}

		f.Close()

		rb, err := vfs.ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile : want error to be nil, got %v", err)
		}

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}
