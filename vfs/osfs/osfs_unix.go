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

//go:build unix

package osfs

import (
	"io/fs"
	"syscall"

	"github.com/avfs/avfs"
)

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chroot(path string) error {
	const op = "chroot"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
	}

	err := syscall.Chroot(path)
	if err != nil {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	return nil
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OsFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return &LinuxSysStat{Sys: info.Sys().(*syscall.Stat_t)} //nolint:forcetypeassert // type assertion must be checked
}

// LinuxSysStat implements SysStater interface returned by fs.FileInfo.Sys() for a Linux file system.
type LinuxSysStat struct {
	Sys *syscall.Stat_t
}

// Gid returns the group id.
func (lst *LinuxSysStat) Gid() int {
	return int(lst.Sys.Gid)
}

// Uid returns the user id.
func (lst *LinuxSysStat) Uid() int {
	return int(lst.Sys.Uid)
}

// Nlink returns the number of hard links.
func (lst *LinuxSysStat) Nlink() uint64 {
	return uint64(lst.Sys.Nlink) //nolint:unconvert // required for 32 bits systems.
}
