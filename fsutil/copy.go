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
	"hash"
	"io"
	"sync"

	"github.com/avfs/avfs"
)

// bufSize is the size of each buffer used to copy files.
const bufSize = 32 * 1024

// bufPool is the buffer pool used to copy files.
var bufPool = &sync.Pool{New: func() interface{} { //nolint:gochecknoglobals
	buf := make([]byte, bufSize)

	return &buf
}}

// copyBufPool copies a source reader to a writer using a buffer from the buffer pool.
func copyBufPool(dst io.Writer, src io.Reader) (written int64, err error) { //nolint:unparam
	buf := bufPool.Get().(*[]byte) //nolint:errcheck
	defer bufPool.Put(buf)

	written, err = io.CopyBuffer(dst, src, *buf)

	return
}

// CopyFile copies a file between file systems and returns the hash sum of the source file.
func CopyFile(dstFs, srcFs avfs.Fs, dstPath, srcPath string, hasher hash.Hash) (sum []byte, err error) {
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

	if hasher == nil {
		return nil, nil
	}

	return hasher.Sum(nil), nil
}

// HashFile hashes a file and returns the hash sum.
func HashFile(fs avfs.Fs, name string, hasher hash.Hash) (sum []byte, err error) {
	f, err := fs.Open(name)
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
