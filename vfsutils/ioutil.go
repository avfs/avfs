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
	"io/fs"
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
