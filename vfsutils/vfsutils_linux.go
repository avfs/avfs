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

// +build linux

package vfsutils

import (
	"os"
	"sync"
	"syscall"

	"github.com/avfs/avfs"
)

// UMaskType is the file mode creation mask.
// it must be set to be read, so it must be protected with a mutex.
type UMaskType struct {
	once sync.Once
	mu   sync.RWMutex
	mask os.FileMode
}

// Get returns the file mode creation mask.
func (um *UMaskType) Get() os.FileMode {
	um.once.Do(func() {
		um.Set(0)
	})

	um.mu.RLock()
	u := um.mask
	um.mu.RUnlock()

	return u
}

// Set sets the file mode creation mask.
// umask must be set to 0 using umask(2) system call to be read,
// so its value is cached and protected by a mutex.
func (um *UMaskType) Set(mask os.FileMode) {
	um.mu.Lock()

	u := syscall.Umask(int(mask))

	if mask == 0 {
		syscall.Umask(u)
		um.mask = os.FileMode(u)
	} else {
		um.mask = mask
	}

	um.mu.Unlock()
}

// AsStatT converts a value as an avfs.StatT.
func AsStatT(value interface{}) *avfs.StatT {
	switch s := value.(type) {
	case *avfs.StatT:
		return s
	case *syscall.Stat_t:
		return &avfs.StatT{Uid: s.Uid, Gid: s.Gid}
	default:
		return &avfs.StatT{Uid: 0, Gid: 0}
	}
}
