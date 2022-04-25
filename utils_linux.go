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

//go:build linux

package avfs

import (
	"io/fs"
	"sync"
	"sync/atomic"
	"syscall"
)

// ShortPathName Retrieves the short path form of the specified path (Windows only).
func ShortPathName(path string) string {
	return path
}

var (
	// umask is the file mode creation mask.
	umask fs.FileMode //nolint:gochecknoglobals // Used by UMask and SetUMask.

	// umLock lock access to the umask.
	umLock sync.RWMutex //nolint:gochecknoglobals // Used by UMask and SetUMask.
)

func init() { //nolint:gochecknoinits // To initialize umask.
	umLock.Lock()

	m := syscall.Umask(0) // read mask.
	syscall.Umask(m)      // restore mask after read.
	umask = fs.FileMode(m)

	umLock.Unlock()
}

// SetUMask sets the file mode creation mask.
// Umask must be set to 0 using umask(2) system call to be read,
// so its value is cached and protected by a mutex.
func SetUMask(mask fs.FileMode) {
	umLock.Lock()

	m := int(mask) & 0o777
	_ = syscall.Umask(m)

	umask = fs.FileMode(m)

	umLock.Unlock()
}

// UMask returns the file mode creation mask.
func UMask() fs.FileMode {
	um := atomic.LoadUint32((*uint32)(&umask))

	return fs.FileMode(um)
}
