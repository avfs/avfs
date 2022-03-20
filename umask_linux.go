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

//go:build linux

package avfs

import (
	"io/fs"
	"syscall"
)

// Set sets the file mode creation mask.
// Umask must be set to 0 using umask(2) system call to be read,
// so its value is cached and protected by a mutex.
func (um *UMaskType) Set(mask fs.FileMode) {
	um.mu.Lock()

	m := syscall.Umask(int(mask))
	if mask != 0 {
		m = syscall.Umask(0)
	}

	um.mask = fs.FileMode(m)

	// restore mask after read.
	syscall.Umask(m)

	um.mu.Unlock()
}
