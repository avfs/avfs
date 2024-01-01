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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
)

func (ts *Suite) TestVFS(t *testing.T) {
	ts.RunTests(t, UsrTest,
		ts.TestAbs,
		ts.TestBase,
		ts.TestClean,
		ts.TestDir,
		ts.TestClone,
		ts.TestChdir,
		ts.TestChtimes,
		ts.TestCreate,
		ts.TestCreateTemp,
		ts.TestEvalSymlink,
		ts.TestFromToSlash,
		ts.TestGlob,
		ts.TestIsAbs,
		ts.TestJoin,
		ts.TestLink,
		ts.TestLstat,
		ts.TestMatch,
		ts.TestMkdir,
		ts.TestMkdirTemp,
		ts.TestMkdirAll,
		ts.TestName,
		ts.TestOpen,
		ts.TestOpenFileWrite,
		ts.TestPathSeparator,
		ts.TestReadDir,
		ts.TestReadFile,
		ts.TestReadlink,
		ts.TestRel,
		ts.TestRemove,
		ts.TestRemoveAll,
		ts.TestRename,
		ts.TestSameFile,
		ts.TestSplit,
		ts.TestSplitAbs,
		ts.TestStat,
		ts.TestSymlink,
		ts.TestTempDir,
		ts.TestToSysStat,
		ts.TestTruncate,
		ts.TestUser,
		ts.TestWalkDir,
		ts.TestWriteFile,
		ts.TestWriteString,
	)

	// Tests to be run as root
	adminUser := ts.idm.AdminUser()
	ts.RunTests(t, adminUser.Name(),
		ts.TestChmod,
		ts.TestChown,
		ts.TestChroot,
		ts.TestCreateSystemDirs,
		ts.TestCreateHomeDir,
		ts.TestLchown,
		ts.TestVolume,
		ts.TestWriteOnReadOnlyFS,
	)
}

// TestChdir tests Chdir and Getwd functions.
func (ts *Suite) TestChdir(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	dirs := ts.createSampleDirs(t, testDir)
	existingFile := ts.emptyFile(t, testDir)

	t.Run("ChdirAbsolute", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(dir.Path)
			RequireNoError(t, err, "Chdir %s", dir.Path)

			curDir, err := vfs.Getwd()
			RequireNoError(t, err, "Getwd")

			if curDir != dir.Path {
				t.Errorf("Getwd : want current directory to be %s, got %s", dir.Path, curDir)
			}
		}
	})

	t.Run("ChdirRelative", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Chdir(testDir)
			RequireNoError(t, err, "Chdir %s", testDir)

			relPath := strings.TrimPrefix(dir.Path, testDir)[1:]

			err = vfs.Chdir(relPath)
			RequireNoError(t, err, "Chdir %s", relPath, err)

			curDir, err := vfs.Getwd()
			RequireNoError(t, err, "Getwd")

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
			RequireNoError(t, err, "Getwd")

			err = vfs.Chdir(path)
			AssertPathError(t, err).Op("chdir").Path(path).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

			newPath, err := vfs.Getwd()
			RequireNoError(t, err, "Getwd")

			if newPath != oldPath {
				t.Errorf("Getwd : want current dir to be %s, got %s", oldPath, newPath)
			}
		}
	})

	t.Run("ChdirOnFile", func(t *testing.T) {
		err := vfs.Chdir(existingFile)
		AssertPathError(t, err).Op("chdir").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinDirNameInvalid).Test()
	})

	t.Run("ChdirPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Chdir")
		pts.Test(t, func(path string) error {
			return vfs.Chdir(path)
		})
	})
}

// TestChmod tests Chmod function.
func (ts *Suite) TestChmod(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Chmod(testDir, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("chmod").Path(testDir).ErrPermDenied().Test()

		return
	}

	existingDir := ts.existingDir(t, testDir)
	existingFile := ts.emptyFile(t, testDir)

	t.Run("ChmodDir", func(t *testing.T) {
		for mode := fs.FileMode(1); mode <= 0o777; mode++ {
			wantMode := mode

			err := vfs.Chmod(existingDir, wantMode)
			RequireNoError(t, err, "Chmod %s", existingDir)

			fst, err := vfs.Stat(existingDir)
			RequireNoError(t, err, "Stat %s", existingDir)

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
			RequireNoError(t, err, "Chmod %s", existingFile)

			fst, err := vfs.Stat(existingFile)
			RequireNoError(t, err, "Stat %s", existingFile)

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
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Chmod(nonExistingFile, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("chmod").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("ChmodPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Chmod")
		pts.Test(t, func(path string) error {
			return vfs.Chmod(path, 0o777)
		})
	})
}

// TestChown tests Chown function.
func (ts *Suite) TestChown(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	idm := vfs.Idm()

	if !ts.canTestPerm && vfs.HasFeature(avfs.FeatRealFS) ||
		vfs.HasFeature(avfs.FeatReadOnly) || avfs.CurrentOSType() == avfs.OsWindows {
		err := vfs.Chown(testDir, 0, 0)

		AssertPathError(t, err).Op("chown").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrOpNotPermitted).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNotSupported).Test()

		return
	}

	t.Run("ChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Chown(nonExistingFile, 0, 0)
		AssertPathError(t, err).Op("chown").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNotSupported).Test()
	})

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		wantUid, wantGid := 42, 42

		err := vfs.Chown(testDir, wantUid, wantGid)
		RequireNoError(t, err, "Chown %s", testDir)

		fst, err := vfs.Stat(testDir)
		RequireNoError(t, err, "Stat %s", testDir)

		sst := vfs.ToSysStat(fst)

		uid, gid := sst.Uid(), sst.Gid()
		if uid != wantUid || gid != wantGid {
			t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
				testDir, wantUid, wantGid, uid, gid)
		}

		return
	}

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	uis := UserInfos()

	t.Run("ChownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

			u, err := idm.LookupUser(wantName)
			RequireNoError(t, err, "LookupUser %s", wantName)

			for _, dir := range dirs {
				err := vfs.Chown(dir.Path, u.Uid(), u.Gid())
				RequireNoError(t, err, "Chown %s", dir.Path)

				fst, err := vfs.Stat(dir.Path)
				if !AssertNoError(t, err, "Stat %s", dir.Path) {
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
			RequireNoError(t, err, "LookupUser %s", wantName)

			for _, file := range files {
				err := vfs.Chown(file.Path, u.Uid(), u.Gid())
				RequireNoError(t, err, "Chown %s", file.Path)

				fst, err := vfs.Stat(file.Path)
				if !AssertNoError(t, err, "Stat %s", file.Path) {
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

	t.Run("ChownPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Chown")
		pts.Test(t, func(path string) error {
			return vfs.Chown(path, 0, 0)
		})
	})
}

// TestChroot tests Chroot function.
func (ts *Suite) TestChroot(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	vfsR, ok := vfs.(avfs.ChRooter)

	if !ok || !ts.canTestPerm {
		return
	}

	if os.Getenv("container") == "podman" {
		return
	}

	t.Run("Chroot", func(t *testing.T) {
		chrootDir := vfs.Join(testDir, "chroot")

		err := vfs.Mkdir(chrootDir, avfs.DefaultDirPerm)
		RequireNoError(t, err, "Mkdir %s", chrootDir)

		const chrootFile = "/file-within-the-chroot.txt"
		chrootFilePath := vfs.Join(chrootDir, chrootFile)

		err = vfs.WriteFile(chrootFilePath, nil, avfs.DefaultFilePerm)
		RequireNoError(t, err, "WriteFile %s", chrootFilePath)

		// A file descriptor is used to save the real root of the file system.
		// See https://devsidestory.com/exit-from-a-chroot-with-golang/
		fSave, err := vfs.OpenFile("/", os.O_RDONLY, 0)
		RequireNoError(t, err, "Open /")

		defer fSave.Close()

		err = vfsR.Chroot(chrootDir)
		RequireNoError(t, err, "Chroot ", chrootDir)

		_, err = vfs.Stat(chrootFile)
		RequireNoError(t, err, "Chroot %s", chrootFile)

		err = fSave.Chdir()
		RequireNoError(t, err, "Chdir")

		err = vfsR.Chroot(".")
		RequireNoError(t, err, "Chroot")

		_, err = vfs.Stat(chrootFilePath)
		RequireNoError(t, err, "Chroot %s ", chrootFilePath)
	})

	t.Run("ChrootOnFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)
		nonExistingFile := vfs.Join(existingFile, "invalid", "path")

		err := vfsR.Chroot(existingFile)
		AssertPathError(t, err).Op("chroot").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test()

		err = vfsR.Chroot(nonExistingFile)
		AssertPathError(t, err).Op("chroot").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test()
	})
}

// TestChtimes tests Chtimes function.
func (ts *Suite) TestChtimes(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Chtimes(testDir, time.Now(), time.Now())
		AssertPathError(t, err).Op("chtimes").Path(testDir).ErrPermDenied().Test()

		return
	}

	t.Run("Chtimes", func(t *testing.T) {
		_ = ts.createSampleDirs(t, testDir)
		files := ts.createSampleFiles(t, testDir)
		tomorrow := time.Now().AddDate(0, 0, 1)

		for _, file := range files {
			err := vfs.Chtimes(file.Path, tomorrow, tomorrow)
			RequireNoError(t, err, "Chtimes %s", file.Path)

			infos, err := vfs.Stat(file.Path)
			RequireNoError(t, err, "Stat %s", file.Path)

			if infos.ModTime() != tomorrow {
				t.Errorf("Chtimes %s : want modtime to bo %s, got %s", file.Path, tomorrow, infos.ModTime())
			}
		}
	})

	t.Run("ChtimesNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Chtimes(nonExistingFile, time.Now(), time.Now())
		AssertPathError(t, err).Op("chtimes").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("ChtimesPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Chtimes")
		pts.Test(t, func(path string) error {
			return vfs.Chtimes(path, time.Now(), time.Now())
		})
	})
}

