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

package vfsutils

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"sort"

	"github.com/avfs/avfs"
)

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func ReadDir(vfs avfs.VFS, dirname string) ([]fs.FileInfo, error) {
	f, err := vfs.Open(dirname)
	if err != nil {
		return nil, err
	}

	list, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })

	return list, nil
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func ReadFile(vfs avfs.VFS, filename string) ([]byte, error) {
	f, err := vfs.Open(filename)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return ioutil.ReadAll(f)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func WriteFile(vfs avfs.VFS, filename string, data []byte, perm fs.FileMode) error {
	f, err := vfs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}

	return err
}
