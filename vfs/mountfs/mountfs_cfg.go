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

package mountfs

import (
	"os"
	"strings"

	"github.com/avfs/avfs"
)

func New(rootFS avfs.VFS, basePath string) *MountFS {
	rootMnt := &mount{
		vfs:      rootFS,
		mntPath:  "",
		basePath: basePath,
	}

	vfs := &MountFS{
		rootFS:  rootFS,
		mounts:  make(mounts),
		rootMnt: rootMnt,
		curMnt:  rootMnt,
	}

	_ = vfs.SetFeatures(rootFS.Features() &^ (avfs.FeatSymlink | avfs.FeatIdentityMgr))
	_ = vfs.SetOSType(avfs.CurrentOSType())
	_ = vfs.SetCurDir("/")

	return vfs
}

// Mount mounts an existing file system mntVFS on mntPath.
func (vfs *MountFS) Mount(mntVFS avfs.VFS, mntPath, basePath string) error {
	const op = "mount"

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	absMntPath, _ := vfs.rootMnt.vfs.Abs(mntPath)

	_, ok := vfs.mounts[absMntPath]
	if ok {
		return &os.PathError{Op: op, Path: mntPath, Err: avfs.ErrFileExists}
	}

	absBasePath, _ := mntVFS.Abs(basePath)

	mnt := &mount{
		vfs:      mntVFS,
		mntPath:  absMntPath,
		basePath: absBasePath,
	}

	vfs.mounts[absMntPath] = mnt

	return nil
}

// Umount unmounts a mounted file system.
func (vfs *MountFS) Umount(mntPath string) error {
	const op = "umount"

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	absMntPath, _ := vfs.Abs(mntPath)

	mnt, ok := vfs.mounts[absMntPath]
	if !ok {
		return &os.PathError{Op: op, Path: mntPath, Err: avfs.ErrNoSuchFileOrDir}
	}

	mnt.vfs = nil

	delete(vfs.mounts, absMntPath)

	return nil
}

func (vfs *MountFS) String() string {
	var buf strings.Builder

	for _, mount := range vfs.mounts {
		buf.WriteString("\nMount = ")
		buf.WriteString(mount.mntPath)
		buf.WriteString(", Type = ")
		buf.WriteString(mount.vfs.Type())
		buf.WriteString(", Name = ")
		buf.WriteString(mount.vfs.Name())
	}

	return buf.String()
}

// Name returns the name of the fileSystem.
func (vfs *MountFS) Name() string {
	return vfs.name
}

// OSType returns the operating system type of the file system.
func (*MountFS) OSType() avfs.OSType {
	return avfs.OsLinux
}

// Type returns the type of the fileSystem or Identity manager.
func (*MountFS) Type() string {
	return "MountFS"
}
