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

package fsutil

import (
	"errors"
	"os"
	"strings"

	"github.com/avfs/avfs"
)

// UMask is the global variable containing the file mode creation mask.
var UMask UMaskType //nolint:gochecknoglobals

// CheckPermission returns true is a user 'u' has 'want' permission on a file or directory represented by 'info'.
func CheckPermission(info os.FileInfo, want avfs.WantMode, u avfs.UserReader) bool {
	if u.IsRoot() {
		return true
	}

	mode := info.Mode()
	sys := info.Sys()
	statT := AsStatT(sys)
	uid, gid := int(statT.Uid), int(statT.Gid)

	switch {
	case uid == u.Uid():
		mode >>= 6
	case gid == u.Gid():
		mode >>= 3
	}

	want &= avfs.WantRWX

	return avfs.WantMode(mode)&want == want
}

// CreateBaseDirs creates base directories on a file system.
func CreateBaseDirs(fs avfs.Fs) error {
	dirs := []struct {
		path string
		perm os.FileMode
	}{
		{path: avfs.HomeDir, perm: 0o755},
		{path: avfs.RootDir, perm: 0o700},
		{path: avfs.TmpDir, perm: 0o1777},
	}

	for _, dir := range dirs {
		err := fs.Mkdir(dir.path, dir.perm)
		if err != nil {
			return err
		}

		err = fs.Chmod(dir.path, dir.perm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateHomeDir creates the home directory of a user.
func CreateHomeDir(fs avfs.Fs, u avfs.UserReader) (avfs.UserReader, error) {
	userDir := fs.Join(avfs.HomeDir, u.Name())

	err := fs.Mkdir(userDir, avfs.HomeDirPerm)
	if err != nil {
		return nil, err
	}

	err = fs.Chown(userDir, u.Uid(), u.Gid())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func IsExist(err error) bool {
	return errors.Is(err, avfs.ErrFileExists)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func IsNotExist(err error) bool {
	return errors.Is(err, avfs.ErrNoSuchFileOrDir)
}

// SegmentPath segments string key paths by separator (using avfs.PathSeparator).
// For example with path = "/a/b/c" it will return in successive calls :
//
// "a", "/b/c"
// "b", "/c"
// "c", ""
//
// 	for start, end, isLast := 1, 0, len(path) <= 1; !isLast; start = end + 1 {
//		end, isLast = fsutil.SegmentPath(path, start)
//		fmt.Println(path[start:end], path[end:])
//	}
//
func SegmentPath(path string, start int) (end int, isLast bool) {
	pos := strings.IndexRune(path[start:], avfs.PathSeparator)
	if pos != -1 {
		return start + pos, false
	}

	return len(path), true
}
