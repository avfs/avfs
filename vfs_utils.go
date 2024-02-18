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
	"io/fs"
	"path/filepath"
)

// FromUnixPath returns valid path for Unix or Windows from a unix path.
// For Windows systems, absolute paths are prefixed with the default volume
// and relative paths are preserved.
func FromUnixPath[T VFSBase](vfs T, path string) string {
	if vfs.OSType() != OsWindows {
		return path
	}

	if path[0] != '/' {
		return filepath.FromSlash(path)
	}

	return filepath.Join(DefaultVolume, filepath.FromSlash(path))
}

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func HomeDirUser[T VFSBase](vfs T, u UserReader) string {
	name := u.Name()
	if vfs.OSType() == OsWindows {
		return vfs.Join(HomeDir(vfs), name)
	}

	if name == AdminUserName(vfs.OSType()) {
		return "/root"
	}

	return vfs.Join(HomeDir(vfs), name)
}

// HomeDir returns the home directory of the file system.
func HomeDir[T VFSBase](vfs T) string {
	switch vfs.OSType() {
	case OsWindows:
		return DefaultVolume + `\Users`
	default:
		return "/home"
	}
}

// HomeDirPerm return the default permission for home directories.
func HomeDirPerm() fs.FileMode {
	return 0o700
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

// MkHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func MkHomeDir[T VFSBase](vfs T, u UserReader) (string, error) {
	userDir := HomeDirUser(vfs, u)

	err := vfs.Mkdir(userDir, HomeDirPerm())
	if err != nil {
		return "", err
	}

	switch vfs.OSType() {
	case OsWindows:
		err = vfs.MkdirAll(TempDirUser(vfs, u.Name()), DefaultDirPerm)
	default:
		err = vfs.Chown(userDir, u.Uid(), u.Gid())
	}

	if err != nil {
		return "", err
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

		if vfs.OSType() != OsWindows {
			err = vfs.Chmod(dir.Path, dir.Perm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SplitAbs splits an absolute path immediately preceding the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, splitPath returns an empty dir
// and file set to path.
// The returned values have the property that path = dir + PathSeparator + file.
func SplitAbs[T VFSBase](vfs T, path string) (dir, file string) {
	l := VolumeNameLen(vfs, path)

	i := len(path) - 1
	for i >= l && !vfs.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i], path[i+1:]
}

// SystemDirs returns an array of system directories always present in the file system.
func SystemDirs[T VFSBase](vfs T, basePath string) []DirInfo {
	const volumeNameLen = 2

	switch vfs.OSType() {
	case OsWindows:
		return []DirInfo{
			{Path: vfs.Join(basePath, HomeDir(vfs)[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: vfs.Join(basePath, TempDirUser(vfs, AdminUserName(vfs.OSType()))[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: vfs.Join(basePath, TempDirUser(vfs, DefaultName)[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: vfs.Join(basePath, `\Windows`), Perm: DefaultDirPerm},
		}
	default:
		return []DirInfo{
			{Path: vfs.Join(basePath, HomeDir(vfs)), Perm: HomeDirPerm()},
			{Path: vfs.Join(basePath, "/root"), Perm: 0o700},
			{Path: vfs.Join(basePath, "/tmp"), Perm: 0o777},
		}
	}
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
func TempDir[T VFSBase](vfs T) string {
	return TempDirUser(vfs, vfs.User().Name())
}

// TempDirUser returns the default directory to use for temporary files for a specific user.
func TempDirUser[T VFSBase](vfs T, username string) string {
	if vfs.OSType() != OsWindows {
		return "/tmp"
	}

	dir := vfs.Join(DefaultVolume, `\Users\`, username, `\AppData\Local\Temp`)

	return dir
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func VolumeName[T VFSBase](vfs T, path string) string {
	return vfs.FromSlash(path[:VolumeNameLen(vfs, path)])
}
