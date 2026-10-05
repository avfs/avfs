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
	"strconv"
	"sync/atomic"
)

// VFSUserDir is the interface that provides the identity and the directories
// associated with the current user of a file system.
//
// The identity (user, home and temporary directories) and the emulated OS type
// are immutable: they are set once, when the file system is built. A file
// system acting as another user, or emulating another OS, is obtained by
// cloning (see [Cloner]), so that the identity a file system acts as can never
// change while an operation is in flight. The current working directory is the
// only mutable part, as it is for [os.Chdir].
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

// UserDirMixin is an embeddable default implementation of the VFSUserDir interface.
//
// Init must be called once during construction. The user, home and temporary
// directories are then read-only; only the current directory changes, through
// SetCurDir (used by Chdir).
type UserDirMixin struct {
	user      UserReader             // user is the current user of the file system.
	homeDir   string                 // homeDir is the home directory of the current user.
	tempDir   string                 // tempDir is the temporary directory.
	curDir    atomic.Pointer[string] // curDir is the current directory.
	IdmMixin                         // IdmMixin is an embeddable default implementation of the IdmProvider interface.
	PathMixin                        // PathMixin is an embeddable default implementation of the VFSPath interface.
}

// Init initializes the identity and the user directories of the file system.
//
// It must be called once, during construction, before any other method: it sets
// the emulated OS type (ost), the identity manager (idm), the current user
// (user, the identity manager administrator if nil) and the current directory,
// which is the home directory of user.
//
// The home and temporary directories are derived from ost and user, so a file
// system acting as another user, or emulating another OS, must be built with —
// or cloned with — that identity. The home directory must exist in the file
// system: it is the directory a clone starts in.
func (udmx *UserDirMixin) Init(ost OSType, idm IdmMgr, user UserReader) error {
	err := udmx.InitOSType(ost)
	if err != nil {
		return err
	}

	err = udmx.InitIdm(idm)
	if err != nil {
		return err
	}

	if user == nil {
		user = udmx.idm.AdminUser()
	}

	udmx.user = user
	udmx.homeDir = homeDirUser(udmx.OSType(), user)
	udmx.tempDir = tempDirUser(udmx.OSType(), user)
	udmx.curDir.Store(&udmx.homeDir)

	return nil
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls [Clean] on the result.
func (udmx *UserDirMixin) Abs(path string) (string, error) {
	if udmx.IsAbs(path) {
		return udmx.Clean(path), nil
	}

	return udmx.Join(udmx.CurDir(), path), nil
}

// CurDir returns the current directory.
func (udmx *UserDirMixin) CurDir() string {
	curDir := udmx.curDir.Load()
	if curDir == nil {
		return ""
	}

	return *curDir
}

// Getwd returns an absolute path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
//
// On Unix platforms, if the environment variable PWD
// provides an absolute name, and it is a name of the
// current directory, it is returned.
func (udmx *UserDirMixin) Getwd() (dir string, err error) {
	return udmx.CurDir(), nil
}

// SetCurDir sets the current directory.
//
// Unlike Chdir, it resolves nothing and checks no permission: the caller is
// responsible for having authorized the directory beforehand.
func (udmx *UserDirMixin) SetCurDir(curDir string) error {
	udmx.curDir.Store(&curDir)

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
func (udmx *UserDirMixin) TempDir() string {
	return udmx.tempDir
}

// User returns the current user.
func (udmx *UserDirMixin) User() UserReader {
	return udmx.user
}

// UserHomeDir returns the current user's home directory.
//
// On Unix, including macOS, it returns the $HOME environment variable.
// On Windows, it returns %USERPROFILE%.
// On Plan 9, it returns the $home environment variable.
//
// If the expected variable is not set in the environment, UserHomeDir
// returns either a platform-specific default value or a non-nil error.
func (udmx *UserDirMixin) UserHomeDir() (string, error) {
	dir := udmx.homeDir

	return dir, nil
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
func homeDirUser(ost OSType, u UserReader) string {
	var dir string

	switch ost {
	case OsWindows:
		dir = `\Users\` + u.Name()
	case OsDarwin:
		dir = "/Users/" + u.Name()
	default:
		if u.IsAdmin() {
			dir = "/root"
		} else {
			dir = "/home/" + u.Name()
		}
	}

	return dir
}

// MkDirs creates a set of directories with specified permissions, ownership, and base path.
func MkDirs[T VFSBase](vfs T, dirs []DirInfo, basePath string) error {
	if vfs.OSType() == OsWindows && basePath == "" {
		basePath = DefaultVolume
	}

	for _, dir := range dirs {
		path := vfs.Join(basePath, dir.Path)

		_, err := vfs.Stat(path)
		if err == nil {
			continue
		}

		err = vfs.MkdirAll(path, dir.Perm)
		if err != nil {
			return err
		}

		switch vfs.OSType() {
		case OsWindows:

		default:
			err = vfs.Chmod(path, dir.Perm)
			if err != nil {
				return err
			}

			err = vfs.Chown(path, dir.Uid, dir.Gid)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SystemDirs returns an array of system directories always present in the file system.
func SystemDirs[T VFSBase](vfs T) []DirInfo {
	var dis []DirInfo

	admin := vfs.Idm().AdminUser()

	switch vfs.OSType() {
	case OsWindows:
		dis = []DirInfo{
			{Path: homeDir(OsWindows), Perm: DefaultDirPerm, Uid: admin.Uid(), Gid: admin.Gid()},
			{Path: tempDirUserWindows(AdminUserName(OsWindows)), Perm: DefaultDirPerm, Uid: admin.Uid(), Gid: admin.Gid()},
		}

	case OsDarwin:
		dis = []DirInfo{
			{Path: homeDir(OsDarwin), Perm: 0o755, Uid: admin.Uid(), Gid: admin.Gid()},
		}

	default:
		dis = []DirInfo{
			{Path: homeDir(OsLinux), Perm: 0o755, Uid: admin.Uid(), Gid: admin.Gid()},
			{Path: "/root", Perm: 0o700, Uid: admin.Uid(), Gid: admin.Gid()},
			{Path: tempDirUserLinux(), Perm: 0o777, Uid: admin.Uid(), Gid: admin.Gid()},
		}
	}

	return dis
}

// UserDirsInfo retrieves metadata for all user directories of the user u from
// the provided virtual file system (vfs). The home directory of u is always
// the first element of the returned slice.
func UserDirsInfo[T VFSBase](vfs T, u UserReader) []DirInfo {
	ost := vfs.OSType()

	dis := []DirInfo{
		{Path: homeDirUser(ost, u), Perm: 0o755, Uid: u.Uid(), Gid: u.Gid()},
	}

	switch ost {
	case OsWindows:
		dis = append(dis, DirInfo{Path: tempDirUserWindows(u.Name()), Perm: DefaultDirPerm, Uid: u.Uid(), Gid: u.Gid()})

	case OsDarwin:
		dis = append(dis, DirInfo{Path: tempDirUserDarwin(u), Perm: 0o777, Uid: u.Uid(), Gid: u.Gid()})

	default:
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