// TestClone tests Clone function.
func (ts *Suite) TestClone(t *testing.T, _ string) {
	vfs := ts.vfsTest

	if vfsClonable, ok := vfs.(avfs.Cloner); ok {
		vfsCloned := vfsClonable.Clone()

		if _, ok := vfsCloned.(avfs.Cloner); !ok {
			t.Errorf("Clone : want cloned vfs to be of type VFS, got type %v", reflect.TypeOf(vfsCloned))
		}
	}
}

// TestCreate tests Create function.
func (ts *Suite) TestCreate(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.Create(testDir)
		AssertPathError(t, err).Op("open").Path(testDir).ErrPermDenied().Test()

		return
	}

	t.Run("CreatePerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Create")
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
func (ts *Suite) TestCreateTemp(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.CreateTemp(testDir, "CreateTemp")
		AssertPathError(t, err).Op("createtemp").ErrPermDenied().Test()

		return
	}

	t.Run("CreateTempEmptyDir", func(t *testing.T) {
		f, err := vfs.CreateTemp("", "CreateTemp")
		RequireNoError(t, err, "CreateTemp")

		wantDir := vfs.TempDir()
		dir := vfs.Dir(f.Name())

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}

		err = f.Close()
		RequireNoError(t, err, "Close")

		err = vfs.Remove(f.Name())
		RequireNoError(t, err, "Remove")
	})

	t.Run("CreateTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.CreateTemp("", badPattern)
		AssertPathError(t, err).Op("createtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator).Test()
	})

	t.Run("CreateTempOnFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		_, err := vfs.CreateTemp(existingFile, "CreateTempOnFile")
		AssertPathError(t, err).Op("open").
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})
}

// TestUser tests User function.
func (ts *Suite) TestUser(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
func (ts *Suite) TestEvalSymlink(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.EvalSymlinks(testDir)
		AssertPathError(t, err).OpLstat().Path(testDir).ErrPermDenied().Test()

		return
	}

	_ = ts.createSampleDirs(t, testDir)
	_ = ts.createSampleFiles(t, testDir)
	_ = ts.createSampleSymlinks(t, testDir)

	t.Run("EvalSymlink", func(t *testing.T) {
		symlinks := ts.sampleSymlinksEval(testDir)
		for _, sl := range symlinks {
			wantPath := sl.OldPath
			slPath := sl.NewPath

			gotPath, err := vfs.EvalSymlinks(sl.NewPath)
			if !AssertNoError(t, err, "EvalSymlinks %s", sl.NewPath) {
				continue
			}

			if wantPath != gotPath {
				t.Errorf("EvalSymlinks %s : want Path to be %s, got %s", slPath, wantPath, gotPath)
			}
		}
	})
}

// TestTempDir tests TempDir function.
func (ts *Suite) TestTempDir(t *testing.T, _ string) {
	vfs := ts.vfsTest
	tmpDir := vfs.TempDir()

	if vfs.OSType() == avfs.OsWindows {
		const wantRe = `(?i)^(c:\\windows\\temp|c:\\Users\\[^\\]+\\AppData\\Local\\Temp)$`

		re := regexp.MustCompile(wantRe)
		if !re.MatchString(tmpDir) {
			t.Errorf("TempDir : want temp dir to match '%s', got %s", wantRe, tmpDir)
		}
	} else {
		wantTmpDir := "/tmp"
		if tmpDir != wantTmpDir {
			t.Errorf("TempDir : want temp dir to be %s, got %s", wantTmpDir, tmpDir)
		}
	}
}

