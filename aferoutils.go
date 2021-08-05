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

package avfs

import "fmt"

// DirExists checks if a path exists and is a directory.
func DirExists(vfs VFS, path string) (bool, error) {
	fi, err := vfs.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}

	if vfs.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// Exists Check if a file or directory exists.
func Exists(vfs VFS, path string) (bool, error) {
	_, err := vfs.Stat(path)
	if err == nil {
		return true, nil
	}

	if vfs.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// IsDir checks if a given path is a directory.
func IsDir(vfs VFS, path string) (bool, error) {
	fi, err := vfs.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}

// IsEmpty checks if a given file or directory is empty.
func IsEmpty(vfs VFS, path string) (bool, error) {
	if b, _ := Exists(vfs, path); !b {
		return false, fmt.Errorf("%q path does not exist", path)
	}

	fi, err := vfs.Stat(path)
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		f, err := vfs.Open(path)
		if err != nil {
			return false, err
		}

		defer f.Close()

		list, err := f.ReadDir(-1)
		if err != nil {
			return false, err
		}

		return len(list) == 0, nil
	}

	return fi.Size() == 0, nil
}
