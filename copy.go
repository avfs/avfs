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
	"sync"
)

var copyPool = newCopyPool() //nolint:gochecknoglobals // copyPool is the buffer pool used to copy files.

// newCopyPool initialize the copy buffer pool.
func newCopyPool() *sync.Pool {
	const bufSize = 32 * 1024

	pool := &sync.Pool{New: func() interface{} {
		buf := make([]byte, bufSize)

		return &buf
	}}

	return pool
}

// CopyFile copies a file between file systems and returns an error if any.
func CopyFile(dstFs, srcFs VFS, dstPath, srcPath string) error {
	_, err := CopyFileHash(dstFs, srcFs, dstPath, srcPath, nil)

	return err
}

// CopyFileHash copies a file between file systems and returns the hash sum of the source file.
func CopyFileHash(dstFs, srcFs VFS, dstPath, srcPath string, hasher hash.Hash) (sum []byte, err error) {
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

// copyBufPool copies a source reader to a writer using a buffer from the buffer pool.
func copyBufPool(dst io.Writer, src io.Reader) (written int64, err error) { //nolint:unparam // written is never used.
	buf := copyPool.Get().(*[]byte) //nolint:errcheck,forcetypeassert // Get() always returns a pointer to a byte slice.
	defer copyPool.Put(buf)

	written, err = io.CopyBuffer(dst, src, *buf)

	return
}
