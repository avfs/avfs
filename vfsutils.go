//
//  Copyright 2021 The AVFS authors
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
	"hash"
	"io"
	"io/fs"
	"strings"
	"sync"
)

// bufSize is the size of each buffer used to copy files.
const bufSize = 32 * 1024

// bufPool is the buffer pool used to copy files.
var bufPool = &sync.Pool{New: func() interface{} { //nolint:gochecknoglobals // BufPool must be global.
	buf := make([]byte, bufSize)

	return &buf
}}

// copyBufPool copies a source reader to a writer using a buffer from the buffer pool.
func copyBufPool(dst io.Writer, src io.Reader) (written int64, err error) { //nolint:unparam // written is never used.
	buf := bufPool.Get().(*[]byte) //nolint:errcheck // Get() always returns a pointer to a byte slice.
	defer bufPool.Put(buf)

	written, err = io.CopyBuffer(dst, src, *buf)

	return
}

// CopyFile copies a file between file systems and returns the hash sum of the source file.
func CopyFile(dstFs, srcFs VFS, dstPath, srcPath string, hasher hash.Hash) (sum []byte, err error) {
	src, err := srcFs.Open(srcPath)
	if err != nil {
		return nil, err
	}

	defer src.Close()

	dst, err := dstFs.Create(dstPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		cerr := dst.Close()
		if cerr == nil {
			err = cerr
		}
	}()

	var out io.Writer

	if hasher == nil {
		out = dst
	} else {
		hasher.Reset()
		out = io.MultiWriter(dst, hasher)
	}

	_, err = copyBufPool(out, src)
	if err != nil {
		return nil, err
	}

	err = dst.Sync()
	if err != nil {
		return nil, err
	}

	info, err := srcFs.Stat(srcPath)
	if err != nil {
		return nil, err
	}

	err = dstFs.Chmod(dstPath, info.Mode())
	if err != nil {
		return nil, err
	}

	if hasher == nil {
		return nil, nil
	}

	return hasher.Sum(nil), nil
}

// HashFile hashes a file and returns the hash sum.
func HashFile(vfs VFS, name string, hasher hash.Hash) (sum []byte, err error) {
	f, err := vfs.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	hasher.Reset()

	_, err = copyBufPool(hasher, f)
	if err != nil {
		return nil, err
	}

	sum = hasher.Sum(nil)

	return sum, nil
}

// BaseDir is a directory always present in a file system.
type BaseDir struct {
	Path string
	Perm fs.FileMode
}

// BaseDirs are the base directories present in a file system.
var BaseDirs = []BaseDir{ //nolint:gochecknoglobals // Used by CreateBaseDirs and TestCreateBaseDirs.
	{Path: HomeDir, Perm: 0o755},
	{Path: RootDir, Perm: 0o700},
	{Path: TmpDir, Perm: 0o777},
}

// CreateBaseDirs creates base directories on a file system.
func CreateBaseDirs(vfs VFS, basePath string) error {
	for _, dir := range BaseDirs {
		path := vfs.Join(basePath, dir.Path)

		err := vfs.Mkdir(path, dir.Perm)
		if err != nil {
			return err
		}

		err = vfs.Chmod(path, dir.Perm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateHomeDir creates the home directory of a user.
func CreateHomeDir(vfs VFS, u UserReader) (UserReader, error) {
	userDir := vfs.Join(HomeDir, u.Name())

	err := vfs.Mkdir(userDir, HomeDirPerm)
	if err != nil {
		return nil, err
	}

	err = vfs.Chown(userDir, u.Uid(), u.Gid())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// SegmentPath segments string key paths by separator (using avfs.PathSeparator).
// For example with path = "/a/b/c" it will return in successive calls :
//
// "a", "/b/c"
// "b", "/c"
// "c", ""
//
// 	for start, end, isLast := 1, 0, len(path) <= 1; !isLast; start = end + 1 {
//		end, isLast = avfs.SegmentPath(path, start)
//		fmt.Println(path[start:end], path[end:])
//	}
//
func SegmentPath(vfs VFS, path string, start int) (end int, isLast bool) {
	pos := strings.IndexRune(path[start:], rune(vfs.PathSeparator()))
	if pos != -1 {
		return start + pos, false
	}

	return len(path), true
}
