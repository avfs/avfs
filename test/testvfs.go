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
	"errors"
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
		CheckPathError(t, err).Op("chdir").Path(testDir).ErrPermDenied()

		_, err = vfs.Getwd()
		CheckPathError(t, err).Op("getwd").Path("").ErrPermDenied()

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	existingFile := sfs.EmptyFile(t, testDir)

	t.Run("ChdirAbsolute", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(dir.Path)
			CheckNoError(t, "Chdir "+dir.Path, err)

			curDir, err := vfs.Getwd()
			CheckNoError(t, "Getwd", err)

			if curDir != dir.Path {
				t.Errorf("Getwd : want current directory to be %s, got %s", dir.Path, curDir)
			}
		}
	})

	t.Run("ChdirRelative", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(testDir)
			if !CheckNoError(t, "Chdir "+testDir, err) {
				return
			}

			relPath := strings.TrimPrefix(dir.Path, testDir)[1:]

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
			path := vfs.Join(dir.Path, defaultNonExisting)

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

		pts := sfs.NewPermTests(testDir, "Chdir", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			return vfs.Chdir(path)
		})
	})
}

// TestChmod tests Chmod function.
func (sfs *SuiteFS) TestChmod(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Chmod(testDir, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("chmod").Path(testDir).ErrPermDenied()

		return
	}

	existingDir := sfs.ExistingDir(t, testDir)
	existingFile := sfs.EmptyFile(t, testDir)

	t.Run("ChmodDir", func(t *testing.T) {
		for mode := fs.FileMode(1); mode <= 0o777; mode++ {
			wantMode := mode

			err := vfs.Chmod(existingDir, wantMode)
			CheckNoError(t, "Chmod "+existingDir, err)

			fst, err := vfs.Stat(existingDir)
			CheckNoError(t, "Stat "+existingDir, err)

			gotMode := fst.Mode() & fs.ModePerm

			// On Windows, only the 0200 bit (owner writable) of mode is used.
			if vfs.OSType() == avfs.OsWindows {
				wantMode &= 0o200
				gotMode &= 0o200
			}

			if gotMode != wantMode {
				t.Errorf("Stat %s : want mode to be %03o, got %03o", existingDir, wantMode, gotMode)
			}
		}
	})

	t.Run("ChmodFile", func(t *testing.T) {
		for mode := fs.FileMode(1); mode <= 0o777; mode++ {
			wantMode := mode

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

		pts := sfs.NewPermTests(testDir, "Chmod", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
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
				err := vfs.Chown(dir.Path, u.Uid(), u.Gid())
				CheckNoError(t, "Chown "+dir.Path, err)

				fst, err := vfs.Stat(dir.Path)
				if !CheckNoError(t, "Stat "+dir.Path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						dir.Path, u.Uid(), u.Gid(), uid, gid)
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
				err := vfs.Chown(file.Path, u.Uid(), u.Gid())
				CheckNoError(t, "Chown "+file.Path, err)

				fst, err := vfs.Stat(file.Path)
				if !CheckNoError(t, "Stat "+file.Path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						file.Path, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("ChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Chown(nonExistingFile, 0, 0)
		CheckPathError(t, err).Op("chown").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)
	})

	t.Run("ChownPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pts := sfs.NewPermTests(testDir, "Chown", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			return vfs.Chown(path, 0, 0)
		})
	})
}