// TestLchown tests Lchown function.
func (ts *Suite) TestLchown(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	idm := vfs.Idm()

	if !ts.canTestPerm && vfs.HasFeature(avfs.FeatRealFS) ||
		vfs.HasFeature(avfs.FeatReadOnly) || avfs.CurrentOSType() == avfs.OsWindows {
		err := vfs.Lchown(testDir, 0, 0)

		AssertPathError(t, err).Op("lchown").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrOpNotPermitted).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNotSupported).Test()

		return
	}

	t.Run("LChownNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Lchown(nonExistingFile, 0, 0)
		AssertPathError(t, err).Op("lchown").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNotSupported).Test()
	})

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		wantUid, wantGid := 42, 42

		err := vfs.Lchown(testDir, wantUid, wantGid)
		RequireNoError(t, err, "Lchown %s", testDir)

		fst, err := vfs.Stat(testDir)
		RequireNoError(t, err, "Stat %s", testDir)

		sst := vfs.ToSysStat(fst)

		uid, gid := sst.Uid(), sst.Gid()
		if uid != wantUid || gid != wantGid {
			t.Errorf("Stat %s : want Uid=%d, Gid=%d, got Uid=%d, Gid=%d",
				testDir, wantUid, wantGid, uid, gid)
		}

		return
	}

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	symlinks := ts.createSampleSymlinks(t, testDir)
	uis := UserInfos()

	t.Run("LchownDir", func(t *testing.T) {
		for _, ui := range uis {
			wantName := ui.Name

			u, err := idm.LookupUser(wantName)
			RequireNoError(t, err, "LookupUser %s", wantName)

			for _, dir := range dirs {
				err := vfs.Lchown(dir.Path, u.Uid(), u.Gid())
				RequireNoError(t, err, "Lchown %s", dir.Path)

				fst, err := vfs.Lstat(dir.Path)
				if !AssertNoError(t, err, "Lstat %s", dir.Path) {
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
			RequireNoError(t, err, "LookupUser %s", wantName)

			for _, file := range files {
				err := vfs.Lchown(file.Path, u.Uid(), u.Gid())
				RequireNoError(t, err, "Lchown %s", file.Path)

				fst, err := vfs.Lstat(file.Path)
				if !AssertNoError(t, err, "Lstat %s", file.Path) {
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
			RequireNoError(t, err, "LookupUser %s", wantName)

			for _, sl := range symlinks {
				err := vfs.Lchown(sl.NewPath, u.Uid(), u.Gid())
				RequireNoError(t, err, "Lchown %s", sl.NewPath)

				fst, err := vfs.Lstat(sl.NewPath)
				if !AssertNoError(t, err, "Lstat %s", sl.NewPath) {
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

	t.Run("LchownPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Lchown")
		pts.Test(t, func(path string) error {
			return vfs.Lchown(path, 0, 0)
		})
	})
}

// TestLink tests Link function.
func (ts *Suite) TestLink(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if !vfs.HasFeature(avfs.FeatHardlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Link(testDir, testDir)
		AssertLinkError(t, err).Op("link").Old(testDir).New(testDir).ErrPermDenied().Test()

		return
	}

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	pathLinks := ts.existingDir(t, testDir)

	t.Run("LinkCreate", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(file.Path, newPath)
			RequireNoError(t, err, "Link "+file.Path+" "+newPath)

			newContent, err := vfs.ReadFile(newPath)
			RequireNoError(t, err, "ReadFile "+newPath)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkExisting", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Link(file.Path, newPath)
			AssertLinkError(t, err).Op("link").Old(file.Path).New(newPath).
				OSType(avfs.OsLinux).Err(avfs.ErrFileExists).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinAlreadyExists).Test()
		}
	})

	t.Run("LinkRemove", func(t *testing.T) {
		for _, file := range files {
			newPath := vfs.Join(pathLinks, vfs.Base(file.Path))

			err := vfs.Remove(file.Path)
			RequireNoError(t, err, "Remove %s", file.Path)

			newContent, err := vfs.ReadFile(newPath)
			RequireNoError(t, err, "ReadFile %s", file.Path)

			if !bytes.Equal(file.Content, newContent) {
				t.Errorf("ReadFile %s : want content to be %s, got %s", newPath, file.Content, newContent)
			}
		}
	})

	t.Run("LinkErrorDir", func(t *testing.T) {
		for _, dir := range dirs {
			newPath := vfs.Join(testDir, defaultDir)

			err := vfs.Link(dir.Path, newPath)
			AssertLinkError(t, err).Op("link").Old(dir.Path).New(newPath).
				OSType(avfs.OsLinux).Err(avfs.ErrOpNotPermitted).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
		}
	})

	t.Run("LinkErrorFile", func(t *testing.T) {
		for _, file := range files {
			InvalidPath := vfs.Join(file.Path, defaultNonExisting)
			NewInvalidPath := vfs.Join(pathLinks, defaultFile)

			err := vfs.Link(InvalidPath, NewInvalidPath)
			AssertLinkError(t, err).Op("link").Old(InvalidPath).New(NewInvalidPath).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
		}
	})

	t.Run("LinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)
		existingFile := ts.emptyFile(t, testDir)

		err := vfs.Link(nonExistingFile, existingFile)
		AssertLinkError(t, err).Op("link").Old(nonExistingFile).New(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("LinkPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "LinkNew")

		oldFile := vfs.Join(pts.permDir, "OldFile")
		ts.createFile(t, oldFile, avfs.DefaultFilePerm)

		pts.Test(t, func(path string) error {
			newFile := vfs.Join(path, "newFile")

			return vfs.Link(oldFile, newFile)
		})
	})
}

// TestLstat tests Lstat function.
func (ts *Suite) TestLstat(t *testing.T, testDir string) {
	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	ts.createSampleSymlinks(t, testDir)

	vfs := ts.vfsTest

	t.Run("LstatDir", func(t *testing.T) {
		for _, dir := range dirs {
			info, err := vfs.Lstat(dir.Path)
			if !AssertNoError(t, err, "Lstat %s", dir.Path) {
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
			if !AssertNoError(t, err, "Lstat %s", file.Path) {
				continue
			}

			if info.Name() != vfs.Base(file.Path) {
				t.Errorf("Lstat %s : want name to be %s, got %s", file.Path, vfs.Base(file.Path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = avfs.DefaultFilePerm
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
		for _, sl := range ts.sampleSymlinksEval(testDir) {
			info, err := vfs.Lstat(sl.NewPath)
			if !AssertNoError(t, err, "Lstat %s", sl.NewPath) {
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
		nonExistingFile := ts.nonExistingFile(t, testDir)

		_, err := vfs.Lstat(nonExistingFile)
		AssertPathError(t, err).OpLstat().Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("LStatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(files[0].Path, "subDirOnFile")

		_, err := vfs.Lstat(subDirOnFile)
		AssertPathError(t, err).OpLstat().Path(subDirOnFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})
}

// TestMkdir tests Mkdir function.
func (ts *Suite) TestMkdir(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Mkdir(testDir, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("mkdir").Path(testDir).ErrPermDenied().Test()

		return
	}

	dirs := ts.sampleDirs(testDir)

	t.Run("Mkdir", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.Mkdir(dir.Path, dir.Mode)
			AssertNoError(t, err, "Mkdir %", dir.Path)

			fi, err := vfs.Stat(dir.Path)
			if !AssertNoError(t, err, "stat %s", dir.Path) {
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
				RequireNoError(t, err, "Stat %s", curPath)

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
			AssertPathError(t, err).Op("mkdir").Path(path).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
		}
	})

	t.Run("MkdirEmptyName", func(t *testing.T) {
		err := vfs.Mkdir("", avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("mkdir").Path("").
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("MkdirOnFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.Mkdir(subDirOnFile, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("mkdir").Path(subDirOnFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("MkdirPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Mkdir")
		pts.Test(t, func(path string) error {
			newDir := vfs.Join(path, "newDir")

			return vfs.Mkdir(newDir, avfs.DefaultDirPerm)
		})
	})
}

// TestMkdirAll tests MkdirAll function.
func (ts *Suite) TestMkdirAll(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.MkdirAll(testDir, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("mkdir").Path(testDir).ErrPermDenied().Test()

		return
	}

	existingFile := ts.emptyFile(t, testDir)
	dirs := ts.sampleDirsAll(testDir)

	t.Run("MkdirAll", func(t *testing.T) {
		for _, dir := range dirs {
			err := vfs.MkdirAll(dir.Path, dir.Mode)
			RequireNoError(t, err, "MkdirAll %s", dir.Path)

			fi, err := vfs.Stat(dir.Path)
			RequireNoError(t, err, "Stat %s", dir.Path)

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
				RequireNoError(t, err, "Stat %s", curPath)

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
			RequireNoError(t, err, "MkdirAll %s", dir.Path)
		}
	})

	t.Run("MkdirAllOnFile", func(t *testing.T) {
		err := vfs.MkdirAll(existingFile, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("mkdir").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("MkdirAllSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(existingFile, defaultDir)

		err := vfs.MkdirAll(subDirOnFile, avfs.DefaultDirPerm)
		AssertPathError(t, err).Op("mkdir").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("MkdirAllPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "MkdirAll")
		pts.Test(t, func(path string) error {
			newDir := vfs.Join(path, "newDir")

			return vfs.MkdirAll(newDir, avfs.DefaultDirPerm)
		})
	})
}

// TestMkdirTemp tests MkdirTemp function.
func (ts *Suite) TestMkdirTemp(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.MkdirTemp(testDir, "MkdirTemp")
		AssertPathError(t, err).Op("mkdirtemp").ErrPermDenied().Test()

		return
	}

	t.Run("MkdirTempEmptyDir", func(t *testing.T) {
		tmpDir, err := vfs.MkdirTemp("", "MkdirTemp")
		RequireNoError(t, err, "MkdirTemp")

		wantDir := vfs.TempDir()
		dir := vfs.Dir(tmpDir)

		if dir != wantDir {
			t.Errorf("want directory to be %s, got %s", wantDir, dir)
		}

		err = vfs.Remove(tmpDir)
		RequireNoError(t, err, "Remove %s", tmpDir)
	})

	t.Run("MkdirTempBadPattern", func(t *testing.T) {
		badPattern := vfs.TempDir()

		_, err := vfs.MkdirTemp("", badPattern)
		AssertPathError(t, err).Op("mkdirtemp").Path(badPattern).Err(avfs.ErrPatternHasSeparator).Test()
	})

	t.Run("MkdirTempOnFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		_, err := vfs.MkdirTemp(existingFile, "MkdirTempOnFile")
		AssertPathError(t, err).Op("mkdir").
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})
}

// TestName tests Name function.
func (ts *Suite) TestName(t *testing.T, _ string) {
	vfs := ts.vfsSetup

	if vfs.Name() != "" {
		t.Errorf("want name to be empty, got %s", vfs.Name())
	}
}

// TestOpen tests Open function.
func (ts *Suite) TestOpen(t *testing.T, testDir string) {
	data := []byte("AAABBBCCCDDD")
	existingFile := ts.existingFile(t, testDir, data)
	existingDir := ts.existingDir(t, testDir)

	vfs := ts.vfsTest

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		f, err := vfs.OpenFile(existingFile, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		defer f.Close()

		gotData, err := io.ReadAll(f)
		RequireNoError(t, err, "ReadAll %s", existingFile)

		if !bytes.Equal(gotData, data) {
			t.Errorf("ReadAll : want error data to be %v, got %v", data, gotData)
		}
	})

	t.Run("OpenFileDirReadOnly", func(t *testing.T) {
		f, err := vfs.OpenFile(existingDir, os.O_RDONLY, 0)
		RequireNoError(t, err, "Open %s", existingDir)

		defer f.Close()

		dirs, err := f.ReadDir(-1)
		RequireNoError(t, err, "ReadDir %s", existingDir)

		if len(dirs) != 0 {
			t.Errorf("ReadDir : want number of directories to be 0, got %d", len(dirs))
		}
	})

	t.Run("OpenNonExistingFile", func(t *testing.T) {
		fileName := ts.nonExistingFile(t, testDir)

		_, err := vfs.OpenFile(fileName, os.O_RDONLY, 0)
		AssertPathError(t, err).Op("open").Path(fileName).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})
}

// TestOpenFileWrite tests OpenFile function for write.
func (ts *Suite) TestOpenFileWrite(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		_, err := vfs.OpenFile(testDir, os.O_WRONLY, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(testDir).ErrPermDenied().Test()

		return
	}

	data := []byte("AAABBBCCCDDD")
	defaultData := []byte("Default data")
	buf3 := make([]byte, 3)

	t.Run("OpenFileWriteOnly", func(t *testing.T) {
		existingFile := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(existingFile, os.O_WRONLY, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		defer f.Close()

		n, err := f.Write(defaultData)
		RequireNoError(t, err, "Write %s", existingFile)

		if n != len(defaultData) {
			t.Errorf("Write : want bytes written to be %d, got %d", len(defaultData), n)
		}

		n, err = f.Read(buf3)
		AssertPathError(t, err).Op("read").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

		if n != 0 {
			t.Errorf("Read : want bytes written to be 0, got %d", n)
		}

		n, err = f.ReadAt(buf3, 3)
		AssertPathError(t, err).Op("read").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

		if n != 0 {
			t.Errorf("ReadAt : want bytes read to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		RequireNoError(t, err, "Chmod %s", existingFile)

		if vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
			u := vfs.User()
			err = f.Chown(u.Uid(), u.Gid())
			RequireNoError(t, err, "Chown%s", existingFile)
		}

		fst, err := f.Stat()
		RequireNoError(t, err, "Stat %s", existingFile)

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		RequireNoError(t, err, "Truncate %s", existingFile)

		err = f.Sync()
		RequireNoError(t, err, "Sync %s", existingFile)
	})

	t.Run("OpenFileAppend", func(t *testing.T) {
		existingFile := ts.existingFile(t, testDir, data)
		appendData := []byte("appendData")

		f, err := vfs.OpenFile(existingFile, os.O_WRONLY|os.O_APPEND, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		n, err := f.Write(appendData)
		RequireNoError(t, err, "Write %s", existingFile)

		if n != len(appendData) {
			t.Errorf("Write : want error to be %d, got %d", len(defaultData), n)
		}

		_ = f.Close()

		gotContent, err := vfs.ReadFile(existingFile)
		RequireNoError(t, err, "ReadFile %s", existingFile)

		wantContent := append(data, appendData...)
		if !bytes.Equal(wantContent, gotContent) {
			t.Errorf("ReadAll : want content to be %s, got %s", wantContent, gotContent)
		}
	})

	t.Run("OpenFileReadOnly", func(t *testing.T) {
		existingFile := ts.existingFile(t, testDir, data)

		f, err := vfs.OpenFile(existingFile, os.O_RDONLY, 0)
		RequireNoError(t, err, "OpenFile %s", existingFile)

		defer f.Close()

		n, err := f.Write(defaultData)
		AssertPathError(t, err).Op("write").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

		if n != 0 {
			t.Errorf("Write : want bytes written to be 0, got %d", n)
		}

		n, err = f.WriteAt(defaultData, 3)
		AssertPathError(t, err).Op("write").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrBadFileDesc).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()

		if n != 0 {
			t.Errorf("WriteAt : want bytes written to be 0, got %d", n)
		}

		err = f.Chmod(0o777)
		RequireNoError(t, err, "Chmod %s", existingFile)

		if vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
			u := vfs.User()
			err = f.Chown(u.Uid(), u.Gid())
			RequireNoError(t, err, "Chown %s", existingFile)
		}

		fst, err := f.Stat()
		RequireNoError(t, err, "Stat %s", existingFile)

		wantName := vfs.Base(f.Name())
		if wantName != fst.Name() {
			t.Errorf("Stat : want name to be %s, got %s", wantName, fst.Name())
		}

		err = f.Truncate(0)
		AssertPathError(t, err).Op("truncate").Path(existingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("OpenFileWriteOnDir", func(t *testing.T) {
		existingDir := ts.existingDir(t, testDir)

		f, err := vfs.OpenFile(existingDir, os.O_WRONLY, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(existingDir).
			OSType(avfs.OsLinux).Err(avfs.ErrIsADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinIsADirectory).Test()

		if !reflect.ValueOf(f).IsNil() {
			t.Errorf("OpenFile : want file to be nil, got %v", f)
		}
	})

	t.Run("OpenFileExcl", func(t *testing.T) {
		fileExcl := vfs.Join(testDir, "fileExcl")

		f, err := vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		RequireNoError(t, err, "OpenFile %s", fileExcl)

		f.Close()

		_, err = vfs.OpenFile(fileExcl, os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(fileExcl).
			OSType(avfs.OsLinux).Err(avfs.ErrFileExists).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileExists).Test()
	})

	t.Run("OpenFileNonExistingPath", func(t *testing.T) {
		nonExistingPath := vfs.Join(testDir, "non/existing/path")

		_, err := vfs.OpenFile(nonExistingPath, os.O_CREATE, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(nonExistingPath).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})

	t.Run("OpenFileInvalidSymlink", func(t *testing.T) {
		if !vfs.HasFeature(avfs.FeatSymlink) {
			return
		}

		nonExistingFile := ts.nonExistingFile(t, testDir)
		InvalidSymlink := vfs.Join(testDir, "InvalidSymlink")

		err := vfs.Symlink(nonExistingFile, InvalidSymlink)
		RequireNoError(t, err, "Invalid Symlink %s %s", nonExistingFile, InvalidSymlink)

		_, err = vfs.OpenFile(InvalidSymlink, os.O_RDONLY, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(InvalidSymlink).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("OpenFileDirPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "OpenFileDir")
		pts.Test(t, func(path string) error {
			f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
			if err != nil {
				return err
			}

			f.Close()

			return nil
		})
	})

	t.Run("OpenFileReadPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTestsWithOptions(t, testDir, "OpenFileRead", &PermOptions{CreateFiles: true})
		pts.Test(t, func(path string) error {
			f, err := vfs.OpenFile(path, os.O_RDONLY, 0)
			if err != nil {
				return err
			}

			f.Close()

			return nil
		})
	})

	t.Run("OpenFileWritePerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTestsWithOptions(t, testDir, "OpenFileWrite", &PermOptions{CreateFiles: true})
		pts.Test(t, func(path string) error {
			f, err := vfs.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				return err
			}

			f.Close()

			return nil
		})
	})
}

// TestPathSeparator tests PathSeparator function.
func (ts *Suite) TestPathSeparator(t *testing.T, _ string) {
	vfs := ts.vfsTest

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
func (ts *Suite) TestReadDir(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	rndTree := ts.randomDir(t, testDir)
	dirs := rndTree.Dirs()
	files := rndTree.Files()
	symLinks := rndTree.SymLinks()

	wantDirs := len(dirs)
	wantFiles := len(files)
	wantSymlinks := len(symLinks)

	existingFile := vfs.Join(testDir, files[0].Name)

	t.Run("ReadDirAll", func(t *testing.T) {
		dirEntries, err := vfs.ReadDir(testDir)
		RequireNoError(t, err, "ReadDir %s", testDir)

		var gotDirs, gotFiles, gotSymlinks int

		for _, dirEntry := range dirEntries {
			_, err = dirEntry.Info()
			if !AssertNoError(t, err, "Info") {
				continue
			}

			switch {
			case dirEntry.IsDir():
				gotDirs++
			case dirEntry.Type()&fs.ModeSymlink != 0:
				gotSymlinks++
			default:
				gotFiles++
			}
		}

		if gotDirs != wantDirs {
			t.Errorf("ReadDir : want number of dirs to be %d, got %d", wantDirs, gotDirs)
		}

		if gotFiles != wantFiles {
			t.Errorf("ReadDir : want number of files to be %d, got %d", wantFiles, gotFiles)
		}

		if gotSymlinks != wantSymlinks {
			t.Errorf("ReadDir : want number of symbolic links to be %d, got %d", wantSymlinks, gotSymlinks)
		}
	})

	t.Run("ReadDirEmptySubDirs", func(t *testing.T) {
		for _, dir := range dirs {
			path := vfs.Join(testDir, dir.Name)

			infos, err := vfs.ReadDir(path)
			if !AssertNoError(t, err, "ReadDir %s", dir) {
				continue
			}

			l := len(infos)
			if l != 0 {
				t.Errorf("ReadDir %s : want count to be O, got %d", path, l)
			}
		}
	})

	t.Run("ReadDirExistingFile", func(t *testing.T) {
		_, err := vfs.ReadDir(existingFile)
		AssertPathError(t, err).Path(existingFile).
			OSType(avfs.OsLinux).Op("readdirent").Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Op("readdir").Err(avfs.ErrWinPathNotFound).Test()
	})
}

// TestReadFile tests ReadFile function.
func (ts *Suite) TestReadFile(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	t.Run("ReadFile", func(t *testing.T) {
		data := []byte("AAABBBCCCDDD")
		path := ts.existingFile(t, testDir, data)

		rb, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})

	t.Run("ReadFileNotExisting", func(t *testing.T) {
		path := ts.nonExistingFile(t, testDir)

		rb, err := vfs.ReadFile(path)
		AssertPathError(t, err).Op("open").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

		if len(rb) != 0 {
			t.Errorf("ReadFile : want read bytes to be 0, got %d", len(rb))
		}
	})
}

// TestReadlink tests Readlink function.
func (ts *Suite) TestReadlink(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) {
		_, err := vfs.Readlink(testDir)

		AssertPathError(t, err).Op("readlink").Path(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrPermDenied).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNotReparsePoint).Test()

		return
	}

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	symlinks := ts.createSampleSymlinks(t, testDir)

	vfs = ts.vfsTest

	t.Run("ReadlinkLink", func(t *testing.T) {
		for _, sl := range symlinks {
			gotPath, err := vfs.Readlink(sl.NewPath)
			RequireNoError(t, err, "Readlink %s", sl.NewPath)

			if sl.OldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", sl.NewPath, sl.OldPath, gotPath)
			}
		}
	})

	t.Run("ReadlinkDir", func(t *testing.T) {
		for _, dir := range dirs {
			_, err := vfs.Readlink(dir.Path)
			AssertPathError(t, err).Op("readlink").Path(dir.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinNotReparsePoint).Test()
		}
	})

	t.Run("ReadlinkFile", func(t *testing.T) {
		for _, file := range files {
			_, err := vfs.Readlink(file.Path)
			AssertPathError(t, err).Op("readlink").Path(file.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinNotReparsePoint).Test()
		}
	})

	t.Run("ReadLinkNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		_, err := vfs.Readlink(nonExistingFile)
		AssertPathError(t, err).Op("readlink").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})
}

// TestRemove tests Remove function.
func (ts *Suite) TestRemove(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Remove(testDir)
		AssertPathError(t, err).Op("remove").Path(testDir).ErrPermDenied().Test()

		return
	}

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	symlinks := ts.createSampleSymlinks(t, testDir)

	t.Run("RemoveFile", func(t *testing.T) {
		for _, file := range files {
			_, err := vfs.Stat(file.Path)
			if !AssertNoError(t, err, "Stat %s", file.Path) {
				continue
			}

			err = vfs.Remove(file.Path)
			RequireNoError(t, err, "Remove %s", file.Path)

			_, err = vfs.Stat(file.Path)
			AssertPathError(t, err).OpStat().Path(file.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		for _, dir := range dirs {
			infos, err := vfs.ReadDir(dir.Path)
			if !AssertNoError(t, err, "ReadDir %s", dir.Path) {
				continue
			}

			err = vfs.Remove(dir.Path)

			isLeaf := len(infos) == 0
			if isLeaf {
				RequireNoError(t, err, "Remove %s", dir.Path)

				_, err = vfs.Stat(dir.Path)
				AssertPathError(t, err).OpStat().Path(dir.Path).
					OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
					OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

				continue
			}

			AssertPathError(t, err).Op("remove").Path(dir.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrDirNotEmpty).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinDirNotEmpty).Test()

			_, err = vfs.Stat(dir.Path)
			RequireNoError(t, err, "Stat %s", dir.Path)
		}
	})

	t.Run("RemoveSymlinks", func(t *testing.T) {
		for _, sl := range symlinks {
			err := vfs.Remove(sl.NewPath)
			RequireNoError(t, err, "Remove %s", sl.NewPath)

			_, err = vfs.Stat(sl.NewPath)
			AssertPathError(t, err).OpStat().Path(sl.NewPath).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
		}
	})

	t.Run("RemoveNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Remove(nonExistingFile)
		AssertPathError(t, err).Op("remove").Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("RemovePerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Remove")
		pts.Test(t, func(path string) error {
			return vfs.Remove(path)
		})
	})
}

// TestRemoveAll tests RemoveAll function.
func (ts *Suite) TestRemoveAll(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.RemoveAll(testDir)
		AssertPathError(t, err).Op("removeall").Path(testDir).ErrPermDenied().Test()

		return
	}

	baseDir := vfs.Join(testDir, "RemoveAll")
	dirs := ts.createSampleDirs(t, baseDir)
	files := ts.createSampleFiles(t, baseDir)
	symlinks := ts.createSampleSymlinks(t, baseDir)

	t.Run("RemoveAll", func(t *testing.T) {
		err := vfs.RemoveAll(baseDir)
		RequireNoError(t, err, "RemoveAll %s", baseDir)

		parentDir := vfs.Dir(baseDir)
		_, err = vfs.Stat(parentDir)
		RequireNoError(t, err, "Stat %s", parentDir)

		for _, dir := range dirs {
			_, err = vfs.Stat(dir.Path)
			AssertPathError(t, err).OpStat().Path(dir.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
		}

		for _, file := range files {
			_, err = vfs.Stat(file.Path)
			AssertPathError(t, err).OpStat().Path(file.Path).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
		}

		for _, sl := range symlinks {
			_, err = vfs.Stat(sl.NewPath)
			AssertPathError(t, err).OpStat().Path(sl.NewPath).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
		}

		_, err = vfs.Stat(baseDir)
		AssertPathError(t, err).OpStat().Path(baseDir).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("RemoveAllOneFile", func(t *testing.T) {
		existingFile := ts.emptyFile(t, testDir)

		err := vfs.RemoveAll(existingFile)
		RequireNoError(t, err, "RemoveAll %s", existingFile)
	})

	t.Run("RemoveAllPathEmpty", func(t *testing.T) {
		_ = ts.createSampleDirs(t, baseDir)

		err := vfs.Chdir(baseDir)
		RequireNoError(t, err, "Chdir %s", baseDir)

		err = vfs.RemoveAll("")
		RequireNoError(t, err, "RemoveAll ''")

		// Verify that nothing was removed.
		for _, dir := range dirs {
			_, err = vfs.Stat(dir.Path)
			RequireNoError(t, err, "Stat %s", dir.Path)
		}
	})

	t.Run("RemoveAllNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.RemoveAll(nonExistingFile)
		RequireNoError(t, err, "RemoveAll %s", nonExistingFile)
	})

	t.Run("RemoveAllPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTestsWithOptions(t, testDir, "RemoveAll", &PermOptions{IgnoreOp: true})
		pts.Test(t, func(path string) error {
			return vfs.RemoveAll(path)
		})
	})
}

// TestRename tests Rename function.
func (ts *Suite) TestRename(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Rename(testDir, testDir)
		AssertLinkError(t, err).Op("rename").Old(testDir).New(testDir).ErrPermDenied().Test()

		return
	}

	data := []byte("data")

	t.Run("RenameDir", func(t *testing.T) {
		dirs := ts.createSampleDirs(t, testDir)

		for i := len(dirs) - 1; i >= 0; i-- {
			oldPath := dirs[i].Path
			newPath := oldPath + "New"

			err := vfs.Rename(oldPath, newPath)
			RequireNoError(t, err, "Rename %s %s", oldPath, newPath)

			_, err = vfs.Stat(oldPath)
			AssertPathError(t, err).OpStat().Path(oldPath).
				OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
				OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

			_, err = vfs.Stat(newPath)
			RequireNoError(t, err, "Stat %s", newPath)
		}
	})

	t.Run("RenameFile", func(t *testing.T) {
		_ = ts.createSampleDirs(t, testDir)
		files := ts.createSampleFiles(t, testDir)

		for _, file := range files {
			newPath := vfs.Join(testDir, vfs.Base(file.Path))

			err := vfs.Rename(file.Path, newPath)
			RequireNoError(t, err, "Rename %s %s", file.Path, newPath)

			_, err = vfs.Stat(file.Path)

			switch {
			case file.Path == newPath:
				RequireNoError(t, err, "Stat %s", file.Path)
			default:
				AssertPathError(t, err).OpStat().Path(file.Path).
					OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
					OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
			}

			_, err = vfs.Stat(newPath)
			RequireNoError(t, err, "Stat %s", newPath)
		}
	})

	t.Run("RenameNonExistingFile", func(t *testing.T) {
		srcNonExistingFile := vfs.Join(testDir, "srcNonExistingFile1")
		dstNonExistingFile := vfs.Join(testDir, "dstNonExistingFile2")

		err := vfs.Rename(srcNonExistingFile, dstNonExistingFile)
		AssertLinkError(t, err).Op("rename").Old(srcNonExistingFile).New(dstNonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("RenameDirToExistingDir", func(t *testing.T) {
		srcExistingDir := ts.existingDir(t, testDir)
		dstExistingDir := ts.existingDir(t, testDir)

		err := vfs.Rename(srcExistingDir, dstExistingDir)
		AssertLinkError(t, err).Op("rename").Old(srcExistingDir).New(dstExistingDir).
			OSType(avfs.OsLinux).Err(avfs.ErrFileExists).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("RenameFileToExistingFile", func(t *testing.T) {
		srcExistingFile := ts.existingFile(t, testDir, data)
		dstExistingFile := ts.emptyFile(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingFile)
		RequireNoError(t, err, "Rename %s %s", srcExistingFile, dstExistingFile)

		_, err = vfs.Stat(srcExistingFile)
		AssertPathError(t, err).OpStat().Path(srcExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

		info, err := vfs.Stat(dstExistingFile)
		RequireNoError(t, err, "Stat %s", dstExistingFile)

		if int(info.Size()) != len(data) {
			t.Errorf("Stat : want size to be %d, got %d", len(data), info.Size())
		}
	})

	t.Run("RenameFileToExistingDir", func(t *testing.T) {
		srcExistingFile := ts.existingFile(t, testDir, data)
		dstExistingDir := ts.existingDir(t, testDir)

		err := vfs.Rename(srcExistingFile, dstExistingDir)
		AssertLinkError(t, err).Op("rename").Old(srcExistingFile).New(dstExistingDir).
			OSType(avfs.OsLinux).Err(avfs.ErrFileExists).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinAccessDenied).Test()
	})

	t.Run("RenamePerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "RenameNew")
		pts.Test(t, func(path string) error {
			oldPath := vfs.Join(pts.permDir, vfs.Base(path))
			newPath := vfs.Join(path, "New")

			ts.createFile(t, oldPath, avfs.DefaultFilePerm)

			return vfs.Rename(oldPath, newPath)
		})
	})
}

// TestSameFile tests SameFile function.
func (ts *Suite) TestSameFile(t *testing.T, testDir string) {
	vfs := ts.vfsSetup

	testDir1 := vfs.Join(testDir, "dir1")
	testDir2 := vfs.Join(testDir, "dir2")

	ts.createSampleDirs(t, testDir1)
	files1 := ts.createSampleFiles(t, testDir1)

	ts.createSampleDirs(t, testDir2)

	t.Run("SameFileLink", func(t *testing.T) {
		if !vfs.HasFeature(avfs.FeatHardlink) {
			return
		}

		for _, file1 := range files1 {
			path1 := file1.Path
			path2 := vfs.Join(testDir2, strings.TrimPrefix(path1, testDir1))

			info1, err := vfs.Stat(path1)
			if !AssertNoError(t, err, "Stat %s", path1) {
				continue
			}

			err = vfs.Link(path1, path2)
			RequireNoError(t, err, "Link %s %s", path1, path2)

			info2, err := vfs.Stat(path2)
			if !AssertNoError(t, err, "Stat %s", path2) {
				continue
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			err = vfs.Remove(path2)
			RequireNoError(t, err, "Remove %s", path2)
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
			if !AssertNoError(t, err, "Stat %s", path1) {
				continue
			}

			err = vfs.Symlink(path1, path2)
			if !AssertNoError(t, err, "Symlink %s %s", path1, path2) {
				continue
			}

			info2, err := vfs.Stat(path2)
			if !AssertNoError(t, err, "Stat %s", path2) {
				continue
			}

			if !vfs.SameFile(info1, info2) {
				t.Fatalf("SameFile %s, %s : not same files\n%v\n%v", path1, path2, info1, info2)
			}

			info3, err := vfs.Lstat(path2)
			if !AssertNoError(t, err, "Lstat %s", path2) {
				continue
			}

			if vfs.SameFile(info1, info3) {
				t.Fatalf("SameFile %s, %s : not the same file\n%v\n%v", path1, path2, info1, info3)
			}

			err = vfs.Remove(path2)
			RequireNoError(t, err, "Remove %s", path2)
		}
	})
}

// TestStat tests Stat function.
func (ts *Suite) TestStat(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	dirs := ts.createSampleDirs(t, testDir)
	files := ts.createSampleFiles(t, testDir)
	_ = ts.createSampleSymlinks(t, testDir)

	t.Run("StatDir", func(t *testing.T) {
		for _, dir := range dirs {
			info, err := vfs.Stat(dir.Path)
			if !AssertNoError(t, err, "Stat %s", dir.Path) {
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
			if !AssertNoError(t, err, "Stat %s", file.Path) {
				continue
			}

			if info.Name() != vfs.Base(file.Path) {
				t.Errorf("Stat %s : want name to be %s, got %s", file.Path, vfs.Base(file.Path), info.Name())
			}

			wantMode := file.Mode &^ vfs.UMask()
			if vfs.OSType() == avfs.OsWindows {
				wantMode = avfs.DefaultFilePerm
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
		for _, sl := range ts.sampleSymlinksEval(testDir) {
			info, err := vfs.Stat(sl.NewPath)
			if !AssertNoError(t, err, "Stat %s ", sl.NewPath) {
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

			if ts.canTestPerm && wantMode != info.Mode() {
				t.Errorf("Stat %s : want mode to be %s, got %s", sl.NewPath, wantMode, info.Mode())
			}
		}
	})

	t.Run("StatNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		_, err := vfs.Stat(nonExistingFile)
		AssertPathError(t, err).OpStat().Path(nonExistingFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinFileNotFound).Test()

		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Stat %s : want errors.Is(err, fs.ErrNotExist) to be true, got false ", nonExistingFile)
		}
	})

	t.Run("StatSubDirOnFile", func(t *testing.T) {
		subDirOnFile := vfs.Join(files[0].Path, "subDirOnFile")

		_, err := vfs.Stat(subDirOnFile)
		AssertPathError(t, err).OpStat().Path(subDirOnFile).
			OSType(avfs.OsLinux).Err(avfs.ErrNotADirectory).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPathNotFound).Test()
	})
}

// TestSymlink tests Symlink function.
func (ts *Suite) TestSymlink(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if !vfs.HasFeature(avfs.FeatSymlink) || vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Symlink(testDir, testDir)
		AssertLinkError(t, err).Op("symlink").Old(testDir).New(testDir).
			OSType(avfs.OsLinux).Err(avfs.ErrPermDenied).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinPrivilegeNotHeld).Test()

		return
	}

	_ = ts.createSampleDirs(t, testDir)
	_ = ts.createSampleFiles(t, testDir)

	t.Run("Symlink", func(t *testing.T) {
		symlinks := ts.sampleSymlinks(testDir)
		for _, sl := range symlinks {
			err := vfs.Symlink(sl.OldPath, sl.NewPath)
			RequireNoError(t, err, "Symlink %s %s", sl.OldPath, sl.NewPath)

			gotPath, err := vfs.Readlink(sl.NewPath)
			RequireNoError(t, err, "Readlink %s", sl.NewPath)

			if sl.OldPath != gotPath {
				t.Errorf("ReadLink %s : want link to be %s, got %s", sl.NewPath, sl.OldPath, gotPath)
			}
		}
	})

	t.Run("SymlinkPerm", func(t *testing.T) {
		if !ts.canTestPerm {
			return
		}

		pts := ts.NewPermTests(t, testDir, "Symlink")
		pts.Test(t, func(path string) error {
			newName := vfs.Join(path, "Symlink")

			return vfs.Symlink(path, newName)
		})
	})
}

// TestSub tests Sub function.
func (ts *Suite) TestSub(t *testing.T, testDir string) {
	vfs, ok := ts.vfsTest.(avfs.VFS)
	if !ok {
		return
	}

	if !vfs.HasFeature(avfs.FeatSubFS) {
		_, err := vfs.Sub(testDir)
		AssertPathError(t, err).Op("sub").Path(testDir).ErrPermDenied().Test()

		return
	}

	t.Run("Sub", func(t *testing.T) {
		//		vfsSub, err := vfs.Sub(testDir)
		//		RequireNoError(t, "Sub "+testDir, err)
	})
}

// TestToSysStat tests ToSysStat function.
func (ts *Suite) TestToSysStat(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	existingFile := ts.emptyFile(t, testDir)

	fst, err := vfs.Stat(existingFile)
	RequireNoError(t, err, "Stat %s", existingFile)

	u := vfs.User()
	if vfs.HasFeature(avfs.FeatReadOnly) {
		u = ts.vfsSetup.User()
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
func (ts *Suite) TestTruncate(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.Truncate(testDir, 0)
		AssertPathError(t, err).Op("truncate").Path(testDir).ErrPermDenied().Test()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("Truncate", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

		for i := len(data); i >= 0; i-- {
			err := vfs.Truncate(path, int64(i))
			RequireNoError(t, err, "Truncate %s", path)

			d, err := vfs.ReadFile(path)
			RequireNoError(t, err, "ReadFile %s", path)

			if len(d) != i {
				t.Errorf("Truncate : want length to be %d, got %d", i, len(d))
			}
		}
	})

	t.Run("TruncateNonExistingFile", func(t *testing.T) {
		nonExistingFile := ts.nonExistingFile(t, testDir)

		err := vfs.Truncate(nonExistingFile, 0)
		AssertPathError(t, err).Path(nonExistingFile).
			OSType(avfs.OsLinux).Op("truncate").Err(avfs.ErrNoSuchFileOrDir).Test().
			OSType(avfs.OsWindows).Op("open").Err(avfs.ErrWinFileNotFound).Test()
	})

	t.Run("TruncateOnDir", func(t *testing.T) {
		err := vfs.Truncate(testDir, 0)
		AssertPathError(t, err).Path(testDir).
			OSType(avfs.OsLinux).Op("truncate").Err(avfs.ErrIsADirectory).Test().
			OSType(avfs.OsWindows).Op("open").Err(avfs.ErrWinIsADirectory).Test()
	})

	t.Run("TruncateSizeNegative", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)

		err := vfs.Truncate(path, -1)
		AssertPathError(t, err).Op("truncate").Path(path).
			OSType(avfs.OsLinux).Err(avfs.ErrInvalidArgument).Test().
			OSType(avfs.OsWindows).Err(avfs.ErrWinNegativeSeek).Test()
	})

	t.Run("TruncateSizeBiggerFileSize", func(t *testing.T) {
		path := ts.existingFile(t, testDir, data)
		newSize := len(data) * 2

		err := vfs.Truncate(path, int64(newSize))
		RequireNoError(t, err, "Truncate %s", path)

		info, err := vfs.Stat(path)
		RequireNoError(t, err, "Stat %s", path)

		if newSize != int(info.Size()) {
			t.Errorf("Stat : want size to be %d, got %d", newSize, info.Size())
		}

		gotContent, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		wantAdded := bytes.Repeat([]byte{0}, len(data))
		gotAdded := gotContent[len(data):]
		if !bytes.Equal(wantAdded, gotAdded) {
			t.Errorf("Bytes Added : want %v, got %v", wantAdded, gotAdded)
		}
	})
}

// TestUMask tests SetUMask and UMask functions.
func (ts *Suite) TestUMask(t *testing.T, _ string) {
	const (
		linuxUMask   = fs.FileMode(0o22)
		windowsUMask = fs.FileMode(0o111)
		testUMask    = fs.FileMode(0o77)
	)

	vfs := ts.vfsTest

	saveUMask := vfs.UMask()
	defer func() { _ = vfs.SetUMask(saveUMask) }()

	defaultUMask := linuxUMask
	if vfs.OSType() == avfs.OsWindows {
		defaultUMask = windowsUMask
	}

	wantedUMask := defaultUMask

	umask := vfs.UMask()
	if umask != wantedUMask {
		t.Errorf("UMask : want OS umask %o, got %o", wantedUMask, umask)
	}

	_ = vfs.SetUMask(testUMask)

	umask = vfs.UMask()
	if umask != testUMask {
		t.Errorf("UMask : want test umask %o, got %o", testUMask, umask)
	}

	_ = vfs.SetUMask(defaultUMask)

	umask = vfs.UMask()
	if umask != defaultUMask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUMask, umask)
	}
}

// TestVolume tests VolumeAdd, VolumeDelete and VolumeList functions.
func (ts *Suite) TestVolume(t *testing.T, _ string) {
	const testVolume = "Z:"

	vfs := ts.vfsTest

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
		AssertPathError(t, err).Op("VolumeAdd").Path(testVolume).Err(avfs.ErrVolumeWindows).Test()

		err = vm.VolumeDelete(testVolume)
		AssertPathError(t, err).Op("VolumeDelete").Path(testVolume).Err(avfs.ErrVolumeWindows).Test()

		return
	}

	t.Run("VolumeManage", func(t *testing.T) {
		vl := vm.VolumeList()
		if len(vl) != 1 {
			t.Errorf("VolumeList : want 1 volume, got %d = %v", len(vl), vl)
		}

		err := vm.VolumeAdd(testVolume)
		RequireNoError(t, err, "VolumeAdd %s", testVolume)

		vl = vm.VolumeList()
		if len(vl) != 2 {
			t.Errorf("VolumeList : want 2 volumes, got %d = %v", len(vl), vl)
		}

		err = vm.VolumeDelete(testVolume)
		RequireNoError(t, err, "VolumeDelete %s", testVolume)

		vl = vm.VolumeList()
		if len(vl) != 1 {
			t.Errorf("VolumeList : want 1 volume, got %d = %v", len(vl), vl)
		}
	})
}

// TestWriteFile tests WriteFile function.
func (ts *Suite) TestWriteFile(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		err := vfs.WriteFile(testDir, []byte{0}, avfs.DefaultFilePerm)
		AssertPathError(t, err).Op("open").Path(testDir).ErrPermDenied().Test()

		return
	}

	data := []byte("AAABBBCCCDDD")

	t.Run("WriteFile", func(t *testing.T) {
		path := vfs.Join(testDir, "WriteFile.txt")

		err := vfs.WriteFile(path, data, avfs.DefaultFilePerm)
		RequireNoError(t, err, "WriteFile %s", path)

		rb, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}

// TestWriteOnReadOnlyFS tests all write functions of a read only file system.
func (ts *Suite) TestWriteOnReadOnlyFS(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	existingFile := ts.emptyFile(t, testDir)

	if !vfs.HasFeature(avfs.FeatReadOnly) {
		// Skip tests if the file system is not read only
		return
	}

	t.Run("ReadOnlyFile", func(t *testing.T) {
		f, err := vfs.OpenFile(existingFile, os.O_RDONLY, 0)
		RequireNoError(t, err, "Open %s", existingFile)

		err = f.Chmod(0o777)
		AssertPathError(t, err).Op("chmod").Path(f.Name()).ErrPermDenied().Test()

		err = f.Chown(0, 0)
		AssertPathError(t, err).Op("chown").Path(f.Name()).ErrPermDenied().Test()

		err = f.Truncate(0)
		AssertPathError(t, err).Op("truncate").Path(f.Name()).ErrPermDenied().Test()

		_, err = f.Write([]byte{})
		AssertPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied().Test()

		_, err = f.WriteAt([]byte{}, 0)
		AssertPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied().Test()

		_, err = f.WriteString("")
		AssertPathError(t, err).Op("write").Path(f.Name()).ErrPermDenied().Test()
	})
}

// TestWriteString tests WriteString function.
func (ts *Suite) TestWriteString(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	if vfs.HasFeature(avfs.FeatReadOnly) {
		return
	}

	data := []byte("AAABBBCCCDDD")
	path := vfs.Join(testDir, "TestWriteString.txt")

	t.Run("WriteString", func(t *testing.T) {
		f, err := vfs.Create(path)
		RequireNoError(t, err, "Create %s", path)

		n, err := f.WriteString(string(data))
		RequireNoError(t, err, "WriteString %s", path)

		if len(data) != n {
			t.Errorf("WriteString : want written bytes to be %d, got %d", len(data), n)
		}

		f.Close()

		rb, err := vfs.ReadFile(path)
		RequireNoError(t, err, "ReadFile %s", path)

		if !bytes.Equal(rb, data) {
			t.Errorf("ReadFile : want content to be %s, got %s", data, rb)
		}
	})
}
