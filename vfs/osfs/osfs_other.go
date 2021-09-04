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

//go:build !linux
// +build !linux

package osfs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OsFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return &OtherSysStat{}
}

// OtherSysStat implements SysStater interface returned by fs.FileInfo.Sys() for a non linux file system.
type OtherSysStat struct{}

// Gid returns the group id.
func (sst *OtherSysStat) Gid() int {
	return avfs.NotImplementedUser.Gid()
}

// Uid returns the user id.
func (sst *OtherSysStat) Uid() int {
	return avfs.NotImplementedUser.Uid()
}

// Nlink returns the number of hard links.
func (sst *OtherSysStat) Nlink() uint64 {
	return 1
}
