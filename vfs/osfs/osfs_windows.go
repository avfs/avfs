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

//go:build windows

package osfs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: avfs.ErrWinNotSupported}
}

// SetUser sets the current user of the file system to uid.
// If the current user has not root privileges avfs.errPermDenied is returned.
func SetUser(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OsFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	u := avfs.CurrentUser

	return &WindowsSysStat{gid: u.Gid(), uid: u.Gid()}
}

// User returns the current user of the OS.
func User() avfs.UserReader {
	return avfs.CurrentUser
}

// WindowsSysStat implements SysStater interface returned by fs.FileInfo.Sys() for a Windows file system.
type WindowsSysStat struct {
	gid int
	uid int
}

// Gid returns the group id.
func (wss *WindowsSysStat) Gid() int {
	return wss.gid
}

// Uid returns the user id.
func (wss *WindowsSysStat) Uid() int {
	return wss.uid
}

// Nlink returns the number of hard links.
func (wss *WindowsSysStat) Nlink() uint64 {
	return 1
}
