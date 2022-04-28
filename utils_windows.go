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

//go:build windows

package avfs

import (
	"errors"
	"io/fs"
	"os"
	"sync/atomic"
	"syscall"
)

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (ut *Utils) IsExist(err error) bool {
	err = errors.Unwrap(err)
	switch e := err.(type) {
	case syscall.Errno:
		return os.IsExist(err)
	case LinuxError:
		return e == ErrFileExists
	case WindowsError:
		return e == ErrWinFileExists
	default:
		return false
	}
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (ut *Utils) IsNotExist(err error) bool {
	err = errors.Unwrap(err)
	switch e := err.(type) {
	case syscall.Errno:
		return os.IsNotExist(err)
	case LinuxError:
		return e == ErrNoSuchFileOrDir
	case WindowsError:
		return e == ErrWinPathNotFound || e == ErrWinFileNotFound
	default:
		return false
	}
}

// ShortPathName Retrieves the short path form of the specified path (Windows only).
func ShortPathName(path string) string {
	p, err := syscall.UTF16FromString(path)
	if err != nil {
		return path
	}

	b := p // GetShortPathName says we can reuse buffer
	n := uint32(len(b))

	for {
		n, err = syscall.GetShortPathName(&p[0], &b[0], n)
		if err != nil {
			return path
		}

		if n <= uint32(len(b)) {
			return syscall.UTF16ToString(b[:n])
		}

		b = make([]uint16, n)
	}
}

// umask is the file mode creation mask.
var umask fs.FileMode = 0o111 //nolint:gochecknoglobals // Used by UMask and SetUMask.

// SetUMask sets the file mode creation mask.
func SetUMask(mask fs.FileMode) {
	m := uint32(mask & fs.ModePerm)
	atomic.StoreUint32((*uint32)(&umask), m)
}

// UMask returns the file mode creation mask.
func UMask() fs.FileMode {
	um := atomic.LoadUint32((*uint32)(&umask))

	return fs.FileMode(um)
}
