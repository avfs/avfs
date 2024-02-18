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
)

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func HomeDirUser(vfs VFSBase, u UserReader) string {
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
func HomeDir(vfs VFSBase) string {
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
func MkHomeDir(vfs VFSBase, u UserReader) (string, error) {
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
func MkSystemDirs(vfs VFSBase, dirs []DirInfo) error {
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

// SystemDirs returns an array of system directories always present in the file system.
func SystemDirs(vfs VFSBase, basePath string) []DirInfo {
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

// TempDirUser returns the default directory to use for temporary files for a specific user.
func TempDirUser(vfs VFSBase, username string) string {
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

// VolumeNameLen returns length of the leading volume name on Windows.
// It returns 0 elsewhere.
func VolumeNameLen[T VFSBase](vfs T, path string) int {
	if vfs.OSType() != OsWindows {
		return 0
	}

	if len(path) < 2 {
		return 0
	}

	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return 2
	}

	// is it UNC? https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if l := len(path); l >= 5 && isSlash(path[0]) && isSlash(path[1]) &&
		!isSlash(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isSlash(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isSlash(path[n]) {
					if path[n] == '.' {
						break
					}

					for ; n < l; n++ {
						if isSlash(path[n]) {
							break
						}
					}

					return n
				}

				break
			}
		}
	}

	return 0
}
