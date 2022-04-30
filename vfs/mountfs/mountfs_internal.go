//
//  Copyright 2022 The AVFS authors
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

package mountfs

import (
	"io/fs"
	"os"
)

// pathToMount resolves a path to a mount point nmt and a path of the mounted file system.
func (vfs *MountFS) pathToMount(path string) (mnt *mount, vfsPath string) {
	absPath, _ := vfs.Abs(path)

	pi := vfs.utils.NewPathIterator(absPath)
	lm := vfs.rootMnt
	lp := path

	for pi.Next() {
		mp := pi.LeftPart()

		m, ok := vfs.mounts[mp]
		if ok {
			lm = m
			lp = pi.Right()
		}
	}

	return lm, lp
}

// toAbsPath
func (mnt *mount) toAbsPath(path string) string {
	return mnt.vfs.Join(mnt.mntPath, mnt.basePath, path)
}

// restoreError restore paths in errors if necessary.
func (mnt *mount) restoreError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *fs.PathError:
		return &fs.PathError{
			Op:   e.Op,
			Path: mnt.toAbsPath(e.Path),
			Err:  e.Err,
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: mnt.toAbsPath(e.Old),
			New: mnt.toAbsPath(e.New),
			Err: e.Err,
		}
	default:
		return err
	}
}