// TestChroot tests Chroot function.
func (sfs *SuiteFS) TestChroot(t *testing.T, testDir string) {
	if !sfs.canTestPerm {
		return
	}

	vfs := sfs.vfsTest
	if !vfs.HasFeature(avfs.FeatChroot) {
		err := vfs.Chroot(testDir)
		CheckPathError(t, err).Op("chroot").Path(testDir).
			Err(avfs.ErrOpNotPermitted, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)

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
		CheckPathError(t, err).Op("chtimes").Path(testDir).ErrPermDenied()

		return
	}

	t.Run("Chtimes", func(t *testing.T) {
		_ = sfs.SampleDirs(t, testDir)
		files := sfs.SampleFiles(t, testDir)
		tomorrow := time.Now().AddDate(0, 0, 1)

		for _, file := range files {
			err := vfs.Chtimes(file.Path, tomorrow, tomorrow)
			CheckNoError(t, "Chtimes "+file.Path, err)

			infos, err := vfs.Stat(file.Path)
			CheckNoError(t, "Stat "+file.Path, err)

			if infos.ModTime() != tomorrow {
				t.Errorf("Chtimes %s : want modtime to bo %s, got %s", file.Path, tomorrow, infos.ModTime())
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

		pts := sfs.NewPermTests(testDir, "Chtimes", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
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
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

		return
	}

	t.Run("CreatePerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pts := sfs.NewPermTests(testDir, "Create", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
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
		_, err := vfs.CreateTemp(testDir, "CreateTemp")
		CheckPathError(t, err).Op("createtemp").ErrPermDenied()

		return
	}

	t.Run("CreateTempEmptyDir", func(t *testing.T) {
		f, err := vfs.CreateTemp("", "CreateTemp")
		CheckNoError(t, "CreateTemp", err)

		wantDir := vfs.TempDir()
		dir := vfs.Dir(f.Name())

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}

		err = f.Close()
		CheckNoError(t, "Close", err)

		err = vfs.Remove(f.Name())
		CheckNoError(t, "Remove", err)
	})

	t.Run("CreateTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.CreateTemp("", badPattern)
		CheckPathError(t, err).Op("createtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator)
	})

	t.Run("CreateTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.CreateTemp(existingFile, "CreateTempOnFile")
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
		CheckPathError(t, err).OpLstat().Path(testDir).ErrPermDenied()

		return
	}

	_ = sfs.SampleDirs(t, testDir)
	_ = sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	t.Run("EvalSymlink", func(t *testing.T) {
		symlinks := sfs.GetSampleSymlinksEval(testDir)
		for _, sl := range symlinks {
			wantPath := sl.OldPath
			slPath := sl.NewPath

			gotPath, err := vfs.EvalSymlinks(sl.NewPath)
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
	wantTmpDir := "/tmp"

	if vfs.OSType() == avfs.OsWindows {
		userName := vfs.User().Name()
		wantTmpDir = avfs.ShortPathName(vfs.Join(avfs.DefaultVolume, `\Users\`, userName, `\AppData\Local\Temp`))
	}

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
				err := vfs.Lchown(dir.Path, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+dir.Path, err)

				fst, err := vfs.Lstat(dir.Path)
				if !CheckNoError(t, "Lstat "+dir.Path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						dir.Path, u.Uid(), u.Gid(), uid, gid)
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
				err := vfs.Lchown(file.Path, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+file.Path, err)

				fst, err := vfs.Lstat(file.Path)
				if !CheckNoError(t, "Lstat "+file.Path, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						file.Path, u.Uid(), u.Gid(), uid, gid)
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

			for _, sl := range symlinks {
				err := vfs.Lchown(sl.NewPath, u.Uid(), u.Gid())
				CheckNoError(t, "Lchown "+sl.NewPath, err)

				fst, err := vfs.Lstat(sl.NewPath)
				if !CheckNoError(t, "Lstat "+sl.NewPath, err) {
					continue
				}

				sst := vfs.ToSysStat(fst)

				uid, gid := sst.Uid(), sst.Gid()
				if uid != u.Uid() || gid != u.Gid() {
					t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
						sl.NewPath, u.Uid(), u.Gid(), uid, gid)
				}
			}
		}
	})

	t.Run("LChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.Lchown(nonExistingFile, 0, 0)
		CheckPathError(t, err).Op("lchown").Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinNotSupported, avfs.OsWindows)
	})

	t.Run("LchownPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pts := sfs.NewPermTests(testDir, "Lchown", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			return vfs.Lchown(path, 0, 0)
		})
	})
}

// TestLink tests Link function.
func (sfs *SuiteFS) TestLink(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatHardlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Link(testDir, testDir)
		CheckLinkError(t, err).Op("link").Old(testDir).New(testDir).ErrPermDenied()

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	pathLinks := sfs.ExistingDir(t, testDir)

	t.Run("LinkCreate", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(file.Path, newPath)
			CheckNoError(t, "Link "+file.Path+" "+newPath, err)

			newContent, err := vfs.ReadFile(newPath)
			CheckNoError(t, "ReadFile "+newPath, err)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkExisting", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(file.Path, newPath)
			CheckLinkError(t, err).Op("link").Old(file.Path).New(newPath).
				Err(avfs.ErrFileExists, avfs.OsLinux).
				Err(avfs.ErrWinAlreadyExists, avfs.OsWindows)
		}
	})

	t.Run("LinkRemove", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Remove(file.Path)
			CheckNoError(t, "Remove "+file.Path, err)

			newContent, err := vfs.ReadFile(newPath)
			CheckNoError(t, "ReadFile "+file.Path, err)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkErrorDir", func(t *testing.T) {
		for _, dir := range dirs {
			newPath := vfs.Join(testDir, defaultDir)

			err := vfs.Link(dir.Path, newPath)
			CheckLinkError(t, err).Op("link").Old(dir.Path).New(newPath).
				Err(avfs.ErrOpNotPermitted, avfs.OsLinux).
				Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
		}
	})

	t.Run("LinkErrorFile", func(t *testing.T) {
		for _, file := range files {
			InvalidPath := vfs.Join(file.Path, defaultNonExisting)
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
		CheckLinkError(t, err).Op("link").Old(nonExistingFile).New(existingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})
}

// TestLstat tests Lstat function.
func (sfs *SuiteFS) TestLstat(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.Lstat(testDir)
		CheckPathError(t, err).OpLstat().Path(testDir).ErrPermDenied()

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	sfs.SampleSymlinks(t, testDir)

	vfs = sfs.vfsTest

	t.Run("LstatDir", func(t *testing.T) {
		for _, dir := range dirs {
			info, err := vfs.Lstat(dir.Path)
			if !CheckNoError(t, "Lstat "+dir.Path, err) {
				continue
			}

			if vfs.Base(dir.Path) != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", dir.Path, vfs.Base(dir.Path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", dir.Path, wantMode, info.Mode())
			}
		}
	})

	t.Run("LstatFile", func(t *testing.T) {
		for _, file := range files {
			info, err := vfs.Lstat(file.Path)
			if !CheckNoError(t, "Lstat "+file.Path, err) {
				continue
			}

			if info.Name() != vfs.Base(file.Path) {
				t.Errorf("Lstat %s : want name to be %s, got %s", file.Path, vfs.Base(file.Path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = 0o666
			}

			if wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", file.Path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", file.Path, wantSize, info.Size())
			}
		}
	})

	t.Run("LstatSymlink", func(t *testing.T) {
		for _, sl := range sfs.GetSampleSymlinksEval(testDir) {
			info, err := vfs.Lstat(sl.NewPath)
			if !CheckNoError(t, "Lstat", err) {
				continue
			}

			wantName := vfs.Base(sl.OldPath)
			wantMode := sl.Mode

			if sl.IsSymlink {
				wantName = vfs.Base(sl.NewPath)
				wantMode = fs.ModeSymlink | fs.ModePerm
			}

			if wantName != info.Name() {
				t.Errorf("Lstat %s : want name to be %s, got %s", sl.NewPath, wantName, info.Name())
			}

			if vfs.OSType() != avfs.OsWindows && wantMode != info.Mode() {
				t.Errorf("Lstat %s : want mode to be %s, got %s", sl.NewPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("LStatNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		_, err := vfs.Lstat(nonExistingFile)
		CheckPathError(t, err).OpLstat().Path(nonExistingFile).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
	})

	t.Run("LStatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(files[0].Path, "subDirOnFile")

		_, err := vfs.Lstat(subDirOnFile)
		CheckPathError(t, err).OpLstat().Path(subDirOnFile).
			Err(avfs.ErrNotADirectory, avfs.OsLinux).
			Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
	})
}

// TestMkdir tests Mkdir function.
func (sfs *SuiteFS) TestMkdir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Mkdir(testDir, avfs.DefaultDirPerm)
		CheckPathError(t, err).Op("mkdir").Path(testDir).ErrPermDenied()

		return
	}

	dirs := sfs.GetSampleDirs(testDir)

	t.Run("Mkdir", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Mkdir(dir.Path, dir.Mode)
			if err != nil {
				t.Errorf("mkdir : want no error, got %v", err)
			}

			fi, err := vfs.Stat(dir.Path)
			if err != nil {
				t.Errorf("stat '%s' : want no error, got %v", dir.Path, err)

				continue
			}

			if !fi.IsDir() {
				t.Errorf("stat '%s' : want path to be a directory, got mode %s", dir.Path, fi.Mode())
			}

			if fi.Size() < 0 {
				t.Errorf("stat '%s': want directory size to be >= 0, got %d", dir.Path, fi.Size())
			}

			now := time.Now()
			if now.Sub(fi.ModTime()) > 2*time.Second {
				t.Errorf("stat '%s' : want time to be %s, got %s", dir.Path, time.Now(), fi.ModTime())
			}

			name := vfs.Base(dir.Path)
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", dir.Path, name, fi.Name())
			}

			// Check directories permissions for each subdirectory of testDir.
			relDir := strings.TrimPrefix(dir.Path, testDir)
			want := strings.Count(relDir, string(vfs.PathSeparator()))

			got := len(dir.WantModes)
			if want != got {
				t.Fatalf("stat %s : want %d directory parts, got %d", dir.Path, want, got)
			}

			curPath := testDir

			pi := avfs.NewPathIterator(vfs, relDir)
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
					t.Errorf("stat %s %s : want mode to be %s, got %s", dir.Path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirOnExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Mkdir(dir.Path, dir.Mode)
			if !errors.Is(err, fs.ErrExist) {
				t.Errorf("mkdir %s : want IsExist(err) to be true, got error %v", dir.Path, err)
			}
		}
	})

	t.Run("MkdirOnNonExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(dir.Path, "can't", "create", "this")

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

		pts := sfs.NewPermTests(testDir, "Mkdir", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
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
		CheckPathError(t, err).Op("mkdir").Path(testDir).ErrPermDenied()

		return
	}

	existingFile := sfs.EmptyFile(t, testDir)
	dirs := sfs.GetSampleDirsAll(testDir)

	t.Run("MkdirAll", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.MkdirAll(dir.Path, dir.Mode)
			CheckNoError(t, "MkdirAll "+dir.Path, err)

			fi, err := vfs.Stat(dir.Path)
			CheckNoError(t, "Stat "+dir.Path, err)

			if !fi.IsDir() {
				t.Errorf("stat '%s' : want path to be a directory, got mode %s", dir.Path, fi.Mode())
			}

			if fi.Size() < 0 {
				t.Errorf("stat '%s': want directory size to be >= 0, got %d", dir.Path, fi.Size())
			}

			now := time.Now()
			if now.Sub(fi.ModTime()) > 2*time.Second {
				t.Errorf("stat '%s' : want time to be %s, got %s", dir.Path, time.Now(), fi.ModTime())
			}

			name := vfs.Base(vfs.FromSlash(dir.Path))
			if fi.Name() != name {
				t.Errorf("stat '%s' : want path to be %s, got %s", dir.Path, name, fi.Name())
			}

			// Check directories permissions for each subdirectory of testDir.
			relDir := strings.TrimPrefix(dir.Path, testDir)
			want := strings.Count(relDir, string(vfs.PathSeparator()))

			got := len(dir.WantModes)
			if want != got {
				t.Fatalf("stat %s : want %d directory parts, got %d", dir.Path, want, got)
			}

			curPath := testDir

			pi := avfs.NewPathIterator(vfs, relDir)
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
					t.Errorf("stat %s %s : want mode to be %s, got %s", dir.Path, curPath, wantMode, mode)
				}
			}
		}
	})

	t.Run("MkdirAllExistingDir", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.MkdirAll(dir.Path, dir.Mode)
			CheckNoError(t, "MkdirAll "+dir.Path, err)
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

		pts := sfs.NewPermTests(testDir, "MkdirAll", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			newDir := vfs.Join(path, "newDir")

			return vfs.MkdirAll(newDir, avfs.DefaultDirPerm)
		})
	})
}

// TestMkdirTemp tests MkdirTemp function.
func (sfs *SuiteFS) TestMkdirTemp(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.MkdirTemp(testDir, "MkdirTemp")
		CheckPathError(t, err).Op("mkdirtemp").ErrPermDenied()

		return
	}

	t.Run("MkdirTempEmptyDir", func(t *testing.T) {
		tmpDir, err := vfs.MkdirTemp("", "MkdirTemp")
		CheckNoError(t, "MkdirTemp", err)

		wantDir := vfs.TempDir()
		dir := vfs.Dir(tmpDir)

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}

		err = vfs.Remove(tmpDir)
		CheckNoError(t, "Remove", err)
	})

	t.Run("MkdirTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.MkdirTemp("", badPattern)
		CheckPathError(t, err).Op("mkdirtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator)
	})

	t.Run("MkdirTempOnFile", func(t *testing.T) {
		existingFile := sfs.EmptyFile(t, testDir)

		_, err := vfs.MkdirTemp(existingFile, "MkdirTempOnFile")
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
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

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
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

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

		if vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
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

		if vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
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

// TestPathSeparator tests PathSeparator function.
func (sfs *SuiteFS) TestPathSeparator(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	wantSep := uint8('/')
	if vfs.OSType() == avfs.OsWindows {
		wantSep = '\\'
	}

	sep := vfs.PathSeparator()
	if sep != wantSep {
		t.Errorf("want separator to be %s, got %s", string(wantSep), string(sep))
	}
}

// TestReadDir tests ReadDir function.
func (sfs *SuiteFS) TestReadDir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		_, err := vfs.ReadDir(testDir)
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

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
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

		return
	}

	vfs = sfs.vfsTest

	t.Run("ReadFile", func(t *testing.T) {
		data := []byte("AAABBBCCCDDD")
		path := sfs.ExistingFile(t, testDir, data)

		rb, err := vfs.ReadFile(path)
		CheckNoError(t, "ReadFile", err)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("ReadFileNotExisting", func(t *testing.T) {
		path := sfs.NonExistingFile(t, testDir)

		rb, err := vfs.ReadFile(path)
		CheckPathError(t, err).Op("open").Path(path).
			Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
			Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

		if len(rb) != 0 {
			t.Errorf("ReadFile : want read bytes to be 0, got %d", len(rb))
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
			gotPath, err := vfs.Readlink(sl.NewPath)
			CheckNoError(t, "Readlink", err)

			if sl.OldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", sl.NewPath, sl.OldPath, gotPath)
			}
		}
	})

	t.Run("ReadlinkDir", func(t *testing.T) {
		for _, dir := range dirs {
			_, err := vfs.Readlink(dir.Path)
			CheckPathError(t, err).Op("readlink").Path(dir.Path).
				Err(avfs.ErrInvalidArgument, avfs.OsLinux).
				Err(avfs.ErrWinNotReparsePoint, avfs.OsWindows)
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			_, err := vfs.Readlink(file.Path)
			CheckPathError(t, err).Op("readlink").Path(file.Path).
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
		CheckPathError(t, err).Op("remove").Path(testDir).ErrPermDenied()

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	symlinks := sfs.SampleSymlinks(t, testDir)

	t.Run("RemoveFile", func(t *testing.T) {
		for _, file := range files {
			_, err := vfs.Stat(file.Path)
			if !CheckNoError(t, "Stat", err) {
				continue
			}

			err = vfs.Remove(file.Path)
			CheckNoError(t, "Remove", err)

			_, err = vfs.Stat(file.Path)
			CheckPathError(t, err).OpStat().Path(file.Path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinFileNotFound, avfs.OsWindows)
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		for _, dir := range dirs {
			dirInfos, err := vfs.ReadDir(dir.Path)
			if !CheckNoError(t, "ReadDir "+dir.Path, err) {
				continue
			}

			err = vfs.Remove(dir.Path)

			isLeaf := len(dirInfos) == 0
			if isLeaf {
				CheckNoError(t, "Remove "+dir.Path, err)

				_, err = vfs.Stat(dir.Path)
				CheckPathError(t, err).OpStat().Path(dir.Path).
					Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
					Err(avfs.ErrWinFileNotFound, avfs.OsWindows)

				continue
			}

			CheckPathError(t, err).Op("remove").Path(dir.Path).
				Err(avfs.ErrDirNotEmpty, avfs.OsLinux).
				Err(avfs.ErrWinDirNotEmpty, avfs.OsWindows)

			_, err = vfs.Stat(dir.Path)
			CheckNoError(t, "Stat "+dir.Path, err)
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			err := vfs.Remove(sl.NewPath)
			CheckNoError(t, "Remove "+sl.NewPath, err)

			_, err = vfs.Stat(sl.NewPath)
			CheckPathError(t, err).OpStat().Path(sl.NewPath).
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
		if !sfs.canTestPerm {
			return
		}

		pts := sfs.NewPermTests(testDir, "Remove", PermDefaultOptions)
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			return vfs.Remove(path)
		})
	})
}

// TestRemoveAll tests RemoveAll function.
func (sfs *SuiteFS) TestRemoveAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.RemoveAll(testDir)
		CheckPathError(t, err).Op("removeall").Path(testDir).ErrPermDenied()

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

		parentDir := vfs.Dir(baseDir)
		_, err = vfs.Stat(parentDir)
		CheckNoError(t, "Stat "+parentDir, err)

		for _, dir := range dirs {
			_, err = vfs.Stat(dir.Path)
			CheckPathError(t, err).OpStat().Path(dir.Path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}

		for _, file := range files {
			_, err = vfs.Stat(file.Path)
			CheckPathError(t, err).OpStat().Path(file.Path).
				Err(avfs.ErrNoSuchFileOrDir, avfs.OsLinux).
				Err(avfs.ErrWinPathNotFound, avfs.OsWindows)
		}

		for _, sl := range symlinks {
			_, err = vfs.Stat(sl.NewPath)
			CheckPathError(t, err).OpStat().Path(sl.NewPath).
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
			_, err = vfs.Stat(dir.Path)
			CheckNoError(t, "Stat", err)
		}
	})

	t.Run("RemoveAllNonExistingFile", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t, testDir)

		err := vfs.RemoveAll(nonExistingFile)
		CheckNoError(t, "RemoveAll "+nonExistingFile, err)
	})

	t.Run("RemoveAllPerm", func(t *testing.T) {
		if !sfs.canTestPerm {
			return
		}

		pts := sfs.NewPermTests(testDir, "RemoveAll", &PermOptions{IgnoreOp: true})
		pts.CreateDirs(t)

		pts.Test(t, func(path string) error {
			return vfs.RemoveAll(path)
		})
	})
}

// TestRename tests Rename function.
func (sfs *SuiteFS) TestRename(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Rename(testDir, testDir)
		CheckLinkError(t, err).Op("rename").Old(testDir).New(testDir).ErrPermDenied()

		return
	}

	data := []byte("data")

	t.Run("RenameDir", func(t *testing.T) {
		dirs := sfs.SampleDirs(t, testDir)

		for i := len(dirs) - 1; i >= 0; i-- {
			oldPath := dirs[i].Path
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
			newPath := vfs.Join(testDir, vfs.Base(file.Path))

			err := vfs.Rename(file.Path, newPath)
			CheckNoError(t, "Rename "+file.Path+" "+newPath, err)

			_, err = vfs.Stat(file.Path)

			switch {
			case file.Path == newPath:
				CheckNoError(t, "Stat "+file.Path, err)
			default:
				CheckPathError(t, err).OpStat().Path(file.Path).
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
	files1 := sfs.SampleFiles(t, testDir1)

	sfs.SampleDirs(t, testDir2)

	t.Run("SameFileLink", func(t *testing.T) {
		if !vfs.HasFeature(avfs.FeatHardlink) {
			return
		}

		for _, file1 := range files1 {
			path1 := file1.Path
			path2 := vfs.Join(testDir2, strings.TrimPrefix(path1, testDir1))

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

		for _, file1 := range files1 {
			path1 := file1.Path
			path2 := vfs.Join(testDir2, strings.TrimPrefix(path1, testDir1))

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
		CheckPathError(t, err).OpStat().Path(testDir).ErrPermDenied()

		return
	}

	dirs := sfs.SampleDirs(t, testDir)
	files := sfs.SampleFiles(t, testDir)
	_ = sfs.SampleSymlinks(t, testDir)

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			info, err := vfs.Stat(dir.Path)
			if !CheckNoError(t, "Stat "+dir.Path, err) {
				continue
			}

			if vfs.Base(dir.Path) != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", dir.Path, vfs.Base(dir.Path), info.Name())
			}

			wantMode := (dir.Mode | fs.ModeDir) &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = fs.ModeDir | fs.ModePerm
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", dir.Path, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatFile", func(t *testing.T) {
		for _, file := range files {
			info, err := vfs.Stat(file.Path)
			if !CheckNoError(t, "Stat "+file.Path, err) {
				continue
			}

			if info.Name() != vfs.Base(file.Path) {
				t.Errorf("Stat %s : want name to be %s, got %s", file.Path, vfs.Base(file.Path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = 0o666
			}

			if wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", file.Path, wantMode, info.Mode())
			}

			wantSize := int64(len(file.Content))
			if wantSize != info.Size() {
				t.Errorf("Lstat %s : want size to be %d, got %d", file.Path, wantSize, info.Size())
			}
		}
	})

	t.Run("StatSymlink", func(t *testing.T) {
		for _, sl := range sfs.GetSampleSymlinksEval(testDir) {
			info, err := vfs.Stat(sl.NewPath)
			if !CheckNoError(t, "Stat "+sl.NewPath, err) {
				continue
			}

			wantName := vfs.Base(sl.OldPath)
			if sl.IsSymlink {
				wantName = vfs.Base(sl.NewPath)
			}

			wantMode := sl.Mode
			if wantName != info.Name() {
				t.Errorf("Stat %s : want name to be %s, got %s", sl.NewPath, wantName, info.Name())
			}

			if sfs.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", sl.NewPath, wantMode, info.Mode())
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
		subDirOnFile := vfs.Join(files[0].Path, "subDirOnFile")

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
		symlinks := sfs.GetSampleSymlinks(testDir)
		for _, sl := range symlinks {
			err := vfs.Symlink(sl.OldPath, sl.NewPath)
			CheckNoError(t, "Symlink "+sl.OldPath+" "+sl.NewPath, err)

			gotPath, err := vfs.Readlink(sl.NewPath)
			CheckNoError(t, "Readlink "+sl.NewPath, err)

			if sl.OldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", sl.NewPath, sl.OldPath, gotPath)
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
		CheckPathError(t, err).Op("truncate").Path(testDir).ErrPermDenied()

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

		wantErr := error(avfs.ErrPermDenied)
		if vfs.OSType() == avfs.OsWindows {
			wantErr = avfs.ErrWinAccessDenied
		}

		_, err := vfs.SetUser(userName)
		if err != wantErr {
			t.Errorf("SetUser : want error to be %v, got %v", wantErr, err)
		}

		return
	}

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

// TestVolume tests VolumeAdd, VolumeDelete and VolumeList functions.
func (sfs *SuiteFS) TestVolume(t *testing.T, testDir string) {
	const testVolume = "Z:"

	vfs := sfs.vfsTest

	vm, ok := vfs.(avfs.VolumeManager)
	if !ok {
		return
	}

	if vfs.OSType() != avfs.OsWindows {
		vl := vm.VolumeList()
		if vl != nil {
			t.Errorf("Want volume list to be empty got %v", vl)
		}

		err := vm.VolumeAdd(testVolume)
		CheckPathError(t, err).Op("VolumeAdd").Path(testVolume).Err(avfs.ErrWinVolumeWindows)

		err = vm.VolumeDelete(testVolume)
		CheckPathError(t, err).Op("VolumeDelete").Path(testVolume).Err(avfs.ErrWinVolumeWindows)

		return
	}

	t.Run("VolumeManage", func(t *testing.T) {
		vl := vm.VolumeList()
		if len(vl) != 1 {
			t.Errorf("VolumeList : want 1 volume, got %d = %v", len(vl), vl)
		}

		err := vm.VolumeAdd(testVolume)
		CheckNoError(t, "VolumeAdd "+testVolume, err)

		vl = vm.VolumeList()
		if len(vl) != 2 {
			t.Errorf("VolumeList : want 2 volumes, got %d = %v", len(vl), vl)
		}

		err = vm.VolumeDelete(testVolume)
		CheckNoError(t, "VolumeDelete "+testVolume, err)

		vl = vm.VolumeList()
		if len(vl) != 1 {
			t.Errorf("VolumeList : want 1 volume, got %d = %v", len(vl), vl)
		}
	})
}

// TestWriteFile tests WriteFile function.
func (sfs *SuiteFS) TestWriteFile(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	if !vfs.HasFeature(avfs.FeatBasicFs) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.WriteFile(testDir, []byte{0}, avfs.DefaultFilePerm)
		CheckPathError(t, err).Op("open").Path(testDir).ErrPermDenied()

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

// TestWriteOnReadOnlyFS tests all write functions of a read only file system.
func (sfs *SuiteFS) TestWriteOnReadOnlyFS(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	existingFile := sfs.EmptyFile(t, testDir)

	if !vfs.HasFeature(avfs.FeatReadOnly) {
		// Skip tests if the file system is not read only
		return
	}

	t.Run("ReadOnlyFile", func(t *testing.T) {
		f, err := vfs.Open(existingFile)
		if !CheckNoError(t, "Open "+existingFile, err) {
			return
		}

		err = f.Chmod(0o777)
		CheckPathError(t, err).Op("chmod").Path(f.Name()).ErrPermDenied()

		err = f.Chown(0, 0)
		CheckPathError(t, err).Op("chown").Path(f.Name()).ErrPermDenied()

		err = f.Truncate(0)
		CheckPathError(t, err).Op("truncate").Path(f.Name()).ErrPermDenied()

		_, err = f.Write([]byte{})
		CheckPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied()

		_, err = f.WriteAt([]byte{}, 0)
		CheckPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied()

		_, err = f.WriteString("")
		CheckPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied()
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
