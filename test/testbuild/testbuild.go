//
//  Copyright 2022 The AVFS authors
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

package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/vfs/osfs"
)

// This code is used by gox to generate an executable for all operating systems (see mage/magefile.go).
// It should use all functions that depend on OS specific syscalls to make sure every system can be built.
func main() {
	vfs := osfs.New()

	tmpDir, err := vfs.MkdirTemp("", "gox")
	if err != nil {
		log.Fatal(err)
	}

	u := &dummyidm.DummyUser{}
	tmpFile := vfs.Join(tmpDir, "file")

	_, _ = vfs.Abs(tmpDir)
	_ = vfs.Base(tmpDir)
	_ = vfs.Chdir(tmpDir)
	_ = vfs.Chmod(tmpDir, avfs.DefaultDirPerm)
	_ = vfs.Chown(tmpDir, 0, 0)
	_ = vfs.Chroot(tmpDir)
	_ = vfs.Chtimes(tmpDir, time.Now(), time.Now())
	_ = vfs.Clean(tmpDir)
	_, _ = vfs.Create(tmpFile)
	_, _ = vfs.CreateTemp(tmpDir, "")
	_ = vfs.Dir(tmpDir)
	_, _ = vfs.EvalSymlinks(tmpDir)
	_ = vfs.Features()
	_ = vfs.FromSlash(tmpDir)
	_, _ = vfs.Getwd()
	_, _ = vfs.Glob("")
	_ = vfs.HasFeature(avfs.FeatSymlink)
	_ = vfs.Idm()
	_ = vfs.IsAbs(tmpDir)
	_ = vfs.IsPathSeparator('a')
	_ = vfs.Join("", "")
	_ = vfs.Lchown(tmpDir, 0, 0)
	_ = vfs.Link(tmpDir, tmpDir)
	_, _ = vfs.Lstat(tmpDir)
	_, _ = vfs.Match("", "")
	_ = vfs.Mkdir(tmpDir, avfs.DefaultDirPerm)
	_ = vfs.MkdirAll(tmpDir, avfs.DefaultDirPerm)
	_, _ = vfs.MkdirTemp(tmpDir, "")
	_ = vfs.Name()
	f, _ := vfs.Open(tmpDir)
	_, _ = vfs.OpenFile(tmpDir, os.O_RDONLY, avfs.DefaultDirPerm)
	_ = vfs.OSType()
	_ = vfs.PathSeparator()
	_, _ = vfs.ReadDir(tmpDir)
	_, _ = vfs.ReadFile(tmpDir)
	_, _ = vfs.Readlink(tmpDir)
	_, _ = vfs.Rel(tmpDir, tmpDir)
	_ = vfs.Remove(tmpDir)
	_ = vfs.RemoveAll(tmpDir)
	_ = vfs.Rename(tmpDir, tmpDir)
	_, _ = vfs.Split(tmpDir)
	info, _ := vfs.Stat(tmpFile)
	_ = vfs.SetUMask(0)
	_ = vfs.SetUser(u)
	_ = vfs.SetUserByName("")
	_, _ = vfs.Sub("")
	_ = vfs.TempDir()
	_ = vfs.ToSlash(tmpDir)
	_ = vfs.ToSysStat(info)
	_ = vfs.Truncate(tmpFile, 0)
	_ = vfs.Type()
	_ = vfs.User()
	_ = vfs.UMask()
	_ = vfs.WalkDir(tmpDir, nil)
	_ = vfs.WriteFile(tmpFile, nil, avfs.DefaultFilePerm)

	_ = f.Chdir()
	_ = f.Chmod(0)
	_ = f.Chown(0, 0)
	_ = f.Close()
	_ = f.Fd()
	_ = f.Name()
	_, _ = f.Read(nil)
	_, _ = f.ReadAt(nil, 0)
	_, _ = f.ReadDir(0)
	_, _ = f.Readdirnames(0)
	_, _ = f.Seek(0, io.SeekStart)
	_, _ = f.Stat()
	_ = f.Sync()
	_, _ = f.Write(nil)
	_, _ = f.WriteAt(nil, 0)
	_, _ = f.WriteString("")
}
