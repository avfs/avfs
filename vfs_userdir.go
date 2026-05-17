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

// VFSUserDir is the interface that manages user directories.
type VFSUserDir interface {
	// Abs returns an absolute representation of path.
	// If the path is not absolute it will be joined with the current
	// working directory to turn it into an absolute path. The absolute
	// path name for a given file is not guaranteed to be unique.
	// Abs calls [Clean] on the result.
	Abs(path string) (string, error)

	// Getwd returns an absolute path name corresponding to the
	// current directory. If the current directory can be
	// reached via multiple paths (due to symbolic links),
	// Getwd may return any one of them.
	//
	// On Unix platforms, if the environment variable PWD
	// provides an absolute name, and it is a name of the
	// current directory, it is returned.
	Getwd() (dir string, err error)

	// SetUser sets the current user.
	// If the user can't be changed an error is returned.
	SetUser(user UserReader) error

	// SetUserByName sets the current user by name.
	// If the user is not found, the returned error is of type UnknownUserError.
	SetUserByName(name string) error

	// TempDir returns the default directory to use for temporary files.
	//
	// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
	// On Windows, it uses GetTempPath, returning the first non-empty
	// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
	// On Plan 9, it returns /tmp.
	//
	// The directory is neither guaranteed to exist nor have accessible
	// permissions.
	TempDir() string

	// User returns the current user.
	User() UserReader
}

// VFSUserDirFn provides functionalities to manage directories and current user in a virtual file system.
type VFSUserDirFn struct {
	user      UserReader // user is the current user of the file system.
	curDir    string     // curDir is the current directory.
	tempDir   string     // tempDir is the temporary directory.
	IdmFn                //
	VFSPathFn            //
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls [Clean] on the result.
func (vuf *VFSUserDirFn) Abs(path string) (string, error) {
	if vuf.IsAbs(path) {
		return vuf.Clean(path), nil
	}

	return vuf.Join(vuf.curDir, path), nil
}

// CurDir returns the current directory.
func (vuf *VFSUserDirFn) CurDir() string {
	return vuf.curDir
}

// Getwd returns an absolute path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
//
// On Unix platforms, if the environment variable PWD
// provides an absolute name, and it is a name of the
// current directory, it is returned.
func (vuf *VFSUserDirFn) Getwd() (dir string, err error) {
	return vuf.curDir, nil
}

// SetUser sets the current user.
func (vuf *VFSUserDirFn) SetUser(user UserReader) error {
	if user == nil {
		user = vuf.Idm().AdminUser()
	}

	vuf.user = user
	vuf.curDir = homeDirUser(vuf.osType, vuf.user)
	vuf.tempDir = tempDirUser(vuf.osType, vuf.user)

	return nil
}

// SetCurDir sets the current directory.
func (vuf *VFSUserDirFn) SetCurDir(curDir string) error {
	vuf.curDir = curDir

	return nil
}

// SetUserByName sets the current user by name.
// If the user is not found, the returned error is of type UnknownUserError.
func (vuf *VFSUserDirFn) SetUserByName(userName string) error {
	idm := vuf.idm

	u, err := idm.LookupUser(userName)
	if err != nil {
		return err
	}

	return vuf.SetUser(u)
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
func (vuf *VFSUserDirFn) TempDir() string {
	var dir string

	u := vuf.user

	switch vuf.osType {
	case OsWindows:
		dir = tempDirUserWindows(u.Name())
	case OsDarwin:
		dir = tempDirUserDarwin(u)
	default:
		dir = tempDirUserLinux()
	}

	return dir
}

// User returns the current user.
func (vuf *VFSUserDirFn) User() UserReader {
	return vuf.user
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
	if basePath == "" && vfs.OSType() == OsWindows {
		basePath = DefaultVolume
	}

	dir := vfs.Join(basePath, homeDirUser(vfs.OSType(), u))

	return dir
}

func homeDirUser(ost OSType, u UserReader) string {
	var dir string

	switch ost {
	case OsWindows:
		dir = homeDir(OsWindows) + "/" + u.Name()
	case OsDarwin:
		dir = homeDir(OsDarwin) + "/" + u.Name()
	default:
		if u.Name() == AdminUserName(OsLinux) {
			dir = "/root"
		} else {
			dir = homeDir(OsLinux) + "/" + u.Name()
		}
	}

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

	switch vfs.OSType() {
	case OsWindows:
		err := vfs.MkdirAll(userDir, DefaultDirPerm)
		if err != nil {
			return userDir, err
		}

	default:
		err := vfs.Mkdir(userDir, HomeDirPerm())
		if err != nil {
			return userDir, err
		}

		err = vfs.Chown(userDir, u.Uid(), u.Gid())
		if err != nil {
			return userDir, err
		}
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
		dis[i].Path = vfs.Join(basePath, di.Path)
	}

	return dis
}

// tempDirUser returns the temporary directory for a specific user on a specific operating system type.
func tempDirUser(ost OSType, u UserReader) string {
	var dir string

	switch ost {
	case OsWindows:
		dir = tempDirUserWindows(u.Name())
	case OsDarwin:
		dir = tempDirUserDarwin(u)
	default:
		dir = tempDirUserLinux()
	}

	return dir
}

// tempDirUserDarwin returns the temporary directory for a specific macOS user.
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

// tempDirUserLinux returns the temporary directory for a specific Linux user.
func tempDirUserLinux() string {
	return "/tmp"
}

// TempDirUserWindows returns the temporary directory for a specific Windows user.
func tempDirUserWindows(userName string) string {
	return `\Users\` + userName + `\AppData\Local\Temp`
}
