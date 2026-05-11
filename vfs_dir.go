//
//  Copyright 2024 The AVFS authors
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

package avfs

import (
	"crypto/sha256"
	"io/fs"
	"strconv"
)

// DirMgr is the interface that manages system directories.
type DirMgr interface {
	// CurDir returns the current directory.
	CurDir() string

	// SetCurDir sets the current directory.
	SetCurDir(curDir string) error
}

// DirFn provides system directories functions to a file system.
type DirFn struct {
	curDir  string // curDir is the current directory.
	tempDir string // tempDir is the temporary directory.
}

// CurDir returns the current directory.
func (df *DirFn) CurDir() string {
	return df.curDir
}

// SetCurDir sets the current directory.
func (df *DirFn) SetCurDir(curDir string) error {
	df.curDir = curDir

	return nil
}

// SetTempDir sets the temporary directory.
func (df *DirFn) SetTempDir(tempDir string) error {
	df.tempDir = tempDir

	return nil
}

// TempDir returns the default directory to use for temporary files.
//
// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
// On Windows, it uses GetTempPath, returning the first non-empty
// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
// On Plan 9, it returns /tmp.
//
// The directory is neither guaranteed to exist nor have accessible
// permissions.
func (df *DirFn) TempDir() string {
	return df.tempDir
}

// HomeDir returns the home directory of the file system.
func homeDir(ost OSType) string {
	switch ost {
	case OsWindows:
		return `\Users`
	case OsDarwin:
		return "/Users"
	default:
		return "/home"
	}
}

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func HomeDirUser[T VFSBase](vfs T, basePath string, u UserReader) string {
	var dir string

	switch vfs.OSType() {
	case OsWindows:
		if basePath == "" {
			basePath = DefaultVolume
		}

		dir = homeDir(OsDarwin) + "/" + u.Name()

	case OsDarwin:
		dir = homeDir(OsDarwin) + "/" + u.Name()
	default:
		if u.Name() == AdminUserName(OsLinux) {
			dir = "/root"
		} else {
			dir = homeDir(OsLinux) + "/" + u.Name()
		}
	}

	dir = Join(vfs, basePath, dir)

	return dir
}

// HomeDirPerm return the default permission for home directories.
func HomeDirPerm() fs.FileMode {
	return 0o755
}

// MkHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func MkHomeDir[T VFSBase](vfs T, basePath string, u UserReader) (string, error) {
	userDir := HomeDirUser(vfs, basePath, u)

	err := vfs.Mkdir(userDir, HomeDirPerm())
	if err != nil {
		return userDir, err
	}

	switch vfs.OSType() {
	case OsWindows:
		err = vfs.MkdirAll(TempDirUser(vfs, basePath, u), DefaultDirPerm)
	default:
		err = vfs.Chown(userDir, u.Uid(), u.Gid())
	}

	if err != nil {
		return userDir, err
	}

	return userDir, nil
}

// MkSystemDirs creates the system directories of a file system.
func MkSystemDirs[T VFSBase](vfs T, dirs []DirInfo) error {
	for _, dir := range dirs {
		err := vfs.MkdirAll(dir.Path, dir.Perm)
		if err != nil {
			return err
		}

		switch vfs.OSType() {
		case OsWindows:

		default:
			err = vfs.Chmod(dir.Path, dir.Perm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SystemDirs returns an array of system directories always present in the file system.
func SystemDirs[T VFSBase](vfs T, basePath string) []DirInfo {
	var dis []DirInfo

	switch vfs.OSType() {
	case OsWindows:
		if basePath == "" {
			basePath = DefaultVolume
		}

		dis = []DirInfo{
			{Path: homeDir(OsWindows), Perm: DefaultDirPerm},
			{Path: tempDirUserWindows(AdminUserName(OsWindows)), Perm: DefaultDirPerm},
			{Path: tempDirUserWindows(DefaultName), Perm: DefaultDirPerm},
			{Path: `\Windows`, Perm: DefaultDirPerm},
		}

	case OsDarwin:
		dis = []DirInfo{
			{Path: homeDir(OsDarwin), Perm: HomeDirPerm()},
			{Path: tempDirUserDarwin(vfs.User()), Perm: 0o777},
		}

	default:
		dis = []DirInfo{
			{Path: homeDir(OsLinux), Perm: HomeDirPerm()},
			{Path: "/root", Perm: 0o700},
			{Path: tempDirUserLinux(), Perm: 0o777},
		}
	}

	for i, di := range dis {
		dis[i].Path = Join(vfs, basePath, di.Path)
	}

	return dis
}

// TempDirUser returns the default directory to use for temporary files with for a specific user.
func TempDirUser[T VFSBase](vfs T, basePath string, u UserReader) string {
	var dir string

	switch vfs.OSType() {
	case OsWindows:
		if basePath == "" {
			basePath = DefaultVolume
		}

		dir = tempDirUserWindows(u.Name())

	case OsDarwin:
		dir = tempDirUserDarwin(u)

	default:
		dir = tempDirUserLinux()
	}

	dir = Join(vfs, basePath, dir)

	return dir
}

// tempDirUserDarwin returns the default directory to use for temporary files with for a specific macOS user.
func tempDirUserDarwin(u UserReader) string {
	if u.IsAdmin() {
		return "/tmp"
	}

	const chars = "0123456789abcdefghijklmnopqrstuvwxyz_"

	data := strconv.Itoa(u.Uid()) + u.Name() + chars
	hash := sha256.Sum256([]byte(data))

	buf := make([]byte, 33)

	for i, b := range hash {
		buf[i+1] = chars[b%byte(len(chars))]
	}

	buf[0] = buf[1]
	buf[1] = buf[2]
	buf[2] = '/'

	dir := "/var/folders/" + string(buf) + "/T/"

	return dir
}

// tempDirUserDarwin returns the default directory to use for temporary files with for a specific Linux user.
func tempDirUserLinux() string {
	return "/tmp"
}

// TempDirUserWindows returns the default directory to use for temporary files with for a specific Windows user.
func tempDirUserWindows(userName string) string {
	return `\Users\` + userName + `\AppData\Local\Temp`
}
